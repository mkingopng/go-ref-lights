// connection_test.go
//go:build unit
// +build unit

package websocket

import (
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// ----------------- FAKE WSConn IMPLEMENTATION -----------------

// fakeConn implements the WSConn interface to simulate a WebSocket connection
// without real network usage. It tracks whether a ping has been sent.
type fakeConn struct {
	pingCaptured bool
}

func (fc *fakeConn) WriteMessage(messageType int, data []byte) error {
	if messageType == websocket.PingMessage {
		fc.pingCaptured = true
	}
	return nil
}

func (fc *fakeConn) SetWriteDeadline(t time.Time) error { return nil }
func (fc *fakeConn) ReadMessage() (int, []byte, error) {
	// Return a dummy message if needed. Some tests won't use it anyway.
	return websocket.TextMessage, []byte(`{"action":"dummy"}`), nil
}
func (fc *fakeConn) Close() error { return nil }
func (fc *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}
func (fc *fakeConn) SetReadLimit(limit int64)            {}
func (fc *fakeConn) SetReadDeadline(t time.Time) error   { return nil }
func (fc *fakeConn) SetPongHandler(h func(string) error) {}

// ----------------- TESTS -----------------

func TestWritePump_Ping(t *testing.T) {
	// Save original pingPeriod/pongWait
	origPing := pingPeriod
	origPong := pongWait

	// Shorten them for test
	pongWait = 50 * time.Millisecond
	pingPeriod = (pongWait * 9) / 10 // e.g. 45ms

	// Restore after test
	defer func() {
		pingPeriod = origPing
		pongWait = origPong
	}()

	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 10),
		meetName: "UnitTestMeet",
	}

	done := make(chan struct{})
	go func() {
		conn.writePump()
		close(done)
	}()

	// Wait enough time to see a ping
	time.Sleep(2 * pingPeriod)

	// Close the send channel to stop the writePump
	close(conn.send)
	<-done

	assert.True(t, fc.pingCaptured, "Expected a ping to be sent at least once")
}

// Example of testing register/unregister if needed
func TestRegisterUnregisterConnection(t *testing.T) {
	InitTest()
	// Clear global connections
	connections = make(map[*Connection]bool)

	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 1),
		meetName: "TestMeet",
	}
	registerConnection(conn)
	assert.Equal(t, 1, len(connections), "Should have 1 connection")

	unregisterConnection(conn)
	assert.Equal(t, 0, len(connections), "Should have 0 after unregistering")
}

// ----------------- ADDITIONAL tests -----------------

//func TestRegisterAndUnregisterConnection(t *testing.T) {
//	InitTest()
//	connections = make(map[*Connection]bool) // clear global map
//
//	fc := &fakeConn{}
//	conn := &Connection{
//		conn:     fc,
//		send:     make(chan []byte, 1),
//		meetName: "UnitTestMeet",
//	}
//
//	registerConnection(conn)
//	assert.Equal(t, 1, len(connections), "Expected one connection to be registered")
//
//	unregisterConnection(conn)
//	assert.Equal(t, 0, len(connections), "Expected no connections after unregistering")
//}
//
//func TestHandleIncoming_RegisterRef(t *testing.T) {
//	InitTest()
//	fc := &fakeConn{}
//	conn := &Connection{
//		conn:     fc,
//		send:     make(chan []byte, 1),
//		meetName: "UnitTestMeet",
//		judgeID:  "",
//	}
//
//	// No-op broadcastRefereeHealth so it does not clutter the test
//	origBRH := broadcastRefereeHealth
//	broadcastRefereeHealth = func(meetName string) {}
//	defer func() { broadcastRefereeHealth = origBRH }()
//
//	msg := DecisionMessage{
//		Action:   "registerRef",
//		MeetName: "UnitTestMeet",
//		JudgeID:  "ref1",
//	}
//	handleIncoming(conn, msg)
//	assert.Equal(t, "ref1", conn.judgeID, "JudgeID should be set after 'registerRef'")
//}

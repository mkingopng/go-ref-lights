//go:build unit
// +build unit

// connection_unit_test.go

// Unit tests for connection.go. These tests use a fakeConn to simulate a WSConn
// so that we can test the connection business logic (registering, handling messages,
// processing decisions, and ensuring pings are sent) without doing any real network I/O.

package websocket

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// Fake WSConn implementation for unit tests

// fakeConn implements the WSConn interface. It provides no‐op implementations
// for methods except that it records when a ping is sent.
type fakeConn struct {
	// pingCaptured is set to true when WriteMessage is called with a PingMessage.
	pingCaptured bool
}

func (fc *fakeConn) WriteMessage(messageType int, data []byte) error {
	if messageType == websocket.PingMessage {
		fc.pingCaptured = true
	}
	// For unit testing, we simply do nothing.
	return nil
}

func (fc *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (fc *fakeConn) ReadMessage() (int, []byte, error) {
	// For unit tests that do not use ReadMessage, simply return a dummy value.
	return websocket.TextMessage, []byte(`{"action": "dummy"}`), nil
}

func (fc *fakeConn) Close() error {
	return nil
}

func (fc *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (fc *fakeConn) SetReadLimit(limit int64) {}

func (fc *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (fc *fakeConn) SetPongHandler(h func(string) error) {}

// Unit tests for connection.go business logic

// TestRegisterAndUnregisterConnection verifies that registerConnection and unregisterConnection
// correctly update the global "connections" map.
func TestRegisterAndUnregisterConnection(t *testing.T) {
	InitTest()
	// Clear the global connections map.
	connections = make(map[*Connection]bool)

	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 1),
		meetName: "UnitTestMeet",
		judgeID:  "",
	}

	registerConnection(conn)
	assert.Equal(t, 1, len(connections), "Expected one connection to be registered")

	unregisterConnection(conn)
	assert.Equal(t, 0, len(connections), "Expected no connections after unregistering")
}

// TestHandleIncoming_RegisterRef tests that when a "registerRef" message is received,
// the connection's judgeID is set. We override broadcastRefereeHealth to a no-op.
func TestHandleIncoming_RegisterRef(t *testing.T) {
	InitTest()
	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 1),
		meetName: "UnitTestMeet",
		judgeID:  "",
	}

	// Override broadcastRefereeHealth so it does nothing during this test.
	origBRH := broadcastRefereeHealth
	broadcastRefereeHealth = func(meetName string) {}
	defer func() { broadcastRefereeHealth = origBRH }()

	msg := DecisionMessage{
		Action:   "registerRef",
		MeetName: "UnitTestMeet",
		JudgeID:  "ref1",
	}
	handleIncoming(conn, msg)
	assert.Equal(t, "ref1", conn.judgeID, "judgeID should be set from registerRef action")
}

// TestProcessDecision tests that processDecision enqueues a broadcast message on the connection's send channel.
func TestProcessDecision(t *testing.T) {
	InitTest()
	// Clear global connections and register one connection.
	connections = make(map[*Connection]bool)
	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 10),
		meetName: "UnitTestMeet",
		judgeID:  "ref1",
	}
	registerConnection(conn)
	defer unregisterConnection(conn)

	decision := DecisionMessage{
		Action:   "submitDecision",
		MeetName: "UnitTestMeet",
		JudgeID:  "ref1",
		Decision: "white",
	}
	processDecision(conn, decision)

	// processDecision is expected to send a JSON message on the send channel.
	select {
	case msg := <-conn.send:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		// Our example processDecision broadcasts a "judgeSubmitted" message.
		assert.Equal(t, "judgeSubmitted", decoded["action"], "Expected action judgeSubmitted")
		assert.Equal(t, "ref1", decoded["judgeId"], "Expected judgeId to be ref1")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected a broadcast message, but none was received")
	}
}

// TestWritePump_Ping verifies that writePump sends a ping message periodically.
// We use our fakeConn's pingCaptured flag to detect that a ping was sent.
func TestWritePump_Ping(t *testing.T) {
	InitTest()
	fc := &fakeConn{}
	conn := &Connection{
		conn:     fc,
		send:     make(chan []byte, 10),
		meetName: "UnitTestMeet",
	}

	// Run writePump in a separate goroutine.
	done := make(chan struct{})
	go func() {
		conn.writePump()
		close(done)
	}()

	// Wait long enough for at least one ping cycle (pingPeriod is defined in connection.go).
	time.Sleep(pingPeriod + 50*time.Millisecond)
	// Stop writePump by closing the send channel.
	close(conn.send)
	<-done

	assert.True(t, fc.pingCaptured, "Expected writePump to send at least one ping")
}

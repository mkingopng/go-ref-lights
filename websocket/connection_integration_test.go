//go:build integration
// +build integration

// integration/connection_integration_test.go
package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// Helper function to start a test WebSocket server
func startTestServer(t *testing.T) (*httptest.Server, *websocket.Conn) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(w, r)
	}))

	wsURL := "ws" + server.URL[4:] + "?meetName=TestMeet"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err, "WebSocket connection should succeed")

	return server, conn
}

// `TestWritePump` should match expected response
func TestWritePump(t *testing.T) {
	server, conn := startTestServer(t)
	defer server.Close()
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Warning: WebSocket close error: %v", err)
		}
	}()

	testConn := &Connection{
		conn: conn,
		send: make(chan []byte, 100), // Increase buffer size
	}

	registerConnection(testConn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		testConn.writePump()
	}()

	time.Sleep(500 * time.Millisecond)

	// Send properly formatted test message
	testMessage := DecisionMessage{
		Action:   "submitDecision",
		JudgeID:  "left",
		Decision: "white",
		MeetName: "TestMeet",
	}
	messageBytes, _ := json.Marshal(testMessage)
	testConn.send <- messageBytes

	// Wait for the message to be read
	done := make(chan struct{})
	go func() {
		defer close(done)

		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Error reading message: %v", err)
			return
		}

		// Ensure message type is correct
		assert.Equal(t, websocket.TextMessage, msgType, "Expected TextMessage")

		// Match actual response from `handleIncoming()`
		expectedResponse := map[string]interface{}{
			"action":  "judgeSubmitted",
			"judgeId": "left",
		}

		var receivedResponse map[string]interface{}
		_ = json.Unmarshal(msg, &receivedResponse)

		assert.Equal(t, expectedResponse, receivedResponse, "Sent and received messages should match")
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message in TestWritePump")
	}

	wg.Wait()
	unregisterConnection(testConn)
	close(testConn.send)
	time.Sleep(500 * time.Millisecond)
}

// Ensure proper cleanup in `TestBroadcastRefereeHealth`
func TestBroadcastRefereeHealth(t *testing.T) {
	server, conn := startTestServer(t)
	defer server.Close()
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Warning: WebSocket close error: %v", err)
		}
	}()

	// Clear global connections before test
	connections = make(map[*Connection]bool)

	mockConn := &Connection{
		conn:     conn,
		send:     make(chan []byte, 1),
		meetName: "TestMeet",
		judgeID:  "left",
	}
	registerConnection(mockConn)

	assert.Equal(t, 1, len(connections), "Should register one connection")

	broadcastRefereeHealth("TestMeet")

	time.Sleep(100 * time.Millisecond)

	// Ensure proper cleanup
	unregisterConnection(mockConn)
	close(mockConn.send)

	assert.Equal(t, 0, len(connections), "Connection should be removed after test")
}

func TestProcessDecision(t *testing.T) {
	mockConn := &Connection{meetName: "TestMeet"}
	decision := DecisionMessage{
		Action:   "submitDecision",
		MeetName: "TestMeet",
		JudgeID:  "left",
		Decision: "white",
	}

	processDecision(mockConn, decision)
}

func TestBroadcastToMeet(t *testing.T) {
	server, conn := startTestServer(t)
	defer server.Close()
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			t.Logf("Warning: WebSocket close error: %v", err)
		}
	}(conn)

	mockConn := &Connection{
		conn:     conn,
		send:     make(chan []byte, 1),
		meetName: "TestMeet",
	}
	registerConnection(mockConn)

	testMessage := []byte(`{"action":"judgeSubmitted","judgeId":"left"}`)
	broadcastToMeet("TestMeet", testMessage)

	time.Sleep(100 * time.Millisecond)

	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err, "Should receive broadcast message")
	assert.JSONEq(t, string(testMessage), string(msg), "Broadcasted message should be received correctly")

	unregisterConnection(mockConn)
}

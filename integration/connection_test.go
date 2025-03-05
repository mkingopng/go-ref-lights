//go:build integration
// +build integration

// integration/connection_test.go
package integration

import (
	"encoding/json"
	websocket2 "go-ref-lights/websocket"
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
		websocket2.ServeWs(w, r)
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

	testConn := &websocket2.Connection{
		conn: conn,
		send: make(chan []byte, 100), // Increase buffer size
	}

	websocket2.registerConnection(testConn)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		testConn.writePump()
	}()

	time.Sleep(500 * time.Millisecond)

	// Send properly formatted test message
	testMessage := websocket2.DecisionMessage{
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
	websocket2.unregisterConnection(testConn)
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
	websocket2.connections = make(map[*websocket2.Connection]bool)

	mockConn := &websocket2.Connection{
		conn:     conn,
		send:     make(chan []byte, 1),
		meetName: "TestMeet",
		judgeID:  "left",
	}
	websocket2.registerConnection(mockConn)

	assert.Equal(t, 1, len(websocket2.connections), "Should register one connection")

	websocket2.broadcastRefereeHealth("TestMeet")

	time.Sleep(100 * time.Millisecond)

	// Ensure proper cleanup
	websocket2.unregisterConnection(mockConn)
	close(mockConn.send)

	assert.Equal(t, 0, len(websocket2.connections), "Connection should be removed after test")
}

func TestProcessDecision(t *testing.T) {
	mockConn := &websocket2.Connection{meetName: "TestMeet"}
	decision := websocket2.DecisionMessage{
		Action:   "submitDecision",
		MeetName: "TestMeet",
		JudgeID:  "left",
		Decision: "white",
	}

	websocket2.processDecision(mockConn, decision)
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

	mockConn := &websocket2.Connection{
		conn:     conn,
		send:     make(chan []byte, 1),
		meetName: "TestMeet",
	}
	websocket2.registerConnection(mockConn)

	testMessage := []byte(`{"action":"judgeSubmitted","judgeId":"left"}`)
	websocket2.broadcastToMeet("TestMeet", testMessage)

	time.Sleep(100 * time.Millisecond)

	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err, "Should receive broadcast message")
	assert.JSONEq(t, string(testMessage), string(msg), "Broadcasted message should be received correctly")

	websocket2.unregisterConnection(mockConn)
}

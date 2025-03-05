//go:build integration
// +build integration

package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastMessageDelivery(t *testing.T) {
	// Step 1: Set up a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(w, r)
	}))
	defer server.Close()

	// Build the WebSocket URL (adjust accordingly)
	wsURL := "ws" + server.URL[4:] + "?meetName=TestMeet"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Step 2: Register a test connection (with matching meetName)
	testConn := &Connection{
		conn:     conn,
		send:     make(chan []byte, 10),
		meetName: "TestMeet",
	}
	registerConnection(testConn) // Using the internal helper function

	// Step 3: Broadcast a test message
	testMessage := map[string]interface{}{
		"action":   "testBroadcast",
		"meetName": "TestMeet",
	}
	BroadcastMessage("TestMeet", testMessage)

	// Step 4: Read the message and assert it matches expectations
	// (Allow some time for message processing)
	time.Sleep(100 * time.Millisecond)
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)

	var received map[string]interface{}
	err = json.Unmarshal(msg, &received)
	assert.NoError(t, err)
	assert.Equal(t, "testBroadcast", received["action"])

	// Step 5: Cleanup
	unregisterConnection(testConn) // Using the internal helper function
	close(testConn.send)
}

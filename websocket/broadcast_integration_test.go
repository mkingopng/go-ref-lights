//go:build integration
// +build integration

// File: websocket/broadcast_integration_test.go
//
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
	// set up a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(w, r)
	}))
	defer server.Close()

	// build the WebSocket URL
	wsURL := "ws" + server.URL[4:] + "?meetName=TestMeet"
	conn, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// register a test connection (with matching meetName)
	testConn := &Connection{
		conn:     conn,
		send:     make(chan []byte, 10),
		meetName: "TestMeet",
	}
	registerConnection(testConn) // using the internal helper function

	// broadcast a test message
	testMessage := map[string]interface{}{
		"action":   "testBroadcast",
		"meetName": "TestMeet",
	}
	BroadcastMessage("TestMeet", testMessage)

	// read the message and assert it matches expectations
	time.Sleep(100 * time.Millisecond)
	_, msg, err := conn.ReadMessage()
	assert.NoError(t, err)

	var received map[string]interface{}
	err = json.Unmarshal(msg, &received)
	assert.NoError(t, err)
	assert.Equal(t, "testBroadcast", received["action"])

	// cleanup
	unregisterConnection(testConn) // using the internal helper function
	close(testConn.send)
}

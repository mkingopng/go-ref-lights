// file: websocket/handler_test.go
package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

// Test: WebSocket connection should be upgraded successfully
func TestServeWs_Success(t *testing.T) {
	// Create a test HTTP server that handles WebSocket requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a test WebSocket connection by adding Test-Mode header
		r.Header.Set("Test-Mode", "true")
		ServeWs(w, r)
	}))
	defer server.Close()

	// Convert "http://" to "ws://" for WebSocket connections
	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Ensure WebSocket upgrade was successful
	assert.NoError(t, err, "Expected WebSocket connection to succeed")

	// Ensure the connection is closed properly
	if conn != nil {
		conn.Close()
	}
}

// Test: WebSocket upgrade should fail with a non-WebSocket request
func TestServeWs_Failure(t *testing.T) {
	req, _ := http.NewRequest("GET", "/ws", nil) // No "Upgrade" header
	w := httptest.NewRecorder()

	ServeWs(w, req) // Ensure function exists

	// Expect `400 Bad Request` instead of `500`
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected failure when not upgrading to WebSocket")
}

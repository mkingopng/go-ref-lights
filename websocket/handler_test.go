// file: websocket/handler_test.go
package websocket_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	"go-ref-lights/websocket"
)

func TestWebSocketConnection(t *testing.T) {
	// Start the broadcast loop in a separate goroutine
	go websocket.HandleMessages()

	r := gin.Default()
	r.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// Create a test server
	server := httptest.NewServer(r)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + server.URL[len("http"):] + "/referee-updates"

	// Connect via Gorilla websocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.NoError(t, err)
	defer conn.Close()

	// Send a test message
	msg := `{"judgeId":"left","decision":"white"}`
	err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
	assert.NoError(t, err)

	// Optionally read broadcast messages back
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	_, resp, err := conn.ReadMessage()
	assert.NoError(t, err)

	// Check the server responded with "judgeSubmitted" or similar
	assert.Contains(t, string(resp), `"action":"judgeSubmitted"`)
}

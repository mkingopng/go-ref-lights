// Package websocket - websocket/connection.go

package websocket

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
	"net"
)

// -----------------------------------------------------------------------------
// Define an interface for the minimal WebSocket connection functionality.
// This allows dependency injection for tests.
type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	SetWriteDeadline(t time.Time) error
	ReadMessage() (int, []byte, error)
	Close() error
	RemoteAddr() net.Addr
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetPongHandler(h func(string) error)
}

// Connection represents a single WebSocket connection for one client.
type Connection struct {
	// Now using WSConn instead of a concrete *websocket.Conn.
	conn WSConn
	// Outbound messages are queued in this channel.
	send chan []byte
	// Each connection belongs to a specific meet.
	meetName string
	// The judge ID for this connection (set via a "registerRef" message).
	judgeID string
}

// Global map for active connections (replaces your old 'clients' and 'connectionMapping').
var connections = make(map[*Connection]bool)

// Configuration constants.
const (
	writeWait      = 10 * time.Second    // Time allowed to write a message.
	pongWait       = 60 * time.Second    // Time allowed to read the next pong.
	pingPeriod     = (pongWait * 9) / 10 // Send pings at this period.
	maxMessageSize = 2048                // Maximum message size from peer.
)

// Upgrader upgrades HTTP requests to WebSocket connections.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for now. Adjust for production if needed.
		return true
	},
}

// ServeWs upgrades the HTTP request to a WebSocket connection and starts the read and write pumps.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		logger.Error.Println("No meet selected; rejecting WebSocket connection")
		http.Error(w, "No meet selected", http.StatusBadRequest)
		return
	}

	logger.Info.Printf("[ServeWs] Upgrading to WS: remoteAddr=%v, meetName=%q", r.RemoteAddr, meetName)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error.Printf("[ServeWs] WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}

	// Create our Connection object.
	c := &Connection{
		conn:     wsConn,
		send:     make(chan []byte, 256),
		meetName: meetName,
		judgeID:  "", // Will be set when a "registerRef" message is received.
	}

	// Register this connection globally.
	registerConnection(c)

	// Start the readPump and writePump goroutines.
	go c.readPump()
	go c.writePump()
}

// readPump handles inbound messages from the client.
func (c *Connection) readPump() {
	defer func() {
		unregisterConnection(c)
		err := c.conn.Close()
		if err != nil {
			return
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			return err
		}
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			logger.Warn.Printf("[readPump] Read error from %v: %v", c.conn.RemoteAddr(), err)
			break
		}
		if messageType != websocket.TextMessage {
			logger.Debug.Printf("[readPump] Ignoring non-text messageType=%d", messageType)
			continue
		}

		var dm DecisionMessage
		if err := json.Unmarshal(message, &dm); err != nil {
			logger.Warn.Printf("[readPump] Invalid JSON from %v: %v", c.conn.RemoteAddr(), err)
			continue
		}
		handleIncoming(c, dm)
	}
}

// writePump handles outbound messages to the client, including periodic pings.
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		err := c.conn.Close()
		if err != nil {
			return
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				return
			}
			if !ok {
				// The channel was closed.
				logger.Debug.Printf("[writePump] Send channel closed for %v", c.conn.RemoteAddr())
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn.Printf("[writePump] Error writing to %v: %v", c.conn.RemoteAddr(), err)
				return
			}

		case <-ticker.C:
			// Send a ping.
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Warn.Printf("[writePump] Ping error for %v: %v", c.conn.RemoteAddr(), err)
				return
			}
		}
	}
}

// registerConnection adds the given connection to the global connections map.
func registerConnection(c *Connection) {
	connections[c] = true
}

// unregisterConnection removes the given connection from the global connections map.
func unregisterConnection(c *Connection) {
	if _, ok := connections[c]; ok {
		delete(connections, c)
	}
}

// DecisionMessage represents the JSON structure of messages from clients.
type DecisionMessage struct {
	Action         string `json:"action"`
	MeetName       string `json:"meetName"`
	JudgeID        string `json:"judgeId"`
	Decision       string `json:"decision"`
	LeftDecision   string `json:"leftDecision"`
	CenterDecision string `json:"centerDecision"`
	RightDecision  string `json:"rightDecision"`
}

// handleIncoming processes an inbound JSON message.
func handleIncoming(c *Connection, dm DecisionMessage) {
	logger.Debug.Printf("[handleIncoming] Action=%s, JudgeID=%s, Meet=%s", dm.Action, dm.JudgeID, dm.MeetName)
	switch dm.Action {
	case "registerRef":
		// Set the judgeID for this connection.
		c.judgeID = dm.JudgeID
		logger.Info.Printf("Referee %s registered on meet %s (conn=%v)", dm.JudgeID, dm.MeetName, c.conn.RemoteAddr())
		// Optionally, broadcast updated referee health.
		broadcastRefereeHealth(dm.MeetName)
	default:
		// Pass the message to processDecision for further handling.
		processDecision(c, dm)
	}
}

// processDecision processes a judge decision message.
// This function is kept for your business logic; update it as needed.
// It now accepts a *Connection instead of a raw *websocket.Conn.
func processDecision(c *Connection, dm DecisionMessage) {
	if dm.JudgeID == "" || dm.Decision == "" {
		logger.Warn.Printf("Incomplete decision message received from %v; ignoring", c.conn.RemoteAddr())
		return
	}
	// Insert your decision-processing logic here.
	logger.Info.Printf("Processing decision from %s: %s (meet: %s)", dm.JudgeID, dm.Decision, dm.MeetName)

	// Example: Broadcast a "judgeSubmitted" message to all connections for this meet.
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": dm.JudgeID,
	}
	out, _ := json.Marshal(submission)
	broadcastToMeet(dm.MeetName, out)
}

// broadcastToMeet sends a message to all connections in the given meet.
func broadcastToMeet(meetName string, message []byte) {
	for c := range connections {
		if c.meetName == meetName {
			select {
			case c.send <- message:
			default:
				// If the send channel is full, drop the message or handle the overflow.
				logger.Warn.Printf("Dropping message for connection %v", c.conn.RemoteAddr())
			}
		}
	}
}

// broadcastRefereeHealth broadcasts the health for a given meet.
// Declare it as a variable so it can be overridden in tests.
var broadcastRefereeHealth = func(meetName string) {
	var connectedIDs []string
	for c := range connections {
		if c.meetName == meetName && c.judgeID != "" {
			connectedIDs = append(connectedIDs, c.judgeID)
		}
	}
	msg := map[string]interface{}{
		"action":            "refereeHealth",
		"connectedRefIDs":   connectedIDs,
		"connectedReferees": len(connectedIDs),
		"requiredReferees":  3, // Adjust as needed.
	}
	out, _ := json.Marshal(msg)
	broadcastToMeet(meetName, out)
}

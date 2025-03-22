// Package websocket provides WebSocket server functionality and connection handling.
// file: websocket/connection.go

package websocket

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// ------------------------- websocket connection interface ------------------

// WSConn defines the interface for a WebSocket connection.
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

// ------------------------- Connection struct ------------------

// Connection represents an individual WebSocket connection.
type Connection struct {
	conn     WSConn      // The actual WebSocket connection interface
	send     chan []byte // Outbound messages get queued here
	meetName string      // The meet to which this connection belongs
	judgeID  string      // Identifies which judge (e.g., "left", "center", etc.) is using it
}

// Global map to store active WebSocket connections.
var connections = make(map[*Connection]bool)
var connectionsMu sync.RWMutex

// ------------------------- Tunable package-level variables ------------------
//
// Changing these from `const` to `var` allows us to override them in tests.
// By default, they have the same long durations as before.

var (
	writeWait      = 4 * time.Hour       // Max time to complete a write
	pongWait       = 4 * time.Hour       // Max time between pongs from the client
	pingPeriod     = (pongWait * 9) / 10 // When to send ping (90% of pongWait)
	maxMessageSize = 2048                // Maximum inbound message size in bytes
)

// Upgrader config: allow any origin for now
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// For local dev or a relaxed policy, we can allow all origins.
		return true
	},
}

// ------------------------- HTTP -> WebSocket upgrade ------------------

// ServeWs upgrades an HTTP request to a WebSocket connection and starts pumps.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		logger.Error.Println("[ServeWs] No meet selected; rejecting WebSocket connection")
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

	// create and register new WebSocket connection
	conn := &Connection{
		conn:     wsConn,
		send:     make(chan []byte, 256), // buffered channel
		meetName: meetName,
		judgeID:  "", // set by "registerRef" message
	}

	registerConnection(conn)

	// start pumps
	go conn.readPump()
	go conn.writePump()
}

// ------------------------ read/write pumps -----------------------

// readPump listens for messages from the WebSocket client.
func (c *Connection) readPump() {
	defer func() {
		unregisterConnection(c)
		_ = c.conn.Close()
	}()

	// Limit message size
	c.conn.SetReadLimit(int64(maxMessageSize))

	// Initial read deadline
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Whenever we get a Pong frame, reset the read deadline
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
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

// writePump handles outgoing messages to the WebSocket client.
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// For each write, update write deadline
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				// channel closed => send a close frame
				logger.Debug.Printf("[writePump] Send channel closed for %v", c.conn.RemoteAddr())
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn.Printf("[writePump] Error writing to %v: %v", c.conn.RemoteAddr(), err)
				return
			}

		case <-ticker.C:
			// Time to send a Ping
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Warn.Printf("[writePump] Ping error for %v: %v", c.conn.RemoteAddr(), err)
				return
			}
		}
	}
}

// ------------------------ connection management -----------------------

// registerConnection adds a new WebSocket connection to the global map.
func registerConnection(c *Connection) {
	connectionsMu.Lock()
	connections[c] = true
	connectionsMu.Unlock()
}

// unregisterConnection removes a WebSocket connection from the global map.
func unregisterConnection(c *Connection) {
	connectionsMu.Lock()
	delete(connections, c)
	connectionsMu.Unlock()
}

// ------------------------ message handling -----------------------

// DecisionMessage is the JSON structure from clients.
type DecisionMessage struct {
	Action         string `json:"action"`
	MeetName       string `json:"meetName"`
	JudgeID        string `json:"judgeId"`
	Decision       string `json:"decision"`
	LeftDecision   string `json:"leftDecision"`
	CenterDecision string `json:"centerDecision"`
	RightDecision  string `json:"rightDecision"`
}

// handleIncoming processes inbound JSON messages.
func handleIncoming(c *Connection, dm DecisionMessage) {
	logger.Debug.Printf("[handleIncoming] Action=%s, JudgeID=%s, Meet=%s",
		dm.Action, dm.JudgeID, dm.MeetName)

	switch dm.Action {
	case "registerRef":
		c.judgeID = dm.JudgeID
		logger.Info.Printf("[handleIncoming] Referee=%s registered on meet=%s (conn=%v)",
			dm.JudgeID, dm.MeetName, c.conn.RemoteAddr())
		broadcastRefereeHealth(dm.MeetName)

	case "startTimer":
		logger.Info.Printf("[handleIncoming] Received startTimer from %v", c.conn.RemoteAddr())
		defaultTimerManager.HandleTimerAction("startTimer", dm.MeetName)

	case "resetLights":
		logger.Info.Printf("[handleIncoming] Received resetLights from %v", c.conn.RemoteAddr())
		msg := map[string]string{
			"action":   "resetLights",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			logger.Error.Printf("[handleIncoming] Error marshaling resetLights: %v", err)
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "resetTimer":
		logger.Info.Printf("[handleIncoming] Received resetTimer from %v", c.conn.RemoteAddr())
		msg := map[string]string{
			"action":   "resetTimer",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			logger.Error.Printf("[handleIncoming] Error marshaling resetTimer: %v", err)
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "submitDecision":
		processDecision(c, dm)

	default:
		logger.Debug.Printf("[handleIncoming] Unhandled action=%s", dm.Action)
	}
}

// processDecision checks if all judge decisions have arrived, then broadcasts final results if so.
func processDecision(c *Connection, dm DecisionMessage) {
	if dm.JudgeID == "" || dm.Decision == "" {
		logger.Warn.Printf("[processDecision] Incomplete decision from %v; ignoring", c.conn.RemoteAddr())
		return
	}
	logger.Info.Printf("[processDecision] Processing decision from judge=%s: %s (meet=%s)",
		dm.JudgeID, dm.Decision, dm.MeetName)

	meetState := DefaultStateProvider.GetMeetState(dm.MeetName)
	meetState.JudgeDecisions[dm.JudgeID] = dm.Decision

	// If all three decisions are in, broadcast final results.
	if len(meetState.JudgeDecisions) >= 3 {
		broadcastFinalResults(dm.MeetName)
	}

	// Also broadcast that this judge submitted a decision.
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": dm.JudgeID,
	}
	out, err := json.Marshal(submission)
	if err != nil {
		logger.Error.Printf("[processDecision] Error marshaling judgeSubmitted: %v", err)
		return
	}
	broadcastToMeet(dm.MeetName, out)
}

// broadcastToMeet sends a message to all connections in the given meet.
var broadcastToMeet = func(meetName string, message []byte) {
	connectionsMu.RLock()
	defer connectionsMu.RUnlock()

	for c := range connections {
		if c.meetName == meetName {
			select {
			case c.send <- message:
			default:
				logger.Warn.Printf("[broadcastToMeet] Dropping message for %v", c.conn.RemoteAddr())
			}
		}
	}
}

var broadcastRefereeHealth = func(meetName string) {
	var connectedIDs []string

	connectionsMu.RLock()
	for c := range connections {
		if c.meetName == meetName && c.judgeID != "" {
			connectedIDs = append(connectedIDs, c.judgeID)
		}
	}
	connectionsMu.RUnlock()

	msg := map[string]interface{}{
		"action":            "refereeHealth",
		"connectedRefIDs":   connectedIDs,
		"connectedReferees": len(connectedIDs),
		"requiredReferees":  3,
	}
	out, _ := json.Marshal(msg)
	broadcastToMeet(meetName, out)
}

// Package websocket provides WebSocket server functionality and connection handling.
// file: websocket/connection.go
package websocket

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"

	"go-ref-lights/logger"

	"github.com/gorilla/websocket"
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

// Connection represents an individual WebSocket connection
type Connection struct {
	conn     WSConn      // the actual WebSocket connection interface
	send     chan []byte // outbound messages get queued here
	meetName string      // the meet to which this connection belongs
	judgeID  string      // identifies which judge (e.g., "left", "center", etc.) is using it
}

// global map to store active WebSocket connections
var connections = make(map[*Connection]bool)
var connectionsMu sync.RWMutex

// ------------------------- Tunable package-level variables ------------------

// Changing these from `const` to `var` allows us to override them in tests.
// By default, they have the same long durations as before.
var (
	writeWait  = 4 * time.Hour       // Max time to complete a write
	pongWait   = 4 * time.Hour       // Max time between pongs from the client
	pingPeriod = (pongWait * 9) / 10 // When to send ping (90% of pongWait)
)

// Upgrader config: allow any origin for now
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// for local dev or a relaxed policy, we can allow all origins
		return true
	},
}

// ------------------------- HTTP -> WebSocket upgrade ------------------

// ServeWs upgrades an HTTP request to a WebSocket connection and starts pumps
func ServeWs(w http.ResponseWriter, r *http.Request) {
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		// Keep WARN level for missing meetName parameter
		logContext := logger.NewWebSocketContext("missing_meet_name", "Anonymous", "", r.RemoteAddr)
		logger.LogWarnWithContext(logContext, "No meetName provided in WebSocket upgrade, proceeding with Anonymous")
		meetName = "Anonymous"
	}

	// Convert routine connection upgrade to DEBUG level
	logContext := logger.NewWebSocketContext("connection_upgrade", meetName, "", r.RemoteAddr)
	logger.LogDebugWithContext(logContext, "WebSocket connection upgrade initiated")
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log WebSocket upgrade failure with comprehensive error context
		errorCtx := logger.NewWebSocketErrorContext(
			"WebSocket upgrade failed",
			meetName,
			"",
			r.RemoteAddr,
		).WithCode("WS_001").
			WithError(err).
			WithDetail("userAgent", r.UserAgent()).
			WithDetail("origin", r.Header.Get("Origin")).
			WithDetail("protocol", r.Header.Get("Sec-WebSocket-Protocol")).
			WithDetail("requestPath", r.URL.Path).
			WithDetail("requestMethod", r.Method)

		errorCtx.LogError()

		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}

	// create a Connection carrying the same context
	conn := &Connection{
		conn:     wsConn,
		send:     make(chan []byte, 256),
		meetName: meetName,
		judgeID:  "",
	}

	registerConnection(conn)

	// start pumps
	go conn.readPump()
	go conn.writePump()
}

// ------------------------ read/write pumps -----------------------

// readPump listens for messages from the WebSocket client
func (c *Connection) readPump() {
	defer func() {
		c.logConnectionClosure()
		unregisterConnection(c)
		_ = c.conn.Close()
	}()

	// Configure connection limits and timeouts
	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		context := logger.NewWebSocketContext("read_deadline_error", c.meetName, c.judgeID, c.conn.RemoteAddr().String())
		logger.LogWarnWithContext(context, "Failed to set initial read deadline: %v", err)
		return
	}
	c.conn.SetPongHandler(func(string) error {
		// Return the error from SetReadDeadline so gorilla can handle connection cleanup
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logReadError(err)
			break
		}

		if messageType == websocket.TextMessage {
			c.handleTextMessage(message)
		}
	}
}

// logConnectionClosure logs connection closure with context
func (c *Connection) logConnectionClosure() {
	context := logger.NewWebSocketContext("connection_closed", c.meetName, c.judgeID, c.conn.RemoteAddr().String())
	logger.LogDebugWithContext(context, "WebSocket connection closed")
}

// logReadError logs read errors with appropriate level based on error type
func (c *Connection) logReadError(err error) {
	context := logger.NewWebSocketContext("read_error", c.meetName, c.judgeID, c.conn.RemoteAddr().String())

	// Check if it's a normal closure
	if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
		errorCtx := logger.NewWebSocketErrorContext(
			"Unexpected WebSocket connection closure",
			c.meetName,
			c.judgeID,
			c.conn.RemoteAddr().String(),
		).WithCode("WS_010").WithError(err)

		errorCtx.LogWarn()
	} else {
		// Normal closure - log at debug level
		logger.LogDebugWithContext(context, "WebSocket connection closed normally")
	}
}

// handleTextMessage processes incoming text messages with enhanced error handling
func (c *Connection) handleTextMessage(message []byte) {
	var dm DecisionMessage
	if jsonErr := json.Unmarshal(message, &dm); jsonErr != nil {
		errorCtx := logger.NewWebSocketErrorContext(
			"Failed to parse incoming JSON message",
			c.meetName,
			c.judgeID,
			c.conn.RemoteAddr().String(),
		).WithCode("WS_002").
			WithError(jsonErr).
			WithDetail("messageLength", len(message)).
			WithDetail("messagePreview", c.safeMessagePreview(message))

		errorCtx.LogWarn()
		return
	}

	handleIncoming(c, dm)
}

// safeMessagePreview creates a safe preview of message content
func (c *Connection) safeMessagePreview(message []byte) string {
	const maxPreviewLength = 100
	if len(message) <= maxPreviewLength {
		return string(message)
	}
	return string(message[:maxPreviewLength]) + "..."
}

// maxMessageSize defines the maximum allowed message size
const maxMessageSize = 512

// writePump handles outgoing messages to the WebSocket client
func (c *Connection) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			// for each write, update write deadline
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if !ok {
				// channel closed => send a close frame
				// Keep as DEBUG level for routine channel closure
				context := logger.NewWebSocketContext("send_channel_closed", c.meetName, c.judgeID, c.conn.RemoteAddr().String())
				logger.LogDebugWithContext(context, "Send channel closed, terminating connection")
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				// Log message write failure with comprehensive error context
				errorCtx := logger.NewWebSocketErrorContext(
					"Failed to write message to WebSocket connection",
					c.meetName,
					c.judgeID,
					c.conn.RemoteAddr().String(),
				).WithCode("WS_003").
					WithError(err).
					WithDetail("messageLength", len(message)).
					WithDetail("messageType", "text")

				errorCtx.LogWarn()
				return
			}

		case <-ticker.C:
			// time to send a Ping
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// Log ping failure with comprehensive error context
				errorCtx := logger.NewWebSocketErrorContext(
					"Failed to send ping message",
					c.meetName,
					c.judgeID,
					c.conn.RemoteAddr().String(),
				).WithCode("WS_004").
					WithError(err).
					WithDetail("messageType", "ping").
					WithDetail("keepAliveCheck", true)

				errorCtx.LogWarn()
				return
			}
		}
	}
}

// ------------------------ connection management -----------------------

// registerConnection adds a new WebSocket connection to the global map
func registerConnection(c *Connection) {
	connectionsMu.Lock()
	connections[c] = true
	connectionsMu.Unlock()
}

// unregisterConnection removes a WebSocket connection from the global map
func unregisterConnection(c *Connection) {
	connectionsMu.Lock()
	delete(connections, c)
	connectionsMu.Unlock()
}

// ------------------------ message handling -----------------------

// DecisionMessage is the JSON structure from clients
type DecisionMessage struct {
	Action         string `json:"action"`
	MeetName       string `json:"meetName"`
	JudgeID        string `json:"judgeId"`
	Decision       string `json:"decision"`
	LeftDecision   string `json:"leftDecision"`
	CenterDecision string `json:"centerDecision"`
	RightDecision  string `json:"rightDecision"`
}

// handleIncoming processes inbound JSON messages
func handleIncoming(c *Connection, dm DecisionMessage) {
	// Keep as DEBUG level for routine message handling
	context := logger.NewWebSocketContext("message_received", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
	context["messageAction"] = dm.Action
	logger.LogDebugWithContext(context, "Processing incoming WebSocket message")

	switch dm.Action {
	case "registerRef":
		c.judgeID = dm.JudgeID
		// Convert routine referee registration to DEBUG level
		context := logger.NewWebSocketContext("referee_registered", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
		logger.LogDebugWithContext(context, "Referee registered successfully")
		broadcastRefereeHealth(dm.MeetName)

	case "startTimer":
		// Convert routine timer start to DEBUG level
		context := logger.NewWebSocketContext("start_timer_received", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
		logger.LogDebugWithContext(context, "Start timer command received")
		defaultTimerManager.HandleTimerAction("startTimer", dm.MeetName)

	case "resetLights":
		// Convert routine reset lights to DEBUG level
		context := logger.NewWebSocketContext("reset_lights_received", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
		logger.LogDebugWithContext(context, "Reset lights command received")
		msg := map[string]string{
			"action":   "resetLights",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			// Log marshaling failure with comprehensive error context
			errorCtx := logger.NewErrorContext(logger.MarshalingError, logger.SeverityMedium, "Failed to marshal resetLights message").
				WithCode("WS_006").
				WithMeet(dm.MeetName, dm.JudgeID).
				WithError(err).
				WithDetail("messageType", "resetLights").
				WithDetail("remoteAddr", c.conn.RemoteAddr().String())

			errorCtx.LogError()
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "resetTimer":
		// Convert routine reset timer to DEBUG level
		context := logger.NewWebSocketContext("reset_timer_received", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
		logger.LogDebugWithContext(context, "Reset timer command received")
		msg := map[string]string{
			"action":   "resetTimer",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			// Log marshaling failure with comprehensive error context
			errorCtx := logger.NewErrorContext(logger.MarshalingError, logger.SeverityMedium, "Failed to marshal resetTimer message").
				WithCode("WS_007").
				WithMeet(dm.MeetName, dm.JudgeID).
				WithError(err).
				WithDetail("messageType", "resetTimer").
				WithDetail("remoteAddr", c.conn.RemoteAddr().String())

			errorCtx.LogError()
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "submitDecision":
		processDecision(c, dm)

	default:
		// Keep as DEBUG level for unhandled actions
		context := logger.NewWebSocketContext("unhandled_action", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
		context["action"] = dm.Action
		logger.LogDebugWithContext(context, "Received unhandled WebSocket action")
	}
}

// processDecision checks if all judge decisions have arrived, then broadcasts final results if so
func processDecision(c *Connection, dm DecisionMessage) {
	if dm.JudgeID == "" || dm.Decision == "" {
		// Log incomplete decision with comprehensive error context
		errorCtx := logger.NewWebSocketErrorContext(
			"Received incomplete decision data, ignoring",
			dm.MeetName,
			dm.JudgeID,
			c.conn.RemoteAddr().String(),
		).WithCode("WS_005").
			WithDetail("judgeIdProvided", dm.JudgeID != "").
			WithDetail("decisionProvided", dm.Decision != "").
			WithDetail("action", dm.Action).
			WithDetail("validationFailure", "missing_required_fields")

		errorCtx.LogWarn()
		return
	}
	// Convert routine decision processing to DEBUG level
	context := logger.NewWebSocketContext("decision_received", dm.MeetName, dm.JudgeID, c.conn.RemoteAddr().String())
	context["decision"] = dm.Decision
	logger.LogDebugWithContext(context, "Processing referee decision")

	meetState := DefaultStateProvider.GetMeetState(dm.MeetName)
	meetState.JudgeDecisions[dm.JudgeID] = dm.Decision

	// if all three decisions are in, broadcast final results
	if len(meetState.JudgeDecisions) >= 3 {
		broadcastFinalResults(dm.MeetName)
	}

	// also broadcast that this judge submitted a decision
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": dm.JudgeID,
	}
	out, err := json.Marshal(submission)
	if err != nil {
		// Log marshaling failure with comprehensive error context
		errorCtx := logger.NewErrorContext(logger.MarshalingError, logger.SeverityMedium, "Failed to marshal judgeSubmitted message").
			WithCode("WS_008").
			WithMeet(dm.MeetName, dm.JudgeID).
			WithError(err).
			WithDetail("messageType", "judgeSubmitted").
			WithDetail("remoteAddr", c.conn.RemoteAddr().String()).
			WithDetail("decision", dm.Decision)

		errorCtx.LogError()
		return
	}
	broadcastToMeet(dm.MeetName, out)
}

// broadcastToMeet sends a message to all connections in the given meet
var broadcastToMeet = func(meetName string, message []byte) {
	connectionsMu.RLock()
	defer connectionsMu.RUnlock()

	for c := range connections {
		if c.meetName == meetName {
			select {
			case c.send <- message:
			default:
				// Log dropped message with comprehensive error context
				errorCtx := logger.NewWebSocketErrorContext(
					"Dropping message due to full send channel",
					meetName,
					c.judgeID,
					c.conn.RemoteAddr().String(),
				).WithCode("WS_009").
					WithDetail("messageLength", len(message)).
					WithDetail("channelFull", true).
					WithDetail("connectionHealth", "degraded")

				errorCtx.LogWarn()
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

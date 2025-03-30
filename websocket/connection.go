// Package websocket provides WebSocket server functionality and connection handling.
// file: websocket/connection.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
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

// Connection represents an individual WebSocket connection
type Connection struct {
	conn     WSConn      // the actual WebSocket connection interface
	send     chan []byte // outbound messages get queued here
	meetName string      // the meet to which this connection belongs
	judgeID  string      // identifies which judge (e.g., "left", "center", etc.) is using it
	ctx      context.Context
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
	//maxMessageSize = 2048                // Maximum inbound message size in bytes
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
		logger.Warn.Println("No meetName provided; proceeding anyway.")
		meetName = "Anonymous"
	}

	// Start a subsegment for the WebSocket upgrade event
	ctx, seg := xray.BeginSubsegment(r.Context(), "WebSocketUpgrade")
	if seg != nil {
		_ = seg.AddAnnotation("remoteAddr", r.RemoteAddr)
		_ = seg.AddAnnotation("meetName", meetName)
	}

	logger.Info.Printf("[ServeWs] Upgrading to WS: remoteAddr=%v, meetName=%q", r.RemoteAddr, meetName)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error.Printf("[ServeWs] WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		if seg != nil {
			seg.Close(nil)
		}
		return
	}

	// end the subsegment once we have the WebSocket
	if seg != nil {
		seg.Close(nil)
	}

	// create a Connection carrying the same context
	conn := &Connection{
		conn:     wsConn,
		send:     make(chan []byte, 256),
		meetName: meetName,
		judgeID:  "",
		ctx:      ctx, // store the context in case readPump/writePump want to do subsegments
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
		unregisterConnection(c)
		_ = c.conn.Close()
	}()

	for {
		// create a subsegment for each read cycle
		_, subSeg := xray.BeginSubsegment(c.ctx, "WebSocketRead")

		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			// if subSeg isn't nil, annotate and close it
			if subSeg != nil {
				_ = subSeg.AddAnnotation("readError", err.Error())
				subSeg.Close(nil)
			}
			// break from the loop (closing the connection)
			break
		}

		// if we have a subSeg, do your annotation
		if subSeg != nil {
			_ = subSeg.AddAnnotation("messageType", fmt.Sprintf("%d", messageType))
			subSeg.Close(nil)
		}

		// handle text messages
		if messageType == websocket.TextMessage {
			var dm DecisionMessage
			if jsonErr := json.Unmarshal(message, &dm); jsonErr != nil {
				logger.Warn.Printf("[readPump] JSON parse error: %v", jsonErr)
				// if subSeg != nil { annotate parse error }
			} else {
				handleIncoming(c, dm)
			}
		}
	}
}

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
				logger.Debug.Printf("[writePump] Send channel closed for %v", c.conn.RemoteAddr())
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Warn.Printf("[writePump] Error writing to %v: %v", c.conn.RemoteAddr(), err)
				return
			}

		case <-ticker.C:
			// time to send a Ping
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
	// start a subsegment for each message action
	_, seg := xray.BeginSubsegment(c.ctx, "HandleIncoming")
	defer seg.Close(nil)

	err := seg.AddAnnotation("action", dm.Action)
	if err != nil {
		return
	}
	err = seg.AddAnnotation("judgeID", dm.JudgeID)
	if err != nil {
		return
	}
	err = seg.AddAnnotation("meet", dm.MeetName)
	if err != nil {
		return
	}

	logger.Debug.Printf("[handleIncoming] Action=%s, JudgeID=%s, Meet=%s",
		dm.Action, dm.JudgeID, dm.MeetName)

	switch dm.Action {
	case "registerRef":
		c.judgeID = dm.JudgeID
		logger.Info.Printf("Referee %s registered on meet %s (conn=%v)",
			dm.JudgeID, dm.MeetName, c.conn.RemoteAddr())
		broadcastRefereeHealth(dm.MeetName)

	case "startTimer":
		logger.Info.Printf("Received startTimer from %v", c.conn.RemoteAddr())
		defaultTimerManager.HandleTimerAction("startTimer", dm.MeetName)

	case "resetLights":
		logger.Info.Printf("Received resetLights from %v", c.conn.RemoteAddr())
		msg := map[string]string{
			"action":   "resetLights",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			logger.Error.Printf("Error marshaling resetLights: %v", err)
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "resetTimer":
		logger.Info.Printf("Received resetTimer from %v", c.conn.RemoteAddr())
		msg := map[string]string{
			"action":   "resetTimer",
			"meetName": dm.MeetName,
		}
		out, err := json.Marshal(msg)
		if err != nil {
			logger.Error.Printf("Error marshaling resetTimer: %v", err)
		} else {
			broadcastToMeet(dm.MeetName, out)
		}

	case "submitDecision":
		processDecision(c, dm)

	default:
		logger.Debug.Printf("Unhandled action: %s", dm.Action)
	}
}

// processDecision checks if all judge decisions have arrived, then broadcasts final results if so
func processDecision(c *Connection, dm DecisionMessage) {
	if dm.JudgeID == "" || dm.Decision == "" {
		logger.Warn.Printf("Incomplete decision from %v; ignoring", c.conn.RemoteAddr())
		return
	}
	logger.Info.Printf("Processing decision from %s: %s (meet: %s)",
		dm.JudgeID, dm.Decision, dm.MeetName)

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
		logger.Error.Printf("Error marshaling judgeSubmitted: %v", err)
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
				logger.Warn.Printf("Dropping message for %v", c.conn.RemoteAddr())
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

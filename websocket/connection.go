// Package websocket - websocket/connection.go
package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
	"net/http"
	"sync"
)

// connectionMapping maps each WebSocket conn -> (meetName, judgeID)
var connectionMapping = make(map[*websocket.Conn]connectionInfo)

// store info about which meet and judge is associated with each connection
// this helps us remove them from the correct meet's state on disconnection
type connectionInfo struct {
	meetName string
	judgeID  string
	mu       sync.Mutex
}

// ServeWs is the main WebSocket entry point
func ServeWs(w http.ResponseWriter, r *http.Request) {
	meetName := r.URL.Query().Get("meetName")
	logger.Info.Printf("[ServeWs] HTTP upgrade for remoteAddr=%v, requested meet=%q", r.RemoteAddr, meetName)

	if meetName == "" {
		logger.Error.Println("No meet selected; rejecting WebSocket connection")
		http.Error(w, "No meet selected", http.StatusBadRequest)
		return
	}

	defer func() {
		if err := recover(); err != nil {
			logger.Error.Printf("⚠️ Recovered from panic: %v", err)
		}
	}()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error.Printf("[ServeWs] WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}

	logger.Info.Printf("[ServeWs] Upgraded to WebSocket: %v", conn.RemoteAddr())

	// track globally
	clients[conn] = true
	logger.Info.Printf("[ServeWs] Now launching handleReads(...) for meet '%s' on conn '%v'", meetName, conn.RemoteAddr())

	// start the heartbeat (pings) in the background
	go startHeartbeat(conn)

	// start reading messages from this connection
	logger.Info.Printf("🌐 Starting WebSocket message reading for: %s", meetName)
	go handleReads(conn)
}

// processDecision handles a decision message from a referee
func processDecision(decisionMsg DecisionMessage, conn *websocket.Conn) {
	meetState := getMeetState(decisionMsg.MeetName)

	// basic validation
	if decisionMsg.JudgeID == "" || decisionMsg.Decision == "" {
		logger.Warn.Printf("Incomplete decision message received from %v; ignoring", conn.RemoteAddr())
		return
	}

	// single-session enforcement for this judge in this meet
	existingConn, exists := meetState.RefereeSessions[decisionMsg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		logger.Warn.Printf("🔴 Kicking out old session for referee: %s in meet: %s", decisionMsg.JudgeID, decisionMsg.MeetName)
		_ = existingConn.Close()
		delete(clients, existingConn) // remove from the global set
		meetState.RefereeSessions[decisionMsg.JudgeID] = nil

		// also remove from connectionMapping
		if info, ok := connectionMapping[existingConn]; ok {
			if info.judgeID == decisionMsg.JudgeID && info.meetName == decisionMsg.MeetName {
				delete(connectionMapping, existingConn)
			}
		}
	}
	// store the new connection in meet state
	meetState.RefereeSessions[decisionMsg.JudgeID] = conn
	meetState.JudgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision

	// update global mapping so we know this connection -> meet/judge
	connectionMapping[conn] = connectionInfo{
		meetName: decisionMsg.MeetName,
		judgeID:  decisionMsg.JudgeID,
	}

	logger.Info.Printf("✅ Decision from %s: %s (meet: %s)", decisionMsg.JudgeID, decisionMsg.Decision, decisionMsg.MeetName)

	// let everyone know a judge submitted
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(submission)
	broadcast <- subMsg

	// also broadcast updated health (who is connected) for this meet
	broadcastRefereeHealth(meetState)

	// if we have decisions from all 3 judges, broadcast final results
	if len(meetState.JudgeDecisions) == 3 {
		broadcastFinalResults(decisionMsg.MeetName)
	}
}

// function that writes messages to WebSocket
func safeWriteMessage(conn *websocket.Conn, messageType int, data []byte) error {
	if info, ok := connectionMapping[conn]; ok {
		info.mu.Lock()
		defer info.mu.Unlock()
	}
	return conn.WriteMessage(messageType, data)
}

// handleReads reads messages from a connection and processes them
func handleReads(conn *websocket.Conn) {
	logger.Info.Printf("[handleReads] Starting read loop for client: %v", conn.RemoteAddr())
	defer func() {
		// On disconnection, remove from global clients map
		logger.Warn.Printf("[handleReads] Closing/cleanup for client: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)

		// If we know which meet/judge this connection belonged to, remove from that meet's state
		if info, ok := connectionMapping[conn]; ok {
			meetState := getMeetState(info.meetName)
			if meetState.RefereeSessions[info.judgeID] == conn {
				meetState.RefereeSessions[info.judgeID] = nil
				logger.Info.Printf("🚪 Removing %s from meet %s (due to disconnect)", info.judgeID, info.meetName)
				broadcastRefereeHealth(meetState)
			}
			delete(connectionMapping, conn)
		}
	}()

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Warn.Printf(
				"[handleReads] read error from %v: %v",
				conn.RemoteAddr(),
				err,
			)
			return
		}

		logger.Debug.Printf(
			"[handleReads] Received messageType=%d from %v: %s",
			messageType,
			conn.RemoteAddr(),
			string(msg),
		)

		var decisionMsg DecisionMessage
		if err := json.Unmarshal(msg, &decisionMsg); err != nil {
			logger.Warn.Printf(
				"[handleReads] invalid JSON from %v: %v",
				conn.RemoteAddr(),
				err,
			)
			continue
		}

		logger.Info.Printf(
			"[handleReads] Action=%s JudgeID=%s Meet=%s",
			decisionMsg.Action,
			decisionMsg.JudgeID,
			decisionMsg.MeetName,
		)

		// if the JSON has no meetName, log a warning
		switch decisionMsg.Action {
		case "registerRef":
			logger.Info.Printf(
				"🟢 startTimer action received from %s for meet: %s",
				decisionMsg.JudgeID,
				decisionMsg.MeetName,
			)
			registerRef(decisionMsg, conn)

		case "startTimer", "startNextAttemptTimer", "resetTimer", "updatePlatformReadyTime":
			handleTimerAction(decisionMsg.Action, decisionMsg.MeetName)

		case "resetLights":
			logger.Info.Println("Resetting lights")
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action": "resetLights",
			})

		case "judgeSubmitted":
			logger.Info.Printf("Judge %s submitted a decision", decisionMsg.JudgeID)
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action":  "judgeSubmitted",
				"judgeId": decisionMsg.JudgeID,
			})

		case "displayResults":
			logger.Info.Println("Displaying results to all clients")
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action":         "displayResults",
				"leftDecision":   decisionMsg.LeftDecision,
				"centreDecision": decisionMsg.CentreDecision,
				"rightDecision":  decisionMsg.RightDecision,
			})

		case "clearResults":
			logger.Info.Println("Clearing results from UI")
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action": "clearResults",
			})

		case "platformReadyExpired":
			logger.Info.Println("Platform ready timer expired")
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action": "platformReadyExpired",
			})

		case "nextAttemptExpired":
			logger.Info.Println("Next attempt timer expired")
			broadcastMessage(decisionMsg.MeetName, map[string]interface{}{
				"action": "nextAttemptExpired",
			})

		default:
			// Treat as a judge decision
			processDecision(decisionMsg, conn)
		}
	}
}

// registerRef marks a referee as connected in the meetState
func registerRef(msg DecisionMessage, conn *websocket.Conn) {
	if msg.JudgeID == "" {
		// if we have no judgeId, raise warning
		logger.Warn.Printf("registerRef called with empty JudgeID, ignoring connection.")
		return
	}

	meetState := getMeetState(msg.MeetName)
	// if there's an existing session for that judge, close it out
	existingConn, exists := meetState.RefereeSessions[msg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		logger.Warn.Printf("🔴 Kicking out old session for ref: %s in meet: %s", msg.JudgeID, msg.MeetName)
		_ = existingConn.Close()
		delete(clients, existingConn)
	}

	// store the new connection in meet state
	meetState.RefereeSessions[msg.JudgeID] = conn

	// also update global mapping
	connectionMapping[conn] = connectionInfo{
		meetName: msg.MeetName,
		judgeID:  msg.JudgeID,
	}

	logger.Info.Printf("✅ Referee %s registered (meet: %s)", msg.JudgeID, msg.MeetName)
	broadcastRefereeHealth(meetState)
}

// broadcastRefereeHealth sends a message to all clients w/ current ref health
func broadcastRefereeHealth(meetState *MeetState) {
	var connectedIDs []string
	for judgeID, c := range meetState.RefereeSessions {
		if c != nil {
			connectedIDs = append(connectedIDs, judgeID)
		}
	}
	msg := map[string]interface{}{
		"action":            "refereeHealth",
		"connectedRefIDs":   connectedIDs,
		"connectedReferees": len(connectedIDs),
		"requiredReferees":  3, // or however many you expect
	}
	out, _ := json.Marshal(msg)
	broadcast <- out
}

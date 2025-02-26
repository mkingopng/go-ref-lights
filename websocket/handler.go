// Package websocket: contains the WebSocket handler and related functions
package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// GLOBALS

// clients tracks all connected clients (for broadcast usage)
var clients = make(map[*websocket.Conn]bool)

// global mutex to synchronise writes
var writeMutex sync.Mutex

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// store info about which meet and judge is associated with each connection
// this helps us remove them from the correct meet's state on disconnection
type connectionInfo struct {
	meetName string
	judgeID  string
	mu       sync.Mutex
}

// connectionMapping maps each WebSocket conn -> (meetName, judgeID)
var connectionMapping = make(map[*websocket.Conn]connectionInfo)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// mutexes for concurrency around timers
var (
	platformReadyMutex = &sync.Mutex{}
	nextAttemptMutex   = &sync.Mutex{}
)

// WEBSOCKET UPGRADE
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all if Test-Mode
		if r.Header.Get("Test-Mode") == "true" {
			return true
		}
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080" ||
			origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// function that writes messages to WebSocket
func safeWriteMessage(conn *websocket.Conn, messageType int, data []byte) error {
	if info, ok := connectionMapping[conn]; ok {
		info.mu.Lock()
		defer info.mu.Unlock()
	}
	return conn.WriteMessage(messageType, data)
}

// ServeWs is the main WebSocket entry point
func ServeWs(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error.Printf("⚠️ Recovered from panic: %v", err)
		}
	}()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	logger.Info.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	// track globally
	clients[conn] = true

	// if there's no meetName query param, fall back to "DEFAULT_MEET"
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		meetName = "DEFAULT_MEET"
	}

	// start the heartbeat (pings) in the background
	go startHeartbeat(conn)

	// start reading messages from this connection
	go handleReads(conn, meetName)
}

// GLOBAL BROADCAST LOOP

// HandleMessages copy clients before iteration
func HandleMessages() {
	for {
		msg := <-broadcast
		clientsCopy := make(map[*websocket.Conn]bool)
		writeMutex.Lock()
		for k, v := range clients {
			clientsCopy[k] = v
		}
		writeMutex.Unlock()

		for conn := range clientsCopy {
			// Use safeWriteMessage
			if err := safeWriteMessage(conn, websocket.TextMessage, msg); err != nil {
				logger.Error.Printf("❌ Failed to send broadcast message to %v: %v", conn.RemoteAddr(), err)
			}
		}
	}
}

// MESSAGE READING & DISCONNECTION HANDLING

// handleReads reads messages from a connection and processes them
func handleReads(conn *websocket.Conn, defaultMeetName string) {
	defer func() {
		// On disconnection, remove from global clients map
		logger.Warn.Printf("⚠️ WebSocket disconnected: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)

		// If we know which meet/judge this conn belonged to, remove from that meet's state
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
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Warn.Printf("⚠️ WebSocket read error: %v", err)
			return
		}

		logger.Debug.Printf("Received raw message from %v: %s", conn.RemoteAddr(), string(msg))

		var decisionMsg DecisionMessage
		if err := json.Unmarshal(msg, &decisionMsg); err != nil {
			logger.Warn.Printf("⚠️ Invalid JSON: %v", err)
			continue
		}

		// if the JSON has no meetName, fallback to the default
		if decisionMsg.MeetName == "" {
			decisionMsg.MeetName = defaultMeetName
		}

		// distinguish between "actions" vs. "decisions"
		switch decisionMsg.Action {
		case "registerRef":
			// store the connection, broadcast health
			registerRef(decisionMsg, conn)
		case "startTimer", "stopTimer", "resetTimer", "startNextAttemptTimer":
			handleTimerAction(decisionMsg.Action, decisionMsg.MeetName)
		default:
			// if there's no recognised Action, we treat it as a Decision
			processDecision(decisionMsg, conn)
		}
	}
}

// registerRef marks a referee as connected in the meetState
func registerRef(msg DecisionMessage, conn *websocket.Conn) {
	if msg.JudgeID == "" {
		// if we have no judgeId, do nothing
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

	// store the new conn
	meetState.RefereeSessions[msg.JudgeID] = conn

	// also update global mapping
	connectionMapping[conn] = connectionInfo{
		meetName: msg.MeetName,
		judgeID:  msg.JudgeID,
	}

	logger.Info.Printf("✅ Referee %s registered (meet: %s)", msg.JudgeID, msg.MeetName)
	broadcastRefereeHealth(meetState)
}

// HEARTBEAT (PING) KEEPS CONNECTIONS ALIVE

// startHeartbeat sends a ping every 10 seconds to keep the connection alive
func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	failedPings := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failedPings++
			logger.Warn.Printf("⚠️ WebSocket ping failed (%d/3): %v", failedPings, err)
			if failedPings >= 5 {
				logger.Error.Println("❌ Connection lost due to repeated ping failures.")
				_ = conn.Close()
				delete(clients, conn)

				// also remove from connectionMapping if needed
				if info, ok := connectionMapping[conn]; ok {
					meetState := getMeetState(info.meetName)
					if meetState.RefereeSessions[info.judgeID] == conn {
						meetState.RefereeSessions[info.judgeID] = nil
					}
					delete(connectionMapping, conn)
					broadcastRefereeHealth(meetState)
				}
				return
			}
		} else {
			failedPings = 0
		}
	}
}

// DECISION & REFEREE HANDLING

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
		delete(clients, existingConn) // remove from global set
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

	// update global mapping so we know this conn -> meet/judge
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

// broadcastFinalResults sends the final decisions to all clients
func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	// 1) broadcast the final decisions
	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centreDecision": meetState.JudgeDecisions["centre"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// 2) immediately start the next-lifter timer
	startNextAttemptTimer(meetState)

	// 3) clear results after set duration
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// 4) reset for next lift
	meetState.JudgeDecisions = make(map[string]string)
}

// TIMER / PLATFORM READY

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	meetState := getMeetState(meetName)
	switch action {
	case "startTimer":
		// only allow "Platform Ready" if all refs are connected // fix_me
		//if !isAllRefsConnected(meetState) {
		//	errMsg := map[string]string{
		//		"action":  "healthError",
		//		"message": "Cannot start timer: Not all referees are connected!",
		//	}
		//	msg, _ := json.Marshal(errMsg)
		//	broadcast <- msg
		//	logger.Error.Println("❌ Timer not started: All referees not connected.")
		//	return
		//}

		// 1) clear old decision
		meetState.JudgeDecisions = make(map[string]string)

		// 2) broadcast a "clearResults" so the Lights page resets its UI
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

		// 3) start the Platform Ready timer
		startPlatformReadyTimer(meetState)

	// not required per Daniel
	//case "stopTimer":
	//	stopPlatformReadyTimer(meetState)

	case "resetTimer":
		resetPlatformReadyTimer(meetState)
		// clear judge decisions on reset if you want
		meetState.JudgeDecisions = make(map[string]string)
		// broadcast 'clearResults' to reset visuals
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

	case "startNextAttemptTimer":
		startNextAttemptTimer(meetState)
	}

	logger.Info.Printf("✅ Timer action processed: %s (meet: %s)", action, meetName)
}

// not required per Daniel
//func isAllRefsConnected(meetState *MeetState) bool {
//	if meetState.RefereeSessions["left"] == nil {
//		return false
//	}
//	if meetState.RefereeSessions["centre"] == nil {
//		return false
//	}
//	if meetState.RefereeSessions["right"] == nil {
//		return false
//	}
//	return true
//}

// startPlatformReadyTimer start/Stop/Reset the Platform Ready Timer
func startPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if meetState.PlatformReadyActive {
		logger.Warn.Println("⚠️ Platform Ready Timer already running.")
		return
	}
	meetState.PlatformReadyActive = true
	meetState.PlatformReadyTimeLeft = 60

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			platformReadyMutex.Lock()
			if !meetState.PlatformReadyActive {
				platformReadyMutex.Unlock()
				return
			}
			meetState.PlatformReadyTimeLeft--
			broadcastTimeUpdateWithIndex("updatePlatformReadyTime", meetState.PlatformReadyTimeLeft, 0)
			if meetState.PlatformReadyTimeLeft <= 0 {
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				meetState.PlatformReadyActive = false
				meetState.PlatformReadyTimeLeft = 60
				platformReadyMutex.Unlock()
				return
			}
			platformReadyMutex.Unlock()
		}
	}()
}

// no longer required per Daniel
// stopPlatformReadyTimer stops the Platform Ready Timer
//func stopPlatformReadyTimer(meetState *MeetState) {
//	platformReadyMutex.Lock()
//	defer platformReadyMutex.Unlock()
//	meetState.PlatformReadyActive = false
//	meetState.PlatformReadyTimeLeft = 60
//}

// resetPlatformReadyTimer resets the Platform Ready Timer
func resetPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		logger.Warn.Println("⚠️ No active timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// track an incrementing ID so each new timer gets a unique ID
var nextAttemptIDCounter int

// todo: clear the timer after it hits 0
// startNextAttemptTimer is a struct for tracking the next attempt timer
func startNextAttemptTimer(meetState *MeetState) {
	nextAttemptMutex.Lock()
	nextAttemptIDCounter++
	timerID := nextAttemptIDCounter

	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: 60,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	nextAttemptMutex.Unlock()

	ticker := time.NewTicker(1 * time.Second)
	go func(id int) {
		defer ticker.Stop()
		for range ticker.C {
			nextAttemptMutex.Lock()

			// 1) locate the timer by ID
			idx := findTimerIndex(meetState.NextAttemptTimers, id)
			if idx == -1 {
				// timer was removed or doesn't exist
				nextAttemptMutex.Unlock()
				return
			}

			// 2) if it's inactive, just exit
			if !meetState.NextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}

			// 3) decrement time
			meetState.NextAttemptTimers[idx].TimeLeft--

			// 4) re-broadcast indexes. We compute a fresh "display index" for each timer:
			//    e.g., if we have 3 timers left, they become #1, #2, #3 in the order they appear.
			//    So we do a separate function to broadcast them all after each second.
			broadcastAllNextAttemptTimers(meetState.NextAttemptTimers)

			// 5) check if it reached 0
			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
				// mark it inactive (or remove it completely)
				meetState.NextAttemptTimers[idx].Active = false

				// Calculate display index for the expired timer (array index + 1)
				expiredDisplayIndex := idx + 1
				// Broadcast an expiration message with the correct display index
				broadcastTimeUpdateWithIndex("nextAttemptExpired", 0, expiredDisplayIndex)

				// remove this timer from the slice
				meetState.NextAttemptTimers = removeTimerByIndex(meetState.NextAttemptTimers, idx)

				// re-broadcast again so the display indexes reset now that this timer is gone
				broadcastAllNextAttemptTimers(meetState.NextAttemptTimers)

				nextAttemptMutex.Unlock()
				return
			}
			nextAttemptMutex.Unlock()
		}
	}(timerID)
}

// findTimerIndex returns the index of the timer with the given ID, or -1 if not found.
func findTimerIndex(timers []NextAttemptTimer, id int) int {
	for i, t := range timers {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// removeTimerByIndex removes the timer at [idx] from the slice
func removeTimerByIndex(timers []NextAttemptTimer, idx int) []NextAttemptTimer {
	return append(timers[:idx], timers[idx+1:]...)
}

// broadcastAllNextAttemptTimers re-broadcasts the TimeLeft for every active timer, computing a fresh "display index" for each in ascending order.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer) {
	for i, t := range timers {
		// i is zero-based, so for display we do i+1
		if t.Active {
			broadcastTimeUpdateWithIndex("updateNextAttemptTime", t.TimeLeft, i+1)
		}
	}
}

// broadcastTimeUpdateWithIndex sends a message to all clients with a time update,
// including a display index so the client can map the update to the correct timer.
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index, // Used by the client to update the correct timer row
	})
	broadcast <- msg
}

// UTILITIES

// broadcastRefereeHealth sends message to all clients w/ current ref health
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

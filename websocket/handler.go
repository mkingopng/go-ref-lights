// Package websocket: contains the WebSocket handler and related functions
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GLOBALS

// clients tracks all connected clients (for broadcast usage)
var clients = make(map[*websocket.Conn]bool)

// Global mutex to synchronize writes
var writeMutex sync.Mutex

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// Store info about which meet & judge is associated with each connection.
// This helps us remove them from the correct meet's state on disconnection.
type connectionInfo struct {
	meetName string
	judgeID  string
}

// connectionMapping maps each WebSocket conn -> (meetName, judgeID)
var connectionMapping = make(map[*websocket.Conn]connectionInfo)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// Mutexes for concurrency around timers
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

// Function that writes messages to WebSocket
func safeWriteMessage(conn *websocket.Conn, messageType int, data []byte) error {
	writeMutex.Lock()         // Acquire lock
	defer writeMutex.Unlock() // Release lock after writing

	return conn.WriteMessage(messageType, data)
}

// ServeWs is our main WebSocket entry point
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	log.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	// Track globally
	clients[conn] = true

	// If there's no meetName query param, fall back to "DEFAULT_MEET"
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		meetName = "DEFAULT_MEET"
	}

	// Start the heartbeat (pings) in the background
	go startHeartbeat(conn)

	// Start reading messages from this connection
	go handleReads(conn, meetName)
}

// GLOBAL BROADCAST LOOP

// HandleMessages listens for messages on the broadcast channel and sends them to all clients
func HandleMessages() {
	for {
		msg := <-broadcast
		for conn := range clients {
			// Use the thread-safe write function
			if err := safeWriteMessage(conn, websocket.TextMessage, msg); err != nil {
				log.Printf("⚠️ WriteMessage error: %v", err)
				_ = conn.Close()
				delete(clients, conn)
				// Also remove from connectionMapping if needed
				if info, ok := connectionMapping[conn]; ok {
					meetState := getMeetState(info.meetName)
					if meetState.RefereeSessions[info.judgeID] == conn {
						meetState.RefereeSessions[info.judgeID] = nil
					}
					delete(connectionMapping, conn)
					// Optionally broadcast updated health for that meet
					broadcastRefereeHealth(meetState)
				}
			}
		}
	}
}

// MESSAGE READING & DISCONNECTION HANDLING

// handleReads reads messages from a connection and processes them
func handleReads(conn *websocket.Conn, defaultMeetName string) {
	defer func() {
		// On disconnection, remove from global clients map
		log.Printf("⚠️ WebSocket disconnected: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)

		// If we know which meet/judge this conn belonged to, remove from that meet's state
		if info, ok := connectionMapping[conn]; ok {
			meetState := getMeetState(info.meetName)
			if meetState.RefereeSessions[info.judgeID] == conn {
				meetState.RefereeSessions[info.judgeID] = nil
				log.Printf("🚪 Removing %s from meet %s (due to disconnect)", info.judgeID, info.meetName)
				// Broadcast updated health for that meet
				broadcastRefereeHealth(meetState)
			}
			delete(connectionMapping, conn)
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("⚠️ WebSocket read error: %v", err)
			return
		}

		var decisionMsg DecisionMessage
		if err := json.Unmarshal(msg, &decisionMsg); err != nil {
			log.Printf("⚠️ Invalid JSON: %v", err)
			continue
		}

		// If the JSON has no meetName, fallback to the default
		if decisionMsg.MeetName == "" {
			decisionMsg.MeetName = defaultMeetName
		}

		// Distinguish between "actions" vs. "decisions"
		switch decisionMsg.Action {
		case "registerRef":
			// store the connection, broadcast health
			registerRef(decisionMsg, conn)
		case "startTimer", "stopTimer", "resetTimer", "startNextAttemptTimer":
			handleTimerAction(decisionMsg.Action, decisionMsg.MeetName)
		default:
			// if there's no recognized Action, we treat it as a Decision
			processDecision(decisionMsg, conn)
		}
	}
}

// registerRef marks a referee as connected in the meetState
func registerRef(msg DecisionMessage, conn *websocket.Conn) {
	if msg.JudgeID == "" {
		// If we have no judgeId, do nothing
		return
	}

	meetState := getMeetState(msg.MeetName)
	// If there's an existing session for that judge, close it out
	existingConn, exists := meetState.RefereeSessions[msg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		log.Printf("🔴 Kicking out old session for ref: %s in meet: %s", msg.JudgeID, msg.MeetName)
		_ = existingConn.Close()
		delete(clients, existingConn)
	}

	// Store the new conn
	meetState.RefereeSessions[msg.JudgeID] = conn

	// Also update global mapping
	connectionMapping[conn] = connectionInfo{
		meetName: msg.MeetName,
		judgeID:  msg.JudgeID,
	}

	log.Printf("✅ Referee %s registered (meet: %s)", msg.JudgeID, msg.MeetName)
	// Broadcast updated health
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
			log.Printf("⚠️ WebSocket ping failed (%d/3): %v", failedPings, err)
			if failedPings >= 3 {
				log.Println("❌ Connection lost due to repeated ping failures.")
				_ = conn.Close()
				delete(clients, conn)

				// Also remove from connectionMapping if needed
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
		return
	}

	// Single-session enforcement for this judge in this meet
	existingConn, exists := meetState.RefereeSessions[decisionMsg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		log.Printf("🔴 Kicking out old session for referee: %s in meet: %s",
			decisionMsg.JudgeID, decisionMsg.MeetName)
		_ = existingConn.Close()
		delete(clients, existingConn) // remove from global set
		meetState.RefereeSessions[decisionMsg.JudgeID] = nil

		// Also remove from connectionMapping
		if info, ok := connectionMapping[existingConn]; ok {
			if info.judgeID == decisionMsg.JudgeID && info.meetName == decisionMsg.MeetName {
				delete(connectionMapping, existingConn)
			}
		}
	}

	// Store the new connection in meet state
	meetState.RefereeSessions[decisionMsg.JudgeID] = conn
	meetState.JudgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision

	// Update global mapping so we know this conn -> meet/judge
	connectionMapping[conn] = connectionInfo{
		meetName: decisionMsg.MeetName,
		judgeID:  decisionMsg.JudgeID,
	}

	log.Printf("✅ Decision from %s: %s (meet: %s)",
		decisionMsg.JudgeID, decisionMsg.Decision, decisionMsg.MeetName)

	// Let everyone know a judge submitted
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(submission)
	broadcast <- subMsg

	// Also broadcast updated health (who is connected) for this meet
	broadcastRefereeHealth(meetState)

	// If we have decisions from all 3 judges, broadcast final results
	if len(meetState.JudgeDecisions) == 3 {
		broadcastFinalResults(decisionMsg.MeetName)
	}
}

func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	// 1) Broadcast the final decisions
	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centreDecision": meetState.JudgeDecisions["centre"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// 2) Immediately start the next-lifter timer
	startNextAttemptTimer(meetState)

	// 3) Clear the results after a set duration
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// 4) Reset for next lift
	meetState.JudgeDecisions = make(map[string]string)
}

// TIMER / PLATFORM READY

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	meetState := getMeetState(meetName)

	switch action {
	case "startTimer":
		// only allow "Platform Ready" if all refs are connected
		if !isAllRefsConnected(meetState) {
			errMsg := map[string]string{
				"action":  "healthError",
				"message": "Cannot start timer: Not all referees are connected!",
			}
			msg, _ := json.Marshal(errMsg)
			broadcast <- msg
			log.Println("❌ Timer not started: All referees not connected.")
			return
		}
		// 1) Clear old decisions
		meetState.JudgeDecisions = make(map[string]string)

		// 2) Broadcast a "clearResults" so the Lights page resets its UI
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

		// 3) Now start the Platform Ready timer
		startPlatformReadyTimer(meetState)

	case "stopTimer":
		stopPlatformReadyTimer(meetState)

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

	log.Printf("✅ Timer action processed: %s (meet: %s)", action, meetName)
}

func isAllRefsConnected(meetState *MeetState) bool {
	if meetState.RefereeSessions["left"] == nil {
		return false
	}
	if meetState.RefereeSessions["centre"] == nil {
		return false
	}
	if meetState.RefereeSessions["right"] == nil {
		return false
	}
	return true
}

// startPlatformReadyTimer start/Stop/Reset the Platform Ready Timer
func startPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if meetState.PlatformReadyActive {
		log.Println("⚠️ Platform Ready Timer already running.")
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

// stopPlatformReadyTimer stops the Platform Ready Timer
func stopPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// resetPlatformReadyTimer resets the Platform Ready Timer
func resetPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		log.Println("⚠️ No active timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// NEXT ATTEMPT TIMER

// NextAttemptTimer is a struct for tracking the next attempt timer
func startNextAttemptTimer(meetState *MeetState) {
	nextAttemptMutex.Lock()
	defer nextAttemptMutex.Unlock()

	newTimer := NextAttemptTimer{
		TimeLeft: 60,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	idx := len(meetState.NextAttemptTimers) - 1

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			nextAttemptMutex.Lock()
			if !meetState.NextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}
			meetState.NextAttemptTimers[idx].TimeLeft--
			broadcastTimeUpdateWithIndex("updateNextAttemptTime",
				meetState.NextAttemptTimers[idx].TimeLeft,
				idx,
			)
			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
				broadcast <- []byte(`{"action":"nextAttemptExpired"}`)
				meetState.NextAttemptTimers[idx].Active = false
				nextAttemptMutex.Unlock()
				return
			}
			nextAttemptMutex.Unlock()
		}
	}()
}

// UTILITIES

// broadcastRefereeHealth sends a message to all clients with the current referee health
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

// broadcastTimeUpdate sends a message to all clients with a time update
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index, // <--- so the client knows which timer
	})
	broadcast <- msg
}

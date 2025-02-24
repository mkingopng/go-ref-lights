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

// GLOBALS NEEDED FOR ALL WEBSOCKETS (NOT SPECIFIC TO A SINGLE MEET)

// clients tracks all connected clients (for broadCast usage)
var clients = make(map[*websocket.Conn]bool)

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// HandleMessages for the global broadcast loop
func HandleMessages() {
	for {
		msg := <-broadcast
		for conn := range clients {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("⚠️ WriteMessage error: %v", err)
				_ = conn.Close()
				delete(clients, conn)
			}
		}
	}
}

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 30
var platformReadyMutex = &sync.Mutex{}
var nextAttemptMutex = &sync.Mutex{}

// upgrader allows us to create WebSocket connections
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

// ServeWs websocket entry point
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	log.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	// track globally
	clients[conn] = true

	// e.g. ws://.../referee-updates?meetName=STATE_CHAMPS_2025
	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		meetName = "DEFAULT_MEET"
	}

	// start the heartbeat (pings) in background
	go startHeartbeat(conn)

	// start reading messages
	go handleReads(conn, meetName)
}

// READING MESSAGES

// handleReads reads incoming messages from a WebSocket connection
func handleReads(conn *websocket.Conn, defaultMeetName string) {
	defer func() {
		log.Printf("⚠️ WebSocket disconnected: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)
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

		// if the incoming JSON has no meetName, fallback to the param
		if decisionMsg.MeetName == "" {
			decisionMsg.MeetName = defaultMeetName
		}

		// if the JSON has an Action, it might be a timer request
		if decisionMsg.Action != "" {
			handleTimerAction(decisionMsg.Action, decisionMsg.MeetName)
		} else {
			processDecision(decisionMsg, conn)
		}
	}
}

// HEARTBEAT

// startHeartbeat sends periodic pings to the client to keep the connection alive
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
				return
			}
		} else {
			failedPings = 0
		}
	}
}

// DECISION HANDLING

// processDecision handles incoming decisions from judges
func processDecision(decisionMsg DecisionMessage, conn *websocket.Conn) {
	meetState := getMeetState(decisionMsg.MeetName)

	// basic validation
	if decisionMsg.JudgeID == "" || decisionMsg.Decision == "" {
		return
	}

	// single session enforcement for this meet
	existingConn, exists := meetState.RefereeSessions[decisionMsg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		log.Printf("🔴 Kicking out old session for referee: %s in meet: %s",
			decisionMsg.JudgeID, decisionMsg.MeetName)
		_ = existingConn.Close()
		delete(clients, existingConn) // This removes them from global too
		meetState.RefereeSessions[decisionMsg.JudgeID] = nil
	}

	meetState.RefereeSessions[decisionMsg.JudgeID] = conn
	meetState.JudgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision
	log.Printf("✅ Decision from %s: %s (meet: %s)",
		decisionMsg.JudgeID, decisionMsg.Decision, decisionMsg.MeetName)

	// let everyone know a judge submitted
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(submission)
	broadcast <- subMsg

	// if we have decisions from all 3 judges, broadcast final results
	if len(meetState.JudgeDecisions) == 3 {
		broadcastFinalResults(decisionMsg.MeetName)
	}
}

// broadcastFinalResults sends the final decisions to all clients
func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centreDecision": meetState.JudgeDecisions["centre"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// reset for next lift
	meetState.JudgeDecisions = make(map[string]string)
}

// TIMER ACTIONS (MEET-AWARE)

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	meetState := getMeetState(meetName)

	switch action {
	case "startTimer":
		// Health check
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
		startPlatformReadyTimer(meetState)

	case "stopTimer":
		stopPlatformReadyTimer(meetState)
	case "resetTimer":
		resetPlatformReadyTimer(meetState)
	case "startNextAttemptTimer":
		startNextAttemptTimer(meetState)
	}
	log.Printf("✅ Timer action processed: %s (meet: %s)", action, meetName)
}

// isAllRefsConnected checks if all 3 referees are connected
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

// startPlatformReadyTimer starts the 60-second timer for Platform Ready
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
			broadcastTimeUpdate("updatePlatformReadyTime", meetState.PlatformReadyTimeLeft)
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

// stopPlatformReadyTimer stops the Platform Ready timer
func stopPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// resetPlatformReadyTimer resets the Platform Ready timer to 60 seconds
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

// startNextAttemptTimer starts a 60-second timer for the next attempt
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
			broadcastTimeUpdate("updateNextAttemptTime", meetState.NextAttemptTimers[idx].TimeLeft)
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

// UTIL

// broadcastTimeUpdate sends a time update to all clients
func broadcastTimeUpdate(action string, timeLeft int) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
	})
	broadcast <- msg
}

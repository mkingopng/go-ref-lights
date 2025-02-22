// Package websocket websocket/handler.go handles the WebSocket connections & messages
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Tracks all connected clients
var clients = make(map[*websocket.Conn]bool)

// Broadcast channel for sending messages
var broadcast = make(chan []byte)

// Single session per position: left, centre, right
var refereeSessions = make(map[string]*websocket.Conn)

// Judge decisions and mutex
var judgeDecisions = make(map[string]string)
var judgeMutex = &sync.Mutex{}

// Platform Ready timer
var platformReadyMutex = &sync.Mutex{}
var platformReadyTimerActive bool
var platformReadyTimeLeft = 60

// NextAttemptTimer structure
type NextAttemptTimer struct {
	TimeLeft int
	Active   bool
}

var nextAttemptMutex = &sync.Mutex{}
var nextAttemptTimers []NextAttemptTimer

// Clear results after 30s
var resultsDisplayDuration = 30

// DecisionMessage from the client
type DecisionMessage struct {
	JudgeID  string `json:"judgeId,omitempty"`
	Decision string `json:"decision,omitempty"`
	Action   string `json:"action,omitempty"`
}

// Only allow certain origins
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// If test mode => allow all
		if r.Header.Get("Test-Mode") == "true" {
			return true
		}
		// Production
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080" || origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// ServeWs upgrades the HTTP connection to a WebSocket connection
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	log.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	clients[conn] = true
	go startHeartbeat(conn)
	go handleReads(conn)
}

// handleReads reads messages from the client
func handleReads(conn *websocket.Conn) {
	defer func() {
		log.Printf("⚠️ WebSocket disconnected: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)

		// If the disconnect is for a known referee, remove from refereeSessions
		removeRefereeConnection(conn)
		// Then broadcast the new referee health to all clients
		broadcastRefereeHealth()
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

		if decisionMsg.Action != "" {
			handleTimerAction(decisionMsg.Action)
		} else {
			processDecision(decisionMsg, conn)
		}
	}
}

// startHeartbeat sends a ping every 10s to the client
func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	failedPings := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failedPings++
			log.Printf("⚠️ WebSocket ping failed (%d/3): %v", failedPings, err)
			if failedPings >= 3 {
				log.Println("❌ Lost connection due to repeated ping failures.")
				_ = conn.Close()
				delete(clients, conn)
				removeRefereeConnection(conn)
				broadcastRefereeHealth()
				return
			}
		} else {
			failedPings = 0
		}
	}
}

// processDecision handles the decision message from the client
func processDecision(decisionMsg DecisionMessage, conn *websocket.Conn) {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	if decisionMsg.JudgeID == "" || decisionMsg.Decision == "" {
		return
	}

	// Single session enforcement
	if existingConn, exists := refereeSessions[decisionMsg.JudgeID]; exists && existingConn != conn {
		log.Printf("🔴 Kicking out old session for referee: %s", decisionMsg.JudgeID)
		_ = existingConn.Close()
		delete(refereeSessions, decisionMsg.JudgeID)
		delete(clients, existingConn)
	}

	refereeSessions[decisionMsg.JudgeID] = conn
	judgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision
	log.Printf("✅ Decision from %s: %s", decisionMsg.JudgeID, decisionMsg.Decision)

	// Notify clients that a judge submitted
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(submission)
	broadcast <- subMsg

	// Broadcast new referee health (in case a new judge connected)
	broadcastRefereeHealth()

	// If we have all 3 decisions, broadcast the final result
	if len(judgeDecisions) == 3 {
		broadcastFinalResults()
	}
}

// broadcastFinalResults sends the final results to all clients
func broadcastFinalResults() {
	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   judgeDecisions["left"],
		"centreDecision": judgeDecisions["centre"],
		"rightDecision":  judgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// Clear results after 30s
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// Reset for next lift
	judgeDecisions = make(map[string]string)
}

// handleTimerAction processes the timer actions
func handleTimerAction(action string) {
	switch action {
	case "startTimer":
		// Health check before allowing the timer to start
		if !isAllRefsConnected() {
			// Broadcast a healthError
			errMsg := map[string]string{
				"action":  "healthError",
				"message": "Cannot start timer: Not all referees are connected!",
			}
			msg, _ := json.Marshal(errMsg)
			broadcast <- msg

			log.Println("❌ Timer not started: All referees not connected.")
			return
		}
		// If all refs are connected, start
		startPlatformReadyTimer()

	case "stopTimer":
		stopPlatformReadyTimer()
	case "resetTimer":
		resetPlatformReadyTimer()
	case "startNextAttemptTimer":
		startNextAttemptTimer()
	}
	log.Printf("✅ Timer action processed: %s", action)
}

// -------------------- Health Check Helpers --------------------

// isAllRefsConnected returns true if "left", "centre", and "right" are present in refereeSessions
func isAllRefsConnected() bool {
	// Acquire judgeMutex if you want to ensure thread safety
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	// Check exactly the keys "left", "centre", "right" exist
	required := []string{"left", "centre", "right"}
	for _, key := range required {
		if refereeSessions[key] == nil {
			return false
		}
	}
	return true
}

// broadcastRefereeHealth notifies clients how many referees are connected
func broadcastRefereeHealth() {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	// Count how many of the three positions have an active connection
	refs := 0
	if refereeSessions["left"] != nil {
		refs++
	}
	if refereeSessions["centre"] != nil {
		refs++
	}
	if refereeSessions["right"] != nil {
		refs++
	}

	data := map[string]interface{}{
		"action":            "refereeHealth",
		"connectedReferees": refs,
		"requiredReferees":  3,
	}
	msg, _ := json.Marshal(data)
	broadcast <- msg
}

// removeRefereeConnection removes the given conn from refereeSessions if it matches
func removeRefereeConnection(conn *websocket.Conn) {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	// Find which judge was using this conn
	for jID, c := range refereeSessions {
		if c == conn {
			delete(refereeSessions, jID)
			log.Printf("ℹ️ Removed referee session for judge: %s", jID)
			break
		}
	}
}

// startPlatformReadyTimer starts a 60s timer for the platform ready
func startPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if platformReadyTimerActive {
		log.Println("⚠️ Platform Ready Timer already running.")
		return
	}
	platformReadyTimerActive = true
	platformReadyTimeLeft = 60

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			platformReadyMutex.Lock()
			if !platformReadyTimerActive {
				platformReadyMutex.Unlock()
				return
			}

			platformReadyTimeLeft--
			broadcastTimeUpdate("updatePlatformReadyTime", platformReadyTimeLeft)

			if platformReadyTimeLeft <= 0 {
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				platformReadyTimerActive = false
				platformReadyTimeLeft = 60
				platformReadyMutex.Unlock()
				return
			}
			platformReadyMutex.Unlock()
		}
	}()
}

// stopPlatformReadyTimer stops the current timer
func stopPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	platformReadyTimerActive = false
	platformReadyTimeLeft = 60
}

// resetPlatformReadyTimer resets the timer to 60s
func resetPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !platformReadyTimerActive {
		log.Println("⚠️ No active timer to reset.")
		return
	}
	platformReadyTimerActive = false
	platformReadyTimeLeft = 60
}

// startNextAttemptTimer starts a 60s timer for the next attempt
func startNextAttemptTimer() {
	nextAttemptMutex.Lock()
	defer nextAttemptMutex.Unlock()

	timer := NextAttemptTimer{
		TimeLeft: 60,
		Active:   true,
	}
	nextAttemptTimers = append(nextAttemptTimers, timer)
	idx := len(nextAttemptTimers) - 1

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			nextAttemptMutex.Lock()
			if !nextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}

			nextAttemptTimers[idx].TimeLeft--
			broadcastTimeUpdate("updateNextAttemptTime", nextAttemptTimers[idx].TimeLeft)

			if nextAttemptTimers[idx].TimeLeft <= 0 {
				broadcast <- []byte(`{"action":"nextAttemptExpired"}`)
				nextAttemptTimers[idx].Active = false
				nextAttemptMutex.Unlock()
				return
			}
			nextAttemptMutex.Unlock()
		}
	}()
}

// broadcastTimeUpdate sends a message to all clients with the current timeLeft
func broadcastTimeUpdate(action string, timeLeft int) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
	})
	broadcast <- msg
}

// HandleMessages The main broadcast loop
func HandleMessages() {
	for {
		msg := <-broadcast
		for conn := range clients {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("⚠️ WriteMessage error: %v", err)
				_ = conn.Close()
				delete(clients, conn)
				removeRefereeConnection(conn)
				broadcastRefereeHealth()
			}
		}
	}
}

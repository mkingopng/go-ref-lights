// Package websocket websocket/handler.go
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Global Variables
// Tracks all connected clients (conn -> true/false)
var clients = make(map[*websocket.Conn]bool)

// Broadcast channel for sending messages to all clients
var broadcast = make(chan []byte)

// Tracks each judge's last decision
var judgeDecisions = make(map[string]string)
var judgeMutex = &sync.Mutex{}

// Timers for "platform ready"
var platformReadyTimerActive bool
var platformReadyTimeLeft int
var platformReadyMutex = &sync.Mutex{}

// Timers for "next attempt"
var nextAttemptMutex = &sync.Mutex{}
var nextAttemptTimerActive bool
var nextAttemptTimeLeft int

// Results display for 30 seconds, then cleared
var resultsDisplayDuration = 30

// Tracks current WebSocket connections for each referee (left, centre, right)
var refereeSessions = make(map[string]*websocket.Conn)

// DecisionMessage is the JSON payload for decisions or actions
type DecisionMessage struct {
	JudgeID  string `json:"judgeId,omitempty"`
	Decision string `json:"decision,omitempty"`
	Action   string `json:"action,omitempty"`
}

// Upgrader config — only allow certain origins
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in test mode
		if r.Header.Get("Test-Mode") == "true" {
			return true
		}
		// Production restriction
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080" || origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// ServeWs - Upgrades HTTP to WebSocket
func ServeWs(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	log.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	// Track this client
	clients[conn] = true

	// Start a heartbeat (ping) goroutine
	go startHeartbeat(conn)

	// Start a goroutine to read incoming messages
	go handleReads(conn)
}

// handleReads - Reads messages from the client until error/close
func handleReads(conn *websocket.Conn) {
	defer func() {
		// clean-up on disconnect
		log.Printf("⚠️ WebSocket disconnected: %v", conn.RemoteAddr())
		_ = conn.Close()
		delete(clients, conn)
	}()

	for {
		// Read the next message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("⚠️ WebSocket read error: %v", err)
			return // exit the loop, triggers the deferring
		}

		// Unmarshal JSON
		var decisionMsg DecisionMessage
		if err := json.Unmarshal(msg, &decisionMsg); err != nil {
			log.Printf("⚠️ Invalid JSON: %v", err)
			continue
		}

		// Process either a decision or an action
		if decisionMsg.Action != "" {
			handleTimerAction(decisionMsg.Action)
		} else {
			processDecision(decisionMsg, conn)
		}
	}
}

// startHeartbeat - Periodically sends ping frames
func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	failedPings := 0

	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failedPings++
			log.Printf("⚠️ WebSocket ping failed (%d/3): %v", failedPings, err)
			if failedPings >= 3 {
				log.Println("❌ WebSocket connection lost due to repeated ping failures.")
				_ = conn.Close()
				delete(clients, conn)
				return
			}
		} else {
			failedPings = 0
		}
	}
}

// processDecision - Store and broadcast judge decisions
func processDecision(decisionMsg DecisionMessage, conn *websocket.Conn) {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	if decisionMsg.JudgeID == "" || decisionMsg.Decision == "" {
		// If there's no judge or decision, ignore
		return
	}

	// Kick out any old session for that judge
	if existingConn, exists := refereeSessions[decisionMsg.JudgeID]; exists {
		log.Printf("🔴 Kicking out old session for referee: %s", decisionMsg.JudgeID)
		_ = existingConn.Close() // ignore errors
		delete(refereeSessions, decisionMsg.JudgeID)
		delete(clients, existingConn)
	}

	// Assign the new session
	refereeSessions[decisionMsg.JudgeID] = conn

	// Record decision
	judgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision
	log.Printf("✅ Received decision from %s: %s", decisionMsg.JudgeID, decisionMsg.Decision)

	// Let others know a judge has submitted
	submissionUpdate := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	submissionMsg, _ := json.Marshal(submissionUpdate)
	broadcast <- submissionMsg

	// If we have decisions from all three judges, broadcast
	if len(judgeDecisions) == 3 {
		broadcastFinalResults()
	}
}

// broadcastFinalResults - Send final 3-judge result
func broadcastFinalResults() {
	whiteCount, redCount := 0, 0
	for _, d := range judgeDecisions {
		if d == "white" {
			whiteCount++
		} else if d == "red" {
			redCount++
		}
	}

	// Send final decisions
	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   judgeDecisions["left"],
		"centreDecision": judgeDecisions["centre"],
		"rightDecision":  judgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// Clear results after 30 seconds
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearMsgJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearMsgJSON
	}()

	// Reset judge decisions
	judgeDecisions = make(map[string]string)
}

// handleTimerAction - Called when user sends "action"
func handleTimerAction(action string) {
	switch action {
	case "startTimer":
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

// Timer Functions for Next Attempt
func startNextAttemptTimer() {
	nextAttemptMutex.Lock()
	defer nextAttemptMutex.Unlock()

	// Stop any old timer before starting a new one
	if nextAttemptTimerActive {
		log.Println("⚠️ Stopping previous next attempt timer before starting a new one")
		nextAttemptTimerActive = false
	}

	nextAttemptTimerActive = true
	nextAttemptTimeLeft = 60

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			nextAttemptMutex.Lock()
			if !nextAttemptTimerActive {
				nextAttemptMutex.Unlock()
				return
			}

			nextAttemptTimeLeft--
			if nextAttemptTimeLeft <= 0 {
				broadcast <- []byte(`{"action":"nextAttemptExpired"}`)
				nextAttemptTimerActive = false
				nextAttemptMutex.Unlock()
				return
			}

			// Broadcast time update
			updateMsg, _ := json.Marshal(map[string]interface{}{
				"action":   "updateNextAttemptTime",
				"timeLeft": nextAttemptTimeLeft,
			})
			broadcast <- updateMsg

			nextAttemptMutex.Unlock()
		}
	}()
}

// Timer Functions for Platform Ready
func startPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if platformReadyTimerActive {
		log.Println("⚠️ Platform Ready Timer already running. Ignoring duplicate start.")
		return
	}

	platformReadyTimerActive = true
	platformReadyTimeLeft = 60

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			platformReadyMutex.Lock()
			if !platformReadyTimerActive {
				platformReadyMutex.Unlock()
				return
			}

			platformReadyTimeLeft--
			if platformReadyTimeLeft <= 0 {
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				platformReadyTimerActive = false
				platformReadyMutex.Unlock()
				return
			}

			// Broadcast time update
			updateMsg, _ := json.Marshal(map[string]interface{}{
				"action":   "updatePlatformReadyTime",
				"timeLeft": platformReadyTimeLeft,
			})
			broadcast <- updateMsg

			platformReadyMutex.Unlock()
		}
	}()
}

func stopPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()
	platformReadyTimerActive = false
}

func resetPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !platformReadyTimerActive {
		log.Println("⚠️ No active timer to reset")
		return
	}
	platformReadyTimerActive = false
	platformReadyTimeLeft = 60
}

// HandleMessages - The broadcast loop
func HandleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("⚠️ WebSocket write error: %v", err)
				_ = client.Close()
				delete(clients, client)
			}
		}
	}
}

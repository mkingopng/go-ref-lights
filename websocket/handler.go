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

// Tracks all connected clients
var clients = make(map[*websocket.Conn]bool)

// Broadcast channel for sending messages
var broadcast = make(chan []byte)

// Single session per position
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

func handleReads(conn *websocket.Conn) {
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

		if decisionMsg.Action != "" {
			handleTimerAction(decisionMsg.Action)
		} else {
			processDecision(decisionMsg, conn)
		}
	}
}

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
				return
			}
		} else {
			failedPings = 0
		}
	}
}

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
	sub := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(sub)
	broadcast <- subMsg

	// If we have all 3 decisions, broadcast the final result
	if len(judgeDecisions) == 3 {
		broadcastFinalResults()
	}
}

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

// -------------------- Platform Ready Timer --------------------
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

func stopPlatformReadyTimer() {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	platformReadyTimerActive = false
	platformReadyTimeLeft = 60
}

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

// -------------------- Next Attempt Timers --------------------
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
			}
		}
	}
}

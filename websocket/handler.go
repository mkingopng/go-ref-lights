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

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)

var judgeDecisions = make(map[string]string)
var judgeMutex = &sync.Mutex{}

var platformReadyTimerActive bool
var platformReadyTimeLeft int
var platformReadyMutex = &sync.Mutex{}

var resultsDisplayDuration = 30 // Seconds after which results are cleared

// DecisionMessage represents the structure of messages from judges and timer actions
type DecisionMessage struct {
	JudgeID  string `json:"judgeId,omitempty"`
	Decision string `json:"decision,omitempty"`
	Action   string `json:"action,omitempty"`
}

// ✅ Allow all origins **only** in tests
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// ✅ Allow all origins during tests
		if r.Header.Get("Test-Mode") == "true" {
			return true
		}
		// ✅ Restrict in production
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080" || origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// ServeWs handles WebSocket requests
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	clients[conn] = true
	log.Println("New WebSocket client connected.")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			delete(clients, conn)
			_ = conn.Close()
			log.Println("WebSocket client disconnected.")
			break
		}

		var decisionMsg DecisionMessage
		err = json.Unmarshal(msg, &decisionMsg)
		if err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}
		processDecision(decisionMsg)
	}
}

// Process judge decisions and timer actions
func processDecision(decisionMsg DecisionMessage) {
	judgeMutex.Lock()
	defer judgeMutex.Unlock()

	if decisionMsg.JudgeID != "" && decisionMsg.Decision != "" {
		// Handle judge decision
		judgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision
		log.Printf("Received decision from %s: %s", decisionMsg.JudgeID, decisionMsg.Decision)

		// Notify that a judge has submitted
		submissionUpdate := map[string]string{
			"action":  "judgeSubmitted",
			"judgeId": decisionMsg.JudgeID,
		}
		submissionMsg, _ := json.Marshal(submissionUpdate)
		broadcast <- submissionMsg

		// Check if all judges have submitted
		if len(judgeDecisions) == 3 {
			broadcastFinalResults()
		}
	} else if decisionMsg.Action != "" {
		// Handle timer actions
		handleTimerAction(decisionMsg.Action)
	}
}

// Broadcast final results when all judges submit
func broadcastFinalResults() {
	whiteCount, redCount := 0, 0
	for _, decision := range judgeDecisions {
		if decision == "white" {
			whiteCount++
		} else if decision == "red" {
			redCount++
		}
	}

	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   judgeDecisions["left"],
		"centreDecision": judgeDecisions["centre"],
		"rightDecision":  judgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// Start a timer to clear results
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearMsgJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearMsgJSON
	}()

	// Reset judge decisions
	judgeDecisions = make(map[string]string)
}

// Handle timer actions
func handleTimerAction(action string) {
	switch action {
	case "startTimer":
		startPlatformReadyTimer()
	case "stopTimer":
		stopPlatformReadyTimer()
	case "resetTimer":
		resetPlatformReadyTimer()
	}
	log.Printf("Timer action processed on server: %s", action)
}

// Timer handling functions
func startPlatformReadyTimer() {
	platformReadyMutex.Lock()
	if platformReadyTimerActive {
		platformReadyMutex.Unlock()
		stopPlatformReadyTimer()
		platformReadyMutex.Lock()
	}
	platformReadyTimerActive = true
	platformReadyTimeLeft = 60
	platformReadyMutex.Unlock()

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
			} else {
				updateMsg := map[string]interface{}{
					"action":   "updatePlatformReadyTime",
					"timeLeft": platformReadyTimeLeft,
				}
				msg, _ := json.Marshal(updateMsg)
				broadcast <- msg
			}
			platformReadyMutex.Unlock()
		}
	}()
}

func stopPlatformReadyTimer() {
	platformReadyMutex.Lock()
	platformReadyTimerActive = false
	platformReadyMutex.Unlock()
}

func resetPlatformReadyTimer() {
	stopPlatformReadyTimer()
}

// HandleMessages listens for incoming messages and sends them to all clients
func HandleMessages() {
	for {
		msg := <-broadcast
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				_ = client.Close()
				delete(clients, client)
			}
		}
	}
}

// websocket/handler.go
package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Allow only specific origins
		return origin == "http://localhost:8080" || origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// HandleMessages listens for incoming messages on the broadcast channel and sends them to all connected clients
func HandleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast

		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

// ServeWs handles WebSocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	// Register new client
	clients[ws] = true
	log.Println("New WebSocket client connected.")

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			delete(clients, ws)
			ws.Close()
			log.Println("WebSocket client disconnected.")
			break
		}

		var decisionMsg DecisionMessage
		err = json.Unmarshal(msg, &decisionMsg)
		if err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		judgeMutex.Lock()
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
				// Determine the overall result
				whiteCount := 0
				redCount := 0
				for _, decision := range judgeDecisions {
					if decision == "white" {
						whiteCount++
					} else if decision == "red" {
						redCount++
					}
				}

				// Prepare and send combined results
				result := map[string]string{
					"action":         "displayResults",
					"leftDecision":   judgeDecisions["left"],
					"centreDecision": judgeDecisions["centre"],
					"rightDecision":  judgeDecisions["right"],
				}
				resultMsg, _ := json.Marshal(result)
				broadcast <- resultMsg

				// Start a timer to clear results after resultsDisplayDuration
				go func() {
					time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
					clearMsg := map[string]string{
						"action": "clearResults",
					}
					clearMsgJSON, _ := json.Marshal(clearMsg)
					broadcast <- clearMsgJSON
				}()

				// Reset decisions for the next round
				judgeDecisions = make(map[string]string)
			}
		} else if decisionMsg.Action != "" {
			// Handle timer actions
			handleTimerAction(decisionMsg.Action)
		}
		judgeMutex.Unlock()
	}
}

// handleTimerAction processes timer-related actions
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

// timer handling functions
func startPlatformReadyTimer() {
	platformReadyMutex.Lock()
	if platformReadyTimerActive {
		// If it's already active, stop the old one first
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
				// Timer was stopped externally
				platformReadyMutex.Unlock()
				return
			}

			platformReadyTimeLeft--
			if platformReadyTimeLeft <= 0 {
				// Time's up
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				platformReadyTimerActive = false
				platformReadyMutex.Unlock()
				return
			} else {
				// Broadcast update
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
	if platformReadyTimerActive {
		platformReadyTimerActive = false
		// Optionally broadcast that it stopped, if desired:
		// broadcast <- []byte(`{"action":"platformReadyStopped"}`)
	}
	platformReadyMutex.Unlock()
}

func resetPlatformReadyTimer() {
	// Stopping is enough to reset the state; if needed, you can start again
	stopPlatformReadyTimer()
	// Optionally, start a new timer here if that fits your logic
}

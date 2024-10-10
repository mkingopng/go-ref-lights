// websocket/handler.go

package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"github.com/gorilla/websocket"
)

// data structure to hold the referee's choice
type Decision struct {
	Referee string `json:"referee"`
	Choice  string `json:"choice"`
}

// variables
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)
var judgeDecisions = make(map[string]string)
var judgeMutex = &sync.Mutex{}

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
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	clients[ws] = true
	log.Println("New WebSocket client connected.")

	go handleMessages(ws)
}

func handleMessages(ws *websocket.Conn) {
	defer func() {
		delete(clients, ws)
		ws.Close()
	}()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var decision Decision
		if err := json.Unmarshal(msg, &decision); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		mu.Lock()
		decisions[decision.Referee] = decision.Choice
		log.Printf("Decision received: %s - %s", decision.Referee, decision.Choice)
		if len(decisions) == 3 {
			// All decisions received, broadcast them
			aggregated, err := json.Marshal(decisions)
			if err != nil {
				log.Printf("Error marshalling decisions: %v", err)
			} else {
				log.Printf("Broadcasting aggregated decisions: %s", string(aggregated))
				broadcast <- aggregated
			}
			// Reset decisions for next round
			decisions = make(map[string]string)
		}
		mu.Unlock()
	}
}

// HandleMessages listens for incoming messages on the broadcast channel and sends them to all connected clients
func HandleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				client.Close()
				delete(clients, client)
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
	// Implement timer action handling if necessary
	// For example, you can broadcast timer actions to all clients
	subAction := map[string]string{
		"action": action,
	}
	subActionJSON, _ := json.Marshal(subAction)
	broadcast <- subActionJSON
	log.Printf("Timer action broadcasted: %s", action)
}

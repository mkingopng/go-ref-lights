// websocket/handler.go

package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)
var judgeDecisions = make(map[string]string)
var judgeMutex = &sync.Mutex{}

type DecisionMessage struct {
	JudgeID  string `json:"judgeId"`
	Decision string `json:"decision"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	clients[ws] = true

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			delete(clients, ws)
			ws.Close()
			break
		}

		var decisionMsg DecisionMessage
		err = json.Unmarshal(msg, &decisionMsg)
		if err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		judgeMutex.Lock()
		judgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision

		// Notify that a judge has submitted
		submissionUpdate := map[string]string{
			"action":  "judgeSubmitted",
			"judgeId": decisionMsg.JudgeID,
		}
		submissionMsg, _ := json.Marshal(submissionUpdate)
		broadcastToClients(submissionMsg)

		// Check if all judges have submitted
		if len(judgeDecisions) == 3 {
			// Prepare and send combined results
			result := map[string]string{
				"action":         "displayResults",
				"leftDecision":   judgeDecisions["left"],
				"centreDecision": judgeDecisions["centre"],
				"rightDecision":  judgeDecisions["right"],
			}
			resultMsg, _ := json.Marshal(result)
			broadcastToClients(resultMsg)

			// Reset decisions for the next round
			judgeDecisions = make(map[string]string)
		}
		judgeMutex.Unlock()
	}
}

func broadcastToClients(message []byte) {
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

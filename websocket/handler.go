// websocket/handler.go

package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Decision struct {
	Referee string `json:"referee"`
	Choice  string `json:"choice"`
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan []byte)
var decisions = make(map[string]string) // Stores decisions keyed by referee
var mu sync.Mutex                       // Mutex to protect the decisions map

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ServeWs handles WebSocket requests from the peer.
func ServeWs(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	clients[ws] = true

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
		}
	}
}

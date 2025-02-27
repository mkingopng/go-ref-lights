// Package websocket websocket/heartbeat.go
package websocket

import (
	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
	"time"
)

// startHeartbeat sends a ping every 10 seconds to keep the connection alive
func startHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	failedPings := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failedPings++
			logger.Warn.Printf("⚠️ WebSocket ping failed (%d/3): %v", failedPings, err)
			if failedPings >= 5 {
				logger.Error.Println("❌ Connection lost due to repeated ping failures.")
				_ = conn.Close()
				delete(clients, conn)

				// also remove from connectionMapping if needed
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

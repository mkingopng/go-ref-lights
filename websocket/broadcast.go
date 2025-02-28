// Package websocket: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
	"time"
)

// HandleMessages copy clients before iteration
func HandleMessages() {
	for {
		msg := <-broadcast
		// attempt to decode the message to see if it contains a "meetName" field
		var msgMap map[string]interface{}
		err := json.Unmarshal(msg, &msgMap)
		// if the message isn't JSON or doesn't have a meetName, then we assume its global
		meetFilter := ""
		if err == nil {
			if m, ok := msgMap["meetName"].(string); ok {
				meetFilter = m
			}
		}

		// copy clients under lock
		clientsCopy := make(map[*websocket.Conn]bool)
		writeMutex.Lock()
		for conn, v := range clients {
			clientsCopy[conn] = v
		}
		writeMutex.Unlock()

		// send message only to connections with matching meetName (if meetFiler is set)
		for conn := range clientsCopy {
			if meetFilter != "" {
				if info, exists := connectionMapping[conn]; exists {
					if info.meetName != meetFilter {
						// skip connections that are not in the target meet
						continue
					}
				} else {
					// no connection info? skip it
					continue
				}
			}
			// use safeWriteMessage
			if err := safeWriteMessage(conn, websocket.TextMessage, msg); err != nil {
				logger.Error.Printf("❌ Failed to send broadcast message to %v: %v", conn.RemoteAddr(), err)
			}
		}
	}
}

func BroadcastMessage(meetName string, message map[string]interface{}) {
	// Add logging to use meetName
	logger.Debug.Printf("Broadcasting next attempt timers for meet: %s", meetName)
	msg, _ := json.Marshal(message)
	broadcast <- msg
}

// broadcastFinalResults sends the final decisions to all clients
func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	// broadcast the final decisions
	submission := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(submission)
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' is being sent with Left=%s, center=%s, Right=%s",
		meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])
	broadcast <- resultMsg

	// immediately start the next-lifter timer
	startNextAttemptTimer(meetState)

	// clear results after set duration
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// reset for next lift
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a message to all clients with a time update,
// including a display index so the client can map the update to the correct timer.
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
		"meetName": meetName,
	})
	broadcast <- msg
}

// broadcastAllNextAttemptTimers re-broadcasts the TimeLeft for every active
// timer, computing a fresh "display index" for each in ascending order.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	for i, t := range timers {
		if t.Active {
			broadcastTimeUpdateWithIndex("updateNextAttemptTime", t.TimeLeft, i+1, meetName)
		}
	}
}

func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

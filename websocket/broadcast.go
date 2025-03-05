// Package websocket: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// Allow tests to override the sleep behavior.
var sleepFunc = time.Sleep

// Allow tests to override the function used to get a MeetState.
var getMeetStateFunc = getMeetState

// HandleMessages copy clients before iteration
func HandleMessages() {
	for {
		msg := <-broadcast
		var msgMap map[string]interface{}
		var meetFilter string
		if err := json.Unmarshal(msg, &msgMap); err == nil {
			if m, ok := msgMap["meetName"].(string); ok {
				meetFilter = m
			}
		}

		for c := range connections {
			if meetFilter != "" && c.meetName != meetFilter {
				continue
			}
			select {
			case c.send <- msg:
			default:
				logger.Warn.Printf("Dropping broadcast message for connection %v", c.conn.RemoteAddr())
			}
		}
	}
}

// BroadcastMessage takes a meetName and a message (as a map marshals the message into JSON, sends it to the broadcast channel
func BroadcastMessage(meetName string, message map[string]interface{}) {
	logger.Debug.Printf("Broadcasting next attempt timers for meet: %s", meetName)
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error.Printf("Error marshalling message: %v", err)
		return
	}
	broadcast <- msg
}

// broadcastFinalResults sends the final decisions to all connections in the given meet,
// then starts the next attempt timer, and after a timeout, broadcasts a "clearResults" message.
func broadcastFinalResults(meetName string) {
	meetState := getMeetStateFunc(meetName) // use the injectable version

	submission := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, err := json.Marshal(submission)
	if err != nil {
		logger.Error.Printf("Error marshalling final results message: %v", err)
		return
	}
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' is being sent with Left=%s, center=%s, Right=%s",
		meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])
	broadcast <- resultMsg

	// Start the next attempt timer (this remains as is).
	StartNextAttemptTimer(meetState)

	// Instead of a direct time.Sleep, we use our injected sleepFunc.
	go func() {
		sleepFunc(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			logger.Error.Printf("Error marshalling clearResults: %v", err)
			return
		}
		broadcast <- clearJSON
	}()
	// Clear the JudgeDecisions.
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a time update message with an index to all connections in the meet.
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) {
	msg, err := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
		"meetName": meetName,
	})
	if err != nil {
		logger.Error.Printf("Error marshalling time update: %v", err)
		return
	}
	broadcast <- msg
}

// SendBroadcastMessage is a helper function that sends raw byte data over the broadcast channel.
func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

// broadcastAllNextAttemptTimers iterates over the provided timers and sends each active timer as a JSON message.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	for _, timer := range timers {
		if timer.Active {
			msg, err := json.Marshal(timer)
			if err != nil {
				logger.Error.Printf("Error marshalling timer: %v", err)
				continue
			}
			broadcast <- msg
		}
	}
}

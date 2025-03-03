// Package websocket: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// HandleMessages copy clients before iteration
func HandleMessages() {
	for {
		msg := <-broadcast
		// attempt to decode the message to see if it contains a "meetName" field
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

// broadcastFinalResults sends the final decisions to all connections in the given meet.
// It then starts the next attempt timer and clears the results after a set duration
func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	// prepare the final decision message
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

	// immediately start the next-lifter timer
	startNextAttemptTimer(meetState)

	// clear results after set duration
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		if err != nil {
			logger.Error.Printf("Error marshalling clearResults: %v", err)
			return
		}
		broadcast <- clearJSON
	}()

	// reset for next lift
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

// broadcastAllNextAttemptTimers iterates over active next-attempt timers and broadcasts their timeLeft.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	for i, t := range timers {
		if t.Active {
			broadcastTimeUpdateWithIndex("updateNextAttemptTime", t.TimeLeft, i+1, meetName)
		}
	}
}

// SendBroadcastMessage is a helper function that sends raw byte data over the broadcast channel.
func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

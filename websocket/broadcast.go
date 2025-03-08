// Package websocket handles real-time WebSocket communication between referees and the meet system.
// file: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// Allow tests to override the sleep behaviour.
var sleepFunc = time.Sleep

// Allow tests to override the function used to get a MeetState.
var getMeetStateFunc = getMeetState

// HandleMessages listens for messages on the broadcast channel and distributes them to connections.
func HandleMessages() {
	for {
		msg := <-broadcast // Read incoming message from the broadcast channel

		var msgMap map[string]interface{}
		var meetFilter string

		// attempt to parse the message as JSON
		if err := json.Unmarshal(msg, &msgMap); err == nil {
			if m, ok := msgMap["meetName"].(string); ok {
				meetFilter = m
			}
		}

		// iterate over all active WebSocket connections
		for c := range connections {
			// if a meet filter is set, only send to matching connections
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

// BroadcastMessage sends a message to all WebSocket clients associated with the given meet.
func BroadcastMessage(meetName string, message map[string]interface{}) {
	logger.Debug.Printf("Broadcasting next attempt timers for meet: %s", meetName)

	// convert message to JSON
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error.Printf("Error marshalling message: %v", err)
		return
	}

	// send the marshalled message to the broadcast channel
	broadcast <- msg
}

// broadcastFinalResults sends the final decisions to all connections in a meet.
// It then starts the next attempt timer and, after a timeout, broadcasts a "clearResults" message.
func broadcastFinalResults(meetName string) {
	meetState := getMeetStateFunc(meetName) // fetch the current meet state

	// prepare the decision submission message
	submission := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}

	// convert submission to JSON
	resultMsg, err := json.Marshal(submission)
	if err != nil {
		logger.Error.Printf("Error marshalling final results message: %v", err)
		return
	}
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' is being sent with Left=%s, center=%s, Right=%s", meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])

	// broadcast the results to all clients
	broadcast <- resultMsg

	// start the next attempt timer
	StartNextAttemptTimer(meetState)

	// after a timeout, send a message to clear results
	go func() {
		sleepFunc(time.Duration(resultsDisplayDuration) * time.Second)

		// prepare a clear message
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			logger.Error.Printf("Error marshalling clearResults: %v", err)
			return
		}

		// send the clear message to the broadcast channel
		broadcast <- clearJSON
	}()

	// reset judge decisions for the next round
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a time update message with an index to all clients in the meet.
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) {
	// prepare the time update message
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

	// send the time update message to the broadcast channel
	broadcast <- msg
}

// SendBroadcastMessage allows raw byte data to be sent over the broadcast channel
func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

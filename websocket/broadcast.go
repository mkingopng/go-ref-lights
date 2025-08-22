// Package websocket handles real-time WebSocket communication between referees and the meet system.
// file: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"time"

	"go-ref-lights/logger"
)

const (
	// ErrEmptyMeetName is the error message for empty meetName validation
	ErrEmptyMeetName = "meetName is empty - message will not be properly filtered"
)

// validateMeetName checks if meetName is valid and logs an error if not
func validateMeetName(meetName, functionName string) bool {
	if meetName == "" {
		logger.Error.Printf("[%s] %s", functionName, ErrEmptyMeetName)
		return false
	}
	if len(meetName) > 100 { // reasonable limit to prevent potential issues
		logger.Error.Printf("[%s] meetName too long (%d chars) - potential security issue", functionName, len(meetName))
		return false
	}
	return true
}

// allow tests to override the sleep behaviour.
var sleepFunc = time.Sleep

// StartNextAttemptTimer is an exported wrapper that triggers the next attempt timer for the given meet.
func StartNextAttemptTimer(meetState *MeetState) {
	// check if the meetState is nil
	if defaultTimerManager == nil {
		logger.Error.Println("[StartNextAttemptTimer] defaultTimerManager is nil!")
		return
	}
	// check if the meetState is nil
	defaultTimerManager.startNextAttemptTimer(meetState)
}

// HandleMessages listens for messages on the broadcast channel and distributes them to connections.
func HandleMessages() {

	for {
		// read incoming message from the broadcast channel
		msg := <-broadcast

		// parse and annotate
		var msgMap map[string]interface{}
		var meetFilter string

		if err := json.Unmarshal(msg, &msgMap); err == nil {
			if m, ok := msgMap["meetName"].(string); ok {
				meetFilter = m
			}
		} else {
			logger.Debug.Printf("[HandleMessages] JSON unmarshal error: %v", err)
		}

		// acquire lock, broadcast to each connection
		connectionsMu.RLock()
		for c := range connections {
			if meetFilter != "" && c.meetName != meetFilter {
				continue
			}
			select {
			case c.send <- msg:
				// message queued
			default:
				logger.Warn.Printf("[HandleMessages] Dropping broadcast msg for %v", c.conn.RemoteAddr())
			}
		}
		connectionsMu.RUnlock()
	}
}

// BroadcastMessage sends a message to all WebSocket clients associated with the given meet.
func BroadcastMessage(meetName string, message map[string]interface{}) {
	if !validateMeetName(meetName, "BroadcastMessage") {
		return
	}

	logger.Debug.Printf("[BroadcastMessage] Broadcasting message for meet=%s", meetName)

	// add meetName to the message to ensure proper filtering
	message["meetName"] = meetName

	// convert message to JSON
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error.Printf("[BroadcastMessage] Error marshalling message: %v", err)
		return
	}

	// send the marshalled message to the broadcast channel
	broadcast <- msg
}

// broadcastFinalResults sends the final decisions to all connections in a meet
func broadcastFinalResults(meetName string) {
	if !validateMeetName(meetName, "broadcastFinalResults") {
		return
	}

	meetState := DefaultStateProvider.GetMeetState(meetName) // fetch the current meet state

	// prepare the decision submission message
	submission := map[string]string{
		"action":         "displayResults",
		"meetName":       meetName,
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}

	// convert submission to JSON
	resultMsg, err := json.Marshal(submission)
	if err != nil {
		logger.Error.Printf("[broadcastFinalResults] Error marshalling final results message: %v", err)
		return
	}
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' with Left=%s, center=%s, Right=%s",
		meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])

	// broadcast the results to all clients
	broadcast <- resultMsg

	// start the next attempt timer
	StartNextAttemptTimer(meetState)

	// after a timeout, send a message to clear results
	go func() {
		sleepFunc(time.Duration(resultsDisplayDuration) * time.Second)
		// prepare a clear message
		clearMsg := map[string]string{
			"action":   "clearResults",
			"meetName": meetName,
		}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			logger.Error.Printf("[broadcastFinalResults] Error marshalling clearResults: %v", err)
			return
		}
		// send the clear message to the broadcast channel
		broadcast <- clearJSON
	}()
	// reset judge decisions for the next round
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a time update message with an index to all clients in the meet.
//
//nolint:unused
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) { //nolint:unused
	if !validateMeetName(meetName, "broadcastTimeUpdateWithIndex") {
		return
	}

	msg, err := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
		"meetName": meetName,
	})
	if err != nil {
		logger.Error.Printf("[broadcastTimeUpdateWithIndex] Error marshalling time update: %v", err)
		return
	}

	// send the time update message to the broadcast channel
	broadcast <- msg
}

// SendBroadcastMessage allows raw byte data to be sent over the broadcast channel.
// Note: This function does not validate meetName as it accepts pre-marshalled data.
// Callers are responsible for ensuring the data contains proper meetName for filtering.
func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

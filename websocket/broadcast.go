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
		// Keep ERROR level for empty meet name validation
		logContext := logger.NewWebSocketContext("validation_error", "", "", "")
		logContext["function"] = functionName
		logContext["error"] = string(ErrEmptyMeetName)
		logger.LogErrorWithContext(logContext, "Meet name validation failed")
		return false
	}
	if len(meetName) > 100 { // reasonable limit to prevent potential issues
		// Keep ERROR level for security-related validation
		context := logger.NewWebSocketContext("validation_error", meetName, "", "")
		context["function"] = functionName
		context["meetNameLength"] = len(meetName)
		logger.LogErrorWithContext(context, "Meet name too long - potential security issue")
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
		// Keep ERROR level for nil timer manager
		context := logger.NewSystemContext("timer_manager_nil", "broadcast")
		logger.LogErrorWithContext(context, "defaultTimerManager is nil")
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
			// Keep as DEBUG level for JSON unmarshal errors
			context := logger.NewWebSocketContext("json_unmarshal_error", "", "", "")
			context = logger.AddError(context, err)
			logger.LogDebugWithContext(context, "JSON unmarshal error in message handling")
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
				// Keep WARN level for dropped messages
				context := logger.NewWebSocketContext("broadcast_message_dropped", "", "", c.conn.RemoteAddr().String())
				logger.LogWarnWithContext(context, "Dropping broadcast message due to full send channel")
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

	// Keep as DEBUG level for routine broadcast operations
	context := logger.NewWebSocketContext("message_broadcasting", meetName, "", "")
	logger.LogDebugWithContext(context, "Broadcasting message to meet")

	// add meetName to the message to ensure proper filtering
	message["meetName"] = meetName

	// convert message to JSON
	msg, err := json.Marshal(message)
	if err != nil {
		// Keep ERROR level for marshaling failures
		context := logger.NewWebSocketContext("broadcast_marshal_error", meetName, "", "")
		context = logger.AddError(context, err)
		logger.LogErrorWithContext(context, "Failed to marshal broadcast message")
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
		// Keep ERROR level for marshaling failures
		context := logger.NewWebSocketContext("final_results_marshal_error", meetName, "", "")
		context = logger.AddError(context, err)
		logger.LogErrorWithContext(context, "Failed to marshal final results message")
		return
	}
	// Convert routine final results broadcast to DEBUG level
	context := logger.NewWebSocketContext("final_results_broadcast", meetName, "", "")
	context["leftDecision"] = meetState.JudgeDecisions["left"]
	context["centerDecision"] = meetState.JudgeDecisions["center"]
	context["rightDecision"] = meetState.JudgeDecisions["right"]
	logger.LogDebugWithContext(context, "Broadcasting final results to meet")

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
			// Keep ERROR level for marshaling failures
			context := logger.NewWebSocketContext("clear_results_marshal_error", meetName, "", "")
			context = logger.AddError(context, err)
			logger.LogErrorWithContext(context, "Failed to marshal clearResults message")
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
		// Keep ERROR level for marshaling failures
		context := logger.NewTimerContext("time_update_marshal_error", "", "time_update", index)
		context = logger.AddError(context, err)
		logger.LogErrorWithContext(context, "Failed to marshal time update message")
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

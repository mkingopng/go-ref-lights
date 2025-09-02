// Package websocket Description: This file contains the implementation of the
// realMessenger struct, which is used to send messages to all connected clients.
// file: websocket/messenger.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
)

var defaultMessenger Messenger = &realMessenger{}

// Messenger is an interface for broadcasting messages.
type Messenger interface {
	BroadcastMessage(meetName string, msg map[string]interface{})
	BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string)
	BroadcastRaw(msg []byte)
}

// realMessenger is a concrete Messenger that writes messages to the global 'broadcast' channel.
type realMessenger struct{}

// BroadcastMessage marshals the message and sends it to all connections in the given meet.
func (r *realMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	if !validateMeetName(meetName, "realMessenger.BroadcastMessage") {
		return
	}

	// Add meetName to the message to ensure proper filtering
	msg["meetName"] = meetName

	m, err := json.Marshal(msg)
	if err != nil {
		// Log marshaling failure with comprehensive error context
		errorCtx := logger.NewErrorContext(logger.MarshalingError, logger.SeverityMedium, "Failed to marshal broadcast message").
			WithCode("MSG_001").
			WithMeet(meetName, "").
			WithError(err).
			WithDetail("messageType", "broadcast").
			WithDetail("messageKeys", getMapKeys(msg))

		errorCtx.LogError()
		return
	}
	broadcast <- m
	// Convert routine broadcast success to DEBUG level
	context := logger.NewWebSocketContext("message_broadcast", meetName, "", "")
	logger.LogDebugWithContext(context, "Message broadcast to meet")
}

// BroadcastTimeUpdate sends a time update message (with index) to all connections.
func (r *realMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	if !validateMeetName(meetName, "realMessenger.BroadcastTimeUpdate") {
		return
	}

	msg := map[string]interface{}{
		"action":   action,
		"index":    index,
		"timeLeft": timeLeft,
		"meetName": meetName,
	}
	m, err := json.Marshal(msg)
	if err != nil {
		// Log marshaling failure with comprehensive error context
		errorCtx := logger.NewTimerErrorContext(
			"Failed to marshal time update message",
			meetName,
			action,
			index,
		).WithCode("MSG_002").
			WithError(err).
			WithDetail("timeLeft", timeLeft).
			WithDetail("messageType", "time_update")

		errorCtx.LogError()
		return
	}
	broadcast <- m
	// Convert routine time updates to DEBUG level to reduce noise
	context := logger.NewTimerContext("time_update_broadcast", meetName, action, index)
	context["timeLeft"] = timeLeft
	logger.LogDebugWithContext(context, "Time update broadcast to meet")
}

// BroadcastRaw sends a raw JSON message.
func (r *realMessenger) BroadcastRaw(msg []byte) {
	broadcast <- msg
	// Convert routine raw broadcasts to DEBUG level
	context := logger.NewWebSocketContext("raw_broadcast", "", "", "")
	context["messageSize"] = len(msg)
	logger.LogDebugWithContext(context, "Raw message broadcast sent")
}

// getMapKeys returns the keys of a map[string]interface{} for logging purposes
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

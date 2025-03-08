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

type realMessenger struct{}

// --------------- Methods on realMessenger -----------------

// BroadcastMessage marshals the message and sends it to all connections.
func (r *realMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("realMessenger: Error marshalling message: %v", err)
		return
	}
	broadcast <- m
	logger.Info.Printf("realMessenger: BroadcastMessage sent to meet %s", meetName)
}

// BroadcastTimeUpdate sends a time update message.
func (r *realMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	msg := map[string]interface{}{
		"action":   action,
		"index":    index,
		"timeLeft": timeLeft,
		"meetName": meetName,
	}
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("realMessenger: Error marshalling time update: %v", err)
		return
	}
	broadcast <- m
	logger.Info.Printf("realMessenger: BroadcastTimeUpdate for meet %s: action=%s, timeLeft=%d", meetName, action, timeLeft)
}

// BroadcastRaw sends a raw JSON message.
func (r *realMessenger) BroadcastRaw(msg []byte) {
	broadcast <- msg
	logger.Info.Printf("realMessenger: BroadcastRaw sent: %s", string(msg))
}

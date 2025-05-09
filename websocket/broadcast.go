// Package websocket handles real-time WebSocket communication between referees and the meet system.
// file: websocket/broadcast.go
package websocket

import (
	"encoding/json"
	"fmt"
	"time"

	"go-ref-lights/logger"
)

// marshallWithRetry attempts to marshal an object to JSON with retries
// Returns marshalled bytes or nil if all attempts fail
func marshallWithRetry(v interface{}, maxRetries int) ([]byte, error) {
	var lastError error
	for i := 0; i < maxRetries; i++ {
		bytes, err := json.Marshal(v)
		if err == nil {
			return bytes, nil
		}
		lastError = err
		logger.Warn.Printf("[marshallWithRetry] Attempt %d/%d failed: %v", i+1, maxRetries, err)
		time.Sleep(10 * time.Millisecond) // Small delay between retries
	}
	logger.Error.Printf("[marshallWithRetry] All %d attempts failed", maxRetries)
	return nil, lastError
}

// safeSend queues data or logs & drops if the buffer is full (prevents deadlock)
func safeSend(data []byte) {
	select {
	case broadcast <- data:
		// Message successfully queued
	default:
		logger.Warn.Println("[safeSend] broadcast channel FULL – dropping msg")
	}
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

		startTime := time.Now()

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

		// Count how many connections we'll broadcast to
		connectionCount := 0
		droppedCount := 0

		// acquire lock, broadcast to each connection
		connectionsMu.RLock()
		for c := range connections {
			if meetFilter != "" && c.meetName != meetFilter {
				continue
			}
			connectionCount++
			select {
			case c.send <- msg:
				// message queued
			default:
				droppedCount++
				logger.Warn.Printf("[HandleMessages] Dropping broadcast msg for %v", c.conn.RemoteAddr())
			}
		}
		connectionsMu.RUnlock()

		duration := time.Since(startTime)

		// Only log performance metrics if this broadcast was non-trivial
		if connectionCount > 0 {
			logger.LogPerformanceMetric(fmt.Sprintf("Broadcast to %d connections (%d dropped)",
				connectionCount, droppedCount), duration)

			// Log exceptional events when we drop a large percentage of messages
			if droppedCount > 0 && float64(droppedCount)/float64(connectionCount) > 0.1 {
				logger.LogExceptional("Dropped %d of %d messages (%.1f%%) during broadcast",
					droppedCount, connectionCount, 100*float64(droppedCount)/float64(connectionCount))
			}
		}
	}
}

// BroadcastMessage sends a message to all WebSocket clients associated with the given meet.
func BroadcastMessage(meetName string, message map[string]interface{}) {
	logger.Debug.Printf("[BroadcastMessage] Broadcasting next attempt timers for meet=%s", meetName)

	// convert message to JSON with retry
	msg, err := marshallWithRetry(message, 3)
	if err != nil {
		logger.Error.Printf("[BroadcastMessage] Error marshalling message after retries: %v", err)
		return
	}

	// send the marshalled message to the broadcast channel
	safeSend(msg)
}

// broadcastFinalResults sends the final decisions to all connections in a meet
func broadcastFinalResults(meetName string) {
	meetState := DefaultStateProvider.GetMeetState(meetName) // fetch the current meet state

	// prepare the decision submission message
	submission := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}

	// convert submission to JSON with retry
	resultMsg, err := marshallWithRetry(submission, 3)
	if err != nil {
		logger.Error.Printf("[broadcastFinalResults] Error marshalling final results message after retries: %v", err)
		return
	}
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' with Left=%s, center=%s, Right=%s",
		meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])

	// broadcast the results to all clients
	safeSend(resultMsg)

	// start the next attempt timer
	StartNextAttemptTimer(meetState)

	// after a timeout, send a message to clear results
	go func() {
		sleepFunc(time.Duration(resultsDisplayDuration) * time.Second)
		// prepare a clear message
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := marshallWithRetry(clearMsg, 3)
		if err != nil {
			logger.Error.Printf("[broadcastFinalResults] Error marshalling clearResults after retries: %v", err)
			return
		}
		// send the clear message to the broadcast channel
		safeSend(clearJSON)
	}()
	// reset judge decisions for the next round
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a time update message with an index to all clients in the meet.
//
//nolint:unused
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) { //nolint:unused
	msg, err := marshallWithRetry(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
		"meetName": meetName,
	}, 3)
	if err != nil {
		logger.Error.Printf("[broadcastTimeUpdateWithIndex] Error marshalling time update after retries: %v", err)
		return
	}

	// send the time update message to the broadcast channel
	safeSend(msg)
}

// SendBroadcastMessage allows raw byte data to be sent over the broadcast channel
func SendBroadcastMessage(data []byte) {
	safeSend(data)
}

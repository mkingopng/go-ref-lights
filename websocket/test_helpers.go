// Package websocket test_helpers.go
// File: websocket/test_helpers.go
package websocket

import "time"

// InitTest sets up the test environment for WebSocket-based meet state handling.
func InitTest() {
	// flush the broadcast channel if necessary.
	for len(broadcast) > 0 {
		<-broadcast
	}
	resultsDisplayDuration = 15 // reset the results display duration if needed.
	sleepFunc = time.Sleep      // reset the sleep function to the standard one.

	if defaultTimerManager != nil {
		defaultTimerManager.nextAttemptIDCounter = 0
	}
}

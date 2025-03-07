// Package websocket contains the WebSocket server and client code for the meet.
// file: websocket/test_helpers.go
package websocket

import "time"

// InitTest sets up the test environment for WebSocket-based meet state handling.
func InitTest() {
	// Flush the broadcast channel if necessary.
	for len(broadcast) > 0 {
		<-broadcast
	}
	resultsDisplayDuration = 15     // Reset the results display duration if needed.
	sleepFunc = time.Sleep          // Reset the sleep function to the standard one.
	getMeetStateFunc = getMeetState // Reset the getMeetStateFunc if you’ve overridden it.
	nextAttemptIDCounter = 0        // Reset the nextAttemptIDCounter.
}

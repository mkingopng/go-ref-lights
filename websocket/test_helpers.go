// Package websocket test_helpers.go
package websocket

import "time"

// InitTest sets up the test environment for WebSocket-based meet state handling.
func InitTest() {
	// Flush the broadcast channel if necessary.
	for len(broadcast) > 0 {
		<-broadcast
	}
	resultsDisplayDuration = 15 // Reset the results display duration if needed.
	sleepFunc = time.Sleep      // Reset the sleep function to the standard one.
	// No need to reset getMeetStateFunc since we now use DefaultStateProvider.GetMeetState.

	// Reset the next attempt timer counter if the default timer manager is initialized.
	if defaultTimerManager != nil {
		defaultTimerManager.nextAttemptIDCounter = 0
	}
}

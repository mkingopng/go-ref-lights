package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
)

// --------------- utility functions -------------------------------------

// findTimerIndex returns the index of the timer with the given ID.
func findTimerIndex(timers []NextAttemptTimer, id int) int {
	for i, t := range timers {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// broadcastAllNextAttemptTimers sends a message with the current next-attempt timers.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	msg := map[string]interface{}{
		"action":   "updateNextAttemptTime", // CHANGED: Use this action name for next-attempt updates.
		"timers":   timers,
		"meetName": meetName,
	}
	out, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("[broadcastAllNextAttemptTimers] Error marshalling next attempt timers: %v", err)
		return
	}
	broadcastToMeet(meetName, out)
}

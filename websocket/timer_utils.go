package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// --------------- utility functions -------------------------------------

// findTimerIndex returns the index of the timer with the given ID
func findTimerIndex(timers []NextAttemptTimer, id int) int {
	for i, t := range timers {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// broadcastAllNextAttemptTimers sends a message with the current next-attempt timers
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	var typedTimers []map[string]interface{}

	for _, t := range timers {
		typedTimers = append(typedTimers, map[string]interface{}{
			"ID":       t.ID,
			"TimeLeft": t.TimeLeft,
			"Active":   t.Active,
			"EndTime":  t.EndTime.Format(time.RFC3339),
			"type":     "nextAttempt",
		})
	}

	msg := map[string]interface{}{
		"action":   "updateNextAttemptTime",
		"timers":   typedTimers,
		"meetName": meetName,
	}

	out, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("[broadcastAllNextAttemptTimers] Error marshalling next attempt timers: %v", err)
		return
	}

	broadcastToMeet(meetName, out)
}

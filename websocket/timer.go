// Package websocket - websocket/timer.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	logger.Info.Printf(
		"[handleTimerAction] Received '%s' for meet '%s'",
		action,
		meetName,
	)

	meetState := getMeetState(meetName)
	logger.Info.Printf(
		"[handleTimerAction] got MeetState pointer %p for meet '%s'",
		meetState,
		meetName,
	)

	switch action {
	case "startTimer":
		logger.Info.Printf("[handleTimerAction] Clearing old decisions, sending 'clearResults' broadcast")

		// 1) clear old decision
		meetState.JudgeDecisions = make(map[string]string)

		// 2) broadcast a "clearResults" so the Lights page resets its UI
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

		// 3) start the Platform Ready timer
		logger.Info.Printf(
			"[handleTimerAction] Now calling startNextAttemptTimer(...) for meet '%s'",
			meetName,
		)
		startPlatformReadyTimer(meetState)

	case "resetTimer":
		logger.Info.Printf("🔄 Processing resetTimer action for meet: %s", meetName)
		resetPlatformReadyTimer(meetState)
		// clear judge decisions on reset if you want
		meetState.JudgeDecisions = make(map[string]string)
		// broadcast 'clearResults' to reset visuals
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

	case "startNextAttemptTimer":
		logger.Info.Printf("[handleTimerAction] Now calling startNextAttemptTimer(...) for meet '%s'", meetName)
		startNextAttemptTimer(meetState)

	case "updatePlatformReadyTime":
		// Do nothing, or log and ignore, since these updates are meant for clients
		logger.Debug.Printf("Ignoring timer update echo from client for meet: %s", meetName)
		return
	default:
		logger.Debug.Printf("[handleTimerAction] action '%s' not recognized in switch", action)
	}

	logger.Info.Printf("[handleTimerAction] Finished processing action '%s' for meet '%s'", action, meetName)
}

// startPlatformReadyTimer start/Stop/Reset the Platform Ready Timer
func startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] called for meet: %s", meetState.MeetName)
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	logger.Info.Println("🚦 Attempting to start Platform Ready Timer for meet: %s", meetState.MeetName)

	// log whether the timer is already active
	if meetState.PlatformReadyActive {
		logger.Warn.Printf("[startPlatformReadyTimer] Timer already active for meet: %s", meetState.MeetName)
		return
	}

	meetState.PlatformReadyActive = true
	meetState.PlatformReadyTimeLeft = 60
	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s", meetState.MeetName)

	// debugging check - ensuring no duplicate goroutines
	logger.Debug.Printf("🛠️ Starting Platform Ready Timer loop for meet: %s", meetState.MeetName)

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			platformReadyMutex.Lock()
			if !meetState.PlatformReadyActive {
				logger.Info.Printf(
					"[startPlatformReadyTimer] Timer was stopped early for meet: %s",
					meetState.MeetName,
				)
				platformReadyMutex.Unlock()
				return
			}

			meetState.PlatformReadyTimeLeft--
			logger.Debug.Printf("⏰ Platform Ready Time Left: %d seconds left in meet %s",
				meetState.PlatformReadyTimeLeft,
				meetState.MeetName,
			)

			broadcastTimeUpdateWithIndex("updatePlatformReadyTime", meetState.PlatformReadyTimeLeft, 0, meetState.MeetName)

			if meetState.PlatformReadyTimeLeft <= 0 {
				logger.Info.Printf("[startPlatformReadyTimer] Timer reached 0; marking expired for meet: %s", meetState.MeetName)
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				meetState.PlatformReadyActive = false
				meetState.PlatformReadyTimeLeft = 60
				platformReadyMutex.Unlock()
				return
			}
			platformReadyMutex.Unlock()
		}
	}()
}

// resetPlatformReadyTimer resets the Platform Ready Timer
func resetPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		logger.Warn.Println("⚠️ No active timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// todo: clear the timer after it hits 0
// startNextAttemptTimer is a struct for tracking the next attempt timer
func startNextAttemptTimer(meetState *MeetState) {
	nextAttemptMutex.Lock()
	nextAttemptIDCounter++
	timerID := nextAttemptIDCounter

	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: 60,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	nextAttemptMutex.Unlock()

	ticker := time.NewTicker(1 * time.Second)
	go func(id int) {
		defer ticker.Stop()
		for range ticker.C {
			nextAttemptMutex.Lock()

			// 1) locate the timer by ID
			idx := findTimerIndex(meetState.NextAttemptTimers, id)
			if idx == -1 {
				// timer was removed or doesn't exist
				nextAttemptMutex.Unlock()
				return
			}

			// 2) if it's inactive, just exit
			if !meetState.NextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}

			// 3) decrement time
			meetState.NextAttemptTimers[idx].TimeLeft--

			// 4) re-broadcast indexes. We compute a fresh "display index" for each timer:
			//    e.g., if we have 3 timers left, they become #1, #2, #3 in the order they appear.
			//    So we do a separate function to broadcast them all after each second.
			broadcastAllNextAttemptTimers(meetState.NextAttemptTimers, meetState.MeetName)

			// 5) check if it reached 0
			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
				// mark it inactive (or remove it completely)
				meetState.NextAttemptTimers[idx].Active = false

				// Calculate display index for the expired timer (array index + 1)
				expiredDisplayIndex := idx + 1
				// Broadcast an expiration message with the correct display index
				broadcastTimeUpdateWithIndex("nextAttemptExpired", 0, expiredDisplayIndex, meetState.MeetName)

				// remove this timer from the slice
				meetState.NextAttemptTimers = removeTimerByIndex(meetState.NextAttemptTimers, idx)

				// re-broadcast again so the display indexes reset now that this timer is gone
				broadcastAllNextAttemptTimers(meetState.NextAttemptTimers, meetState.MeetName)

				nextAttemptMutex.Unlock()
				return
			}
			nextAttemptMutex.Unlock()
		}
	}(timerID)
}

// findTimerIndex returns the index of the timer with the given ID, or -1 if not found.
func findTimerIndex(timers []NextAttemptTimer, id int) int {
	for i, t := range timers {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// removeTimerByIndex removes the timer at [idx] from the slice
func removeTimerByIndex(timers []NextAttemptTimer, idx int) []NextAttemptTimer {
	return append(timers[:idx], timers[idx+1:]...)
}

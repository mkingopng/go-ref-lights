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
		meetState.JudgeDecisions = make(map[string]string)

		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

		BroadcastMessage(meetName, map[string]interface{}{"action": "startTimer"})
		logger.Info.Printf("[handleTimerAction] Now calling startNextAttemptTimer(...) for meet '%s'", meetName)
		startPlatformReadyTimer(meetState)

	case "resetTimer":
		logger.Info.Printf("🔄 Processing resetTimer action for meet: %s", meetName)
		resetPlatformReadyTimer(meetState)
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

	case "startNextAttemptTimer":
		logger.Info.Printf("[handleTimerAction] Now calling startNextAttemptTimer(...) for meet '%s'", meetName)
		startNextAttemptTimer(meetState)

	case "updatePlatformReadyTime":
		logger.Debug.Printf("Ignoring timer update echo from client for meet: %s", meetName)
		return
	default:
		logger.Debug.Printf("[handleTimerAction] action '%s' not recognized in switch", action)
	}
	logger.Info.Printf("[handleTimerAction] Finished processing action '%s' for meet '%s'", action, meetName)
}

// startPlatformReadyTimer uses a time-based approach to avoid ticker drift
func startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] called for meet: %s", meetState.MeetName)
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if meetState.PlatformReadyActive {
		logger.Warn.Printf("[startPlatformReadyTimer] Timer already active for meet: %s", meetState.MeetName)
		return
	}

	meetState.PlatformReadyActive = true

	// MAJOR CHANGE: Instead of storing an integer countdown, store the end time
	// e.g., 60 seconds from now
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)

	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s, endTime=%v",
		meetState.MeetName,
		meetState.PlatformReadyEnd,
	)

	ticker := time.NewTicker(1 * time.Second)
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

			// Compute how many seconds remain
			timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())

			if timeLeft <= 0 {
				// Timer expired
				logger.Info.Printf("[startPlatformReadyTimer] Timer reached 0; marking expired for meet: %s", meetState.MeetName)
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				meetState.PlatformReadyActive = false

				// Reset or just keep the old end time if you like
				meetState.PlatformReadyEnd = time.Time{}

				platformReadyMutex.Unlock()
				return
			}

			// Broadcast how many seconds remain
			broadcastTimeUpdateWithIndex("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)
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

			idx := findTimerIndex(meetState.NextAttemptTimers, id)
			if idx == -1 {
				nextAttemptMutex.Unlock()
				return
			}

			if !meetState.NextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}

			meetState.NextAttemptTimers[idx].TimeLeft--
			broadcastAllNextAttemptTimers(meetState.NextAttemptTimers, meetState.MeetName)
			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
				meetState.NextAttemptTimers[idx].Active = false
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

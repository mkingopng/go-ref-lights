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

// startPlatformReadyTimer the Platform Ready Timer calls the lifter to the platform
func startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] called for meet: %s", meetState.MeetName)
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	logger.Info.Println("🚦 Attempting to start Platform Ready Timer for meet: %s", meetState.MeetName)

	if meetState.PlatformReadyActive {
		logger.Warn.Printf("[startPlatformReadyTimer] Timer already active for meet: %s", meetState.MeetName)
		return
	}

	meetState.PlatformReadyActive = true
	meetState.PlatformReadyTimeLeft = 60

	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s", meetState.MeetName)
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

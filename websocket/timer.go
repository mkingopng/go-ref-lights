package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"time"
)

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	meetState := getMeetState(meetName)
	switch action {
	case "startTimer":
		// only allow "Platform Ready" if all refs are connected // fix_me
		//if !isAllRefsConnected(meetState) {
		//	errMsg := map[string]string{
		//		"action":  "healthError",
		//		"message": "Cannot start timer: Not all referees are connected!",
		//	}
		//	msg, _ := json.Marshal(errMsg)
		//	broadcast <- msg
		//	logger.Error.Println("❌ Timer not started: All referees not connected.")
		//	return
		//}

		// 1) clear old decision
		meetState.JudgeDecisions = make(map[string]string)

		// 2) broadcast a "clearResults" so the Lights page resets its UI
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

		// 3) start the Platform Ready timer
		startPlatformReadyTimer(meetState)

	// not required per Daniel
	//case "stopTimer":
	//	stopPlatformReadyTimer(meetState)

	case "resetTimer":
		resetPlatformReadyTimer(meetState)
		// clear judge decisions on reset if you want
		meetState.JudgeDecisions = make(map[string]string)
		// broadcast 'clearResults' to reset visuals
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

	case "startNextAttemptTimer":
		startNextAttemptTimer(meetState)
	}

	logger.Info.Printf("✅ Timer action processed: %s (meet: %s)", action, meetName)
}

// startPlatformReadyTimer start/Stop/Reset the Platform Ready Timer
func startPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if meetState.PlatformReadyActive {
		logger.Warn.Println("⚠️ Platform Ready Timer already running.")
		return
	}
	meetState.PlatformReadyActive = true
	meetState.PlatformReadyTimeLeft = 60

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			platformReadyMutex.Lock()
			if !meetState.PlatformReadyActive {
				platformReadyMutex.Unlock()
				return
			}
			meetState.PlatformReadyTimeLeft--
			broadcastTimeUpdateWithIndex("updatePlatformReadyTime", meetState.PlatformReadyTimeLeft, 0, meetState.MeetName)
			if meetState.PlatformReadyTimeLeft <= 0 {
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

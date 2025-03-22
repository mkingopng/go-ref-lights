// Package websocket manages timers for platform readiness and next attempts.
// File: websocket/timer_manager.go

package websocket

import (
	"context"
	"encoding/json"
	"go-ref-lights/logger"
	"sync"
	"time"
)

// platformReadyTimer is declared here but not used directly. (We rely on context/cancel.)
var platformReadyTimer *time.Timer

// Default instance of TimerManager.
var defaultTimerManager *TimerManager

// Overridable function for broadcasting next attempt timers (used in tests).
var broadcastAllNextAttemptTimersFunc = broadcastAllNextAttemptTimers

// TimerManager manages platform readiness and next attempt timers.
type TimerManager struct {
	Provider              StateProvider // Provides access to meet state
	Messenger             Messenger     // Handles message broadcasting
	TickerInterval        time.Duration // Interval between timer updates
	NextAttemptStartValue int           // Default start value for next attempt timers
	nextAttemptMutex      sync.Mutex    // Mutex for next attempt timers
	platformReadyMutex    sync.Mutex    // Mutex for platform readiness timer
	nextAttemptIDCounter  int           // Counter for next attempt timers
}

// init sets up the default timer manager.
func init() {
	defaultTimerManager = &TimerManager{
		Provider:              DefaultStateProvider,
		Messenger:             defaultMessenger,
		TickerInterval:        1 * time.Second, // default 1s interval
		NextAttemptStartValue: 60,              // default 60s for next attempt
	}
}

// --------------------- timer action handler ---------------------

// HandleTimerAction processes different timer actions like "startTimer", "resetTimer", etc.
func (tm *TimerManager) HandleTimerAction(action, meetName string) {
	logger.Info.Printf("[HandleTimerAction] Received '%s' for meet='%s'", action, meetName)

	meetState := tm.Provider.GetMeetState(meetName)
	logger.Info.Printf("[HandleTimerAction] Using MeetState pointer %p for meet='%s'", meetState, meetName)

	switch action {
	case "startTimer":
		// Clear previous decisions and notify clients to clear results
		logger.Info.Printf("[HandleTimerAction] Clearing old decisions, sending 'clearResults'")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

		// Explicitly cancel any active platform ready timer
		CancelPlatformReadyTimer(meetName)

		// Start the platform ready timer
		tm.Messenger.BroadcastMessage(meetName, map[string]interface{}{"action": "startTimer"})
		logger.Info.Printf("[HandleTimerAction] Now calling startPlatformReadyTimer for meet='%s'", meetName)
		tm.startPlatformReadyTimer(meetState)

	case "resetTimer":
		logger.Info.Printf("[HandleTimerAction] 🔄 Processing resetTimer action for meet='%s'", meetName)
		tm.resetPlatformReadyTimer(meetState)
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

	case "startNextAttemptTimer":
		logger.Info.Printf("[HandleTimerAction] Now calling startNextAttemptTimer for meet='%s'", meetName)
		tm.startNextAttemptTimer(meetState)

	case "updatePlatformReadyTime":
		// The UI might be echoing updates; we typically ignore or no-op here
		logger.Debug.Printf("[HandleTimerAction] Ignoring timer update echo from client for meet='%s'", meetName)
		return

	default:
		logger.Debug.Printf("[HandleTimerAction] Action='%s' not recognized", action)
	}

	logger.Info.Printf("[HandleTimerAction] Finished processing action='%s' for meet='%s'", action, meetName)
}

// -------------------- platform ready timer management --------------------

// startPlatformReadyTimer starts a 60-second platform readiness timer
func (tm *TimerManager) startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] Called for meet='%s'", meetState.MeetName)

	tm.platformReadyMutex.Lock()
	// Cancel existing timer if running
	if meetState.PlatformReadyCancel != nil {
		meetState.PlatformReadyCancel()
	}

	// Create a new cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	meetState.PlatformReadyCtx = ctx
	meetState.PlatformReadyCancel = cancel

	// Increment the timer ID for tracking
	meetState.PlatformReadyTimerID++
	localTimerID := meetState.PlatformReadyTimerID

	// Set the single timer to active and store its end time
	meetState.PlatformReadyActive = true
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)
	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet='%s', endTime=%v",
		meetState.MeetName, meetState.PlatformReadyEnd)
	tm.platformReadyMutex.Unlock()

	// Clear lights and broadcast initial time left
	clearMsg := map[string]string{"action": "clearResults"}
	clearJSON, _ := json.Marshal(clearMsg)
	tm.Messenger.BroadcastRaw(clearJSON)

	timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
	tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)

	// Timer countdown using a ticker
	ticker := time.NewTicker(tm.interval())

	go func(ctx context.Context, timerID int) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tm.platformReadyMutex.Lock()

				// If a new timer started, exit this one
				if meetState.PlatformReadyTimerID != timerID {
					logger.Info.Printf("[startPlatformReadyTimer] Timer ID mismatch for meet='%s'; exiting old timer",
						meetState.MeetName)
					tm.platformReadyMutex.Unlock()
					return
				}

				// If the timer is no longer active, exit
				if !meetState.PlatformReadyActive {
					logger.Info.Printf("[startPlatformReadyTimer] Timer was stopped early for meet='%s'",
						meetState.MeetName)
					tm.platformReadyMutex.Unlock()
					return
				}

				// Calculate time left
				timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
				if timeLeft < 0 {
					timeLeft = 0
				}

				// If time is up, broadcast and reset
				if timeLeft <= 0 {
					logger.Info.Printf("[startPlatformReadyTimer] Timer reached 0; marking expired for meet='%s'",
						meetState.MeetName)
					tm.Messenger.BroadcastRaw([]byte(`{"action":"platformReadyExpired"}`))
					meetState.PlatformReadyActive = false
					meetState.PlatformReadyEnd = time.Time{}
					tm.platformReadyMutex.Unlock()
					return
				}

				// Otherwise, broadcast the updated time
				tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)
				tm.platformReadyMutex.Unlock()

			case <-ctx.Done():
				logger.Info.Printf("[startPlatformReadyTimer] Context cancelled for meet='%s'", meetState.MeetName)
				return
			}
		}
	}(ctx, localTimerID)
}

// resetPlatformReadyTimer stops the platform ready timer.
func (tm *TimerManager) resetPlatformReadyTimer(meetState *MeetState) {
	tm.platformReadyMutex.Lock()
	defer tm.platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		logger.Warn.Println("[resetPlatformReadyTimer] ⚠️ No active platform ready timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	// Optionally reset the time left to 60 if you want
	meetState.PlatformReadyTimeLeft = 60
}

// -------------------- next attempt timer management --------------------

// startNextAttemptTimer starts a timer for the next attempt.
func (tm *TimerManager) startNextAttemptTimer(meetState *MeetState) {
	tm.nextAttemptMutex.Lock()
	tm.nextAttemptIDCounter++
	timerID := tm.nextAttemptIDCounter

	// Default the next attempt to 60 seconds (or whatever NextAttemptStartValue is)
	startVal := 60
	if tm.NextAttemptStartValue > 0 {
		startVal = tm.NextAttemptStartValue
	}

	// Create a new NextAttemptTimer
	deadline := time.Now().Add(time.Duration(startVal) * time.Second)
	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: startVal,
		Active:   true,
		EndTime:  deadline,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	tm.nextAttemptMutex.Unlock()

	// Broadcast the updated list of timers
	broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

	// Start the countdown in a separate goroutine
	ticker := time.NewTicker(tm.interval())
	go func(id int) {
		defer ticker.Stop()
		for range ticker.C {
			tm.nextAttemptMutex.Lock()
			idx := findTimerIndex(meetState.NextAttemptTimers, id)
			if idx == -1 {
				// Timer not found; must've been removed or ended
				tm.nextAttemptMutex.Unlock()
				return
			}
			if !meetState.NextAttemptTimers[idx].Active {
				// Already inactive
				tm.nextAttemptMutex.Unlock()
				return
			}

			// Recalc time left from EndTime
			timeLeft := int(meetState.NextAttemptTimers[idx].EndTime.Sub(time.Now()).Seconds())
			if timeLeft < 0 {
				timeLeft = 0
			}
			meetState.NextAttemptTimers[idx].TimeLeft = timeLeft

			// Broadcast updated timers
			broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

			if timeLeft <= 0 {
				// Timer is done
				meetState.NextAttemptTimers[idx].Active = false
				tm.nextAttemptMutex.Unlock()
				return
			}
			tm.nextAttemptMutex.Unlock()
		}
	}(timerID)
}

// -------------------- timer management utilities --------------------

// interval returns the ticker interval (defaults to 1 second if unset).
func (tm *TimerManager) interval() time.Duration {
	if tm.TickerInterval > 0 {
		return tm.TickerInterval
	}
	return 1 * time.Second
}

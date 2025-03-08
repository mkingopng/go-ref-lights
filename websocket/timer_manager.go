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

// -------------- timer manager setup --------------

// platformReadyTimer represents a timer for the next attempt
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
		TickerInterval:        1 * time.Second, // default to 1s interval
		NextAttemptStartValue: 60,              // default next attempt timer to 60s
	}
}

// --------------------- timer action handler ---------------------

// HandleTimerAction processes different timer actions.
func (tm *TimerManager) HandleTimerAction(action, meetName string) {
	logger.Info.Printf("[HandleTimerAction] Received '%s' for meet '%s'", action, meetName)
	// Now using the unified state provider.
	meetState := tm.Provider.GetMeetState(meetName)
	logger.Info.Printf("[HandleTimerAction] Using unified MeetState pointer %p for meet '%s'", meetState, meetName)

	switch action {
	case "startTimer":
		// Clear previous decisions and notify clients.
		logger.Info.Printf("[HandleTimerAction] Clearing old decisions, sending 'clearResults' broadcast")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

		// explicitly cancel any active platform ready timer.
		CancelPlatformReadyTimer(meetName)

		// start the Platform Ready timer.
		tm.Messenger.BroadcastMessage(meetName, map[string]interface{}{"action": "startTimer"})
		logger.Info.Printf("[HandleTimerAction] Now calling startPlatformReadyTimer for meet '%s'", meetName)
		tm.startPlatformReadyTimer(meetState)

	case "resetTimer":
		logger.Info.Printf("🔄 Processing resetTimer action for meet: %s", meetName)
		tm.resetPlatformReadyTimer(meetState)
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

	case "startNextAttemptTimer":
		logger.Info.Printf("[HandleTimerAction] Now calling startNextAttemptTimer for meet '%s'", meetName)
		tm.startNextAttemptTimer(meetState)

	case "updatePlatformReadyTime":
		logger.Debug.Printf("Ignoring timer update echo from client for meet: %s", meetName)
		return

	default:
		logger.Debug.Printf("[HandleTimerAction] Action '%s' not recognized", action)
	}
	logger.Info.Printf("[HandleTimerAction] Finished processing action '%s' for meet '%s'", action, meetName)
}

// -------------------- platform ready timer management --------------------

// startPlatformReadyTimer starts a 60-second platform readiness timer
func (tm *TimerManager) startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] Called for meet: %s", meetState.MeetName)

	tm.platformReadyMutex.Lock()
	// Cancel existing timer if running
	if meetState.PlatformReadyCancel != nil {
		meetState.PlatformReadyCancel()
	}

	// Create a new cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	meetState.PlatformReadyCtx = ctx
	meetState.PlatformReadyCancel = cancel

	// Increment timer ID for tracking
	meetState.PlatformReadyTimerID++
	localTimerID := meetState.PlatformReadyTimerID

	meetState.PlatformReadyActive = true
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)
	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s, endTime=%v", meetState.MeetName, meetState.PlatformReadyEnd)
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

				// If a new timer started, exit this one.
				if meetState.PlatformReadyTimerID != timerID {
					logger.Info.Printf("[startPlatformReadyTimer] Timer ID mismatch for meet: %s; exiting old timer", meetState.MeetName)
					tm.platformReadyMutex.Unlock()
					return
				}

				if !meetState.PlatformReadyActive {
					logger.Info.Printf("[startPlatformReadyTimer] Timer was stopped early for meet: %s", meetState.MeetName)
					tm.platformReadyMutex.Unlock()
					return
				}

				timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
				if timeLeft <= 0 {
					logger.Info.Printf("[startPlatformReadyTimer] Timer reached 0; marking expired for meet: %s", meetState.MeetName)
					tm.Messenger.BroadcastRaw([]byte(`{"action":"platformReadyExpired"}`))
					meetState.PlatformReadyActive = false
					meetState.PlatformReadyEnd = time.Time{}
					tm.platformReadyMutex.Unlock()
					return
				}

				tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)
				tm.platformReadyMutex.Unlock()

			case <-ctx.Done():
				logger.Info.Printf("[startPlatformReadyTimer] Context cancelled for meet: %s", meetState.MeetName)
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
		logger.Warn.Println("⚠️ No active timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// -------------------- next attempt timer management --------------------

// startNextAttemptTimer starts a timer for the next attempt.
func (tm *TimerManager) startNextAttemptTimer(meetState *MeetState) {
	tm.nextAttemptMutex.Lock()
	tm.nextAttemptIDCounter++
	timerID := tm.nextAttemptIDCounter

	// determine the starting value.
	startVal := 60
	if tm.NextAttemptStartValue > 0 {
		startVal = tm.NextAttemptStartValue
	}

	// create and store the timer.
	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: startVal,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	tm.nextAttemptMutex.Unlock()

	// broadcast the new timer state
	broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

	// start timer countdown
	ticker := time.NewTicker(tm.interval())
	go func(id int) {
		defer ticker.Stop()
		for range ticker.C {
			tm.nextAttemptMutex.Lock()
			idx := findTimerIndex(meetState.NextAttemptTimers, id)
			if idx == -1 {
				tm.nextAttemptMutex.Unlock()
				return
			}
			if !meetState.NextAttemptTimers[idx].Active {
				tm.nextAttemptMutex.Unlock()
				return
			}

			// Decrement the timer.
			meetState.NextAttemptTimers[idx].TimeLeft--
			broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
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

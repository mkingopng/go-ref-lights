// Package websocket Description: TimerManager manages timers for platform readiness and next attempts.
// File: websocket/timer_manager.go
package websocket

import (
	"context"
	"encoding/json"
	"go-ref-lights/logger"
	"sync"
	"time"
)

var defaultTimerManager *TimerManager

// Overridable broadcast function variable.
// In tests we can override this function (for example, to a no-op).
var broadcastAllNextAttemptTimersFunc = broadcastAllNextAttemptTimers

// TimerManager manages timers for platform readiness and next attempts.
type TimerManager struct {
	Provider              StateProvider
	Messenger             Messenger
	TickerInterval        time.Duration
	NextAttemptStartValue int

	nextAttemptMutex     sync.Mutex
	platformReadyMutex   sync.Mutex
	nextAttemptIDCounter int
}

// init sets up the default timer manager.
func init() {
	defaultTimerManager = &TimerManager{
		Provider:              defaultStateProvider,
		Messenger:             defaultMessenger,
		TickerInterval:        1 * time.Second, // production default
		NextAttemptStartValue: 60,              // production default
	}
}

// startPlatformReadyTimer starts a 60‑second timer for platform readiness.
func (tm *TimerManager) startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] Called for meet: %s", meetState.MeetName)

	tm.platformReadyMutex.Lock()
	// Cancel any existing timer if present.
	if meetState.PlatformReadyCancel != nil {
		meetState.PlatformReadyCancel()
	}
	// Create a new cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	meetState.PlatformReadyCtx = ctx
	meetState.PlatformReadyCancel = cancel

	// Increment the timer ID for tagging.
	meetState.PlatformReadyTimerID++
	localTimerID := meetState.PlatformReadyTimerID

	meetState.PlatformReadyActive = true
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)
	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s, endTime=%v",
		meetState.MeetName, meetState.PlatformReadyEnd)
	tm.platformReadyMutex.Unlock()

	// Immediately clear the lights.
	clearMsg := map[string]string{"action": "clearResults"}
	clearJSON, _ := json.Marshal(clearMsg)
	tm.Messenger.BroadcastRaw(clearJSON)

	// Immediately broadcast the time left.
	timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
	tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)

	ticker := time.NewTicker(tm.interval())
	go func(ctx context.Context, timerID int) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tm.platformReadyMutex.Lock()
				// If a new timer has been started, exit this goroutine.
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
				tLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
				if tLeft <= 0 {
					logger.Info.Printf("[startPlatformReadyTimer] Timer reached 0; marking expired for meet: %s", meetState.MeetName)
					tm.Messenger.BroadcastRaw([]byte(`{"action":"platformReadyExpired"}`))
					meetState.PlatformReadyActive = false
					meetState.PlatformReadyEnd = time.Time{}
					tm.platformReadyMutex.Unlock()
					return
				}
				tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", tLeft, 0, meetState.MeetName)
				tm.platformReadyMutex.Unlock()
			case <-ctx.Done():
				logger.Info.Printf("[startPlatformReadyTimer] Context cancelled for meet: %s", meetState.MeetName)
				return
			}
		}
	}(ctx, localTimerID)
}

// resetPlatformReadyTimer stops the platform-ready timer.
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

// startNextAttemptTimer adds a new next-attempt timer.
// It uses NextAttemptStartValue if set; otherwise defaults to 60.
func (tm *TimerManager) startNextAttemptTimer(meetState *MeetState) {
	tm.nextAttemptMutex.Lock()
	tm.nextAttemptIDCounter++
	timerID := tm.nextAttemptIDCounter

	startVal := 60
	if tm.NextAttemptStartValue > 0 {
		startVal = tm.NextAttemptStartValue
	}
	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: startVal,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	tm.nextAttemptMutex.Unlock()

	// Immediately broadcast the initial next-attempt timer state.
	broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

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

// interval returns the ticker interval, defaulting to 1 second if not set.
func (tm *TimerManager) interval() time.Duration {
	if tm.TickerInterval > 0 {
		return tm.TickerInterval
	}
	return 1 * time.Second
}

// HandleTimerAction processes different timer actions.
func (tm *TimerManager) HandleTimerAction(action, meetName string) {
	logger.Info.Printf("[HandleTimerAction] Received '%s' for meet '%s'", action, meetName)
	meetState := tm.Provider.GetMeetState(meetName)
	logger.Info.Printf("[HandleTimerAction] Got MeetState pointer %p for meet '%s'", meetState, meetName)

	switch action {
	case "startTimer":
		// Clear old decisions and broadcast clearResults.
		logger.Info.Printf("[HandleTimerAction] Clearing old decisions, sending 'clearResults' broadcast")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

		// Immediately broadcast the startTimer action.
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

// StartNextAttemptTimer Package-level wrapper for legacy code.
func StartNextAttemptTimer(meetState *MeetState) {
	if defaultTimerManager == nil {
		logger.Error.Println("defaultTimerManager is nil!")
		return
	}
	defaultTimerManager.startNextAttemptTimer(meetState)
}

// Package websocket - websocket/timer.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"sync"
	"time"
)

// Interfaces for dependency injection:
// These allow us to replace external dependencies with mocks in tests.

type StateProvider interface {
	GetMeetState(meetName string) *MeetState
}

type Messenger interface {
	BroadcastMessage(meetName string, msg map[string]interface{})
	BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string)
	BroadcastRaw(msg []byte)
}

// TimerManager manages timer actions using dependency injection.
// -----------------------------------------------------------------------------
// ✨ TickerInterval: lets tests override how often the ticker ticks (default is 1s)
// ✨ NextAttemptStartValue: lets tests override the starting value for next‑attempt timers (default is 60)
type TimerManager struct {
	Provider              StateProvider
	Messenger             Messenger
	TickerInterval        time.Duration
	NextAttemptStartValue int

	nextAttemptMutex     sync.Mutex
	platformReadyMutex   sync.Mutex
	nextAttemptIDCounter int
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
		logger.Info.Printf("[HandleTimerAction] Clearing old decisions, sending 'clearResults' broadcast")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON) // Broadcast clearResults.
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

// startPlatformReadyTimer starts a 60‑second timer for platform readiness.
func (tm *TimerManager) startPlatformReadyTimer(meetState *MeetState) {
	logger.Info.Printf("[startPlatformReadyTimer] Called for meet: %s", meetState.MeetName)
	tm.platformReadyMutex.Lock()
	defer tm.platformReadyMutex.Unlock()

	if meetState.PlatformReadyActive {
		logger.Warn.Printf("[startPlatformReadyTimer] Timer already active for meet: %s", meetState.MeetName)
		return
	}

	meetState.PlatformReadyActive = true
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)
	logger.Info.Printf("[startPlatformReadyTimer] Timer is set to 60s for meet: %s, endTime=%v",
		meetState.MeetName, meetState.PlatformReadyEnd)

	ticker := time.NewTicker(tm.interval())
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			tm.platformReadyMutex.Lock()
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
		}
	}()
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
			// ✨ Call an overridable function for broadcasting next-attempt timers.
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

// findTimerIndex returns the index of the timer with the given ID.
func findTimerIndex(timers []NextAttemptTimer, id int) int {
	for i, t := range timers {
		if t.ID == id {
			return i
		}
	}
	return -1
}

// -----------------------------------------------------------------------------
// Overridable broadcast function variable.
// In tests we can override this function (for example, to a no-op).
var broadcastAllNextAttemptTimersFunc = broadcastAllNextAttemptTimers

// Package-level default dependencies using dummy implementations.
// In production replace these with your actual implementations.
type dummyStateProvider struct{}

func (d *dummyStateProvider) GetMeetState(meetName string) *MeetState {
	return &MeetState{
		MeetName:          meetName,
		JudgeDecisions:    make(map[string]string),
		NextAttemptTimers: []NextAttemptTimer{},
	}
}

type dummyMessenger struct{}

func (d *dummyMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	logger.Info.Printf("dummyMessenger: BroadcastMessage to %s: %+v", meetName, msg)
}

func (d *dummyMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	logger.Info.Printf("dummyMessenger: BroadcastTimeUpdate for %s: action=%s, timeLeft=%d", meetName, action, timeLeft)
}

func (d *dummyMessenger) BroadcastRaw(msg []byte) {
	logger.Info.Printf("dummyMessenger: BroadcastRaw: %s", string(msg))
}

var defaultStateProvider StateProvider = &dummyStateProvider{}
var defaultMessenger Messenger = &dummyMessenger{}

var defaultTimerManager *TimerManager

func init() {
	defaultTimerManager = &TimerManager{
		Provider:              defaultStateProvider,
		Messenger:             defaultMessenger,
		TickerInterval:        1 * time.Second, // production default
		NextAttemptStartValue: 60,              // production default
	}
}

// StartNextAttemptTimer Package-level wrapper for legacy code.
func StartNextAttemptTimer(meetState *MeetState) {
	if defaultTimerManager == nil {
		logger.Error.Println("defaultTimerManager is nil!")
		return
	}
	defaultTimerManager.startNextAttemptTimer(meetState)
}

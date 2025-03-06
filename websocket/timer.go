// Package websocket - websocket/timer.go
package websocket

import (
	"encoding/json"
	"go-ref-lights/logger"
	"sync"
	"time"
)

// StateProvider is an interface for fetching MeetState objects.
type StateProvider interface {
	GetMeetState(meetName string) *MeetState
}

// Messenger is an interface for broadcasting messages.
type Messenger interface {
	BroadcastMessage(meetName string, msg map[string]interface{})
	BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string)
	BroadcastRaw(msg []byte)
}

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

type realMessenger struct{}

// BroadcastMessage marshals the message and sends it to all connections.
func (r *realMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("realMessenger: Error marshalling message: %v", err)
		return
	}
	broadcast <- m
	logger.Info.Printf("realMessenger: BroadcastMessage sent to meet %s", meetName)
}

// BroadcastTimeUpdate sends a time update message.
func (r *realMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	msg := map[string]interface{}{
		"action":   action,
		"index":    index,
		"timeLeft": timeLeft,
		"meetName": meetName,
	}
	m, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("realMessenger: Error marshalling time update: %v", err)
		return
	}
	broadcast <- m
	logger.Info.Printf("realMessenger: BroadcastTimeUpdate for meet %s: action=%s, timeLeft=%d", meetName, action, timeLeft)
}

// BroadcastRaw sends a raw JSON message.
func (r *realMessenger) BroadcastRaw(msg []byte) {
	broadcast <- msg
	logger.Info.Printf("realMessenger: BroadcastRaw sent: %s", string(msg))
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
		// CHANGED: Clear old decisions and broadcast clearResults.
		logger.Info.Printf("[HandleTimerAction] Clearing old decisions, sending 'clearResults' broadcast")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		tm.Messenger.BroadcastRaw(clearJSON)

		// CHANGED: Immediately broadcast the startTimer action.
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

	// CHANGED: Immediately broadcast the current platform ready time left.
	timeLeft := int(meetState.PlatformReadyEnd.Sub(time.Now()).Seconds())
	tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)

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

	// CHANGED: Immediately broadcast the initial next-attempt timer state.
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

// broadcastAllNextAttemptTimers sends a message with the current next-attempt timers.
func broadcastAllNextAttemptTimers(timers []NextAttemptTimer, meetName string) {
	msg := map[string]interface{}{
		"action":   "updateNextAttemptTime", // CHANGED: Use this action name for next-attempt updates.
		"timers":   timers,
		"meetName": meetName,
	}
	out, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("Error marshaling next attempt timers: %v", err)
		return
	}
	broadcastToMeet(meetName, out)
}

// ----- REALISTIC STATE PROVIDER IMPLEMENTATION -----
// CHANGED: Replace the dummy state provider with one that persists MeetState objects.
type realStateProvider struct {
	mu    sync.Mutex
	state map[string]*MeetState
}

// GetMeetState returns the persistent MeetState for the given meet name.
func (r *realStateProvider) GetMeetState(meetName string) *MeetState {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.state[meetName]; ok {
		return s
	}
	// Create a new MeetState if none exists.
	s := &MeetState{
		MeetName:          meetName,
		JudgeDecisions:    make(map[string]string),
		NextAttemptTimers: []NextAttemptTimer{},
	}
	r.state[meetName] = s
	return s
}

// ----- END REALISTIC STATE PROVIDER IMPLEMENTATION -----

// Package-level default dependencies.
// CHANGED: Use the realStateProvider instead of the dummy one.
var defaultStateProvider StateProvider = &realStateProvider{
	state: make(map[string]*MeetState),
}
var defaultMessenger Messenger = &realMessenger{}

// var defaultMessenger Messenger = &dummyMessenger{}
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

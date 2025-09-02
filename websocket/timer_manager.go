// Package websocket
/*
TimerManager coordinates concurrency for the platform readiness timer and next attempt timers.
Each function starts or stops a single timer, broadcasting countdown updates via WebSocket.
Concurrency details (like context cancellation and a per-second ticker) are handled internally.
*/
// File: websocket/timer_manager.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ref-lights/logger"
	"sync"
	"time"
)

// default instance of TimerManager.
var defaultTimerManager *TimerManager

// overridable function for broadcasting next attempt timers (used in tests).
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

// HandleTimerAction
/*
HandleTimerAction processes a specified timer action (e.g., "startTimer", "resetTimer",
or "startNextAttemptTimer") for the given meet. Depending on the action, this may reset
existing timers, clear judge decisions, or begin a fresh countdown. Broadcasts status
updates to connected clients as needed.
*/
func (tm *TimerManager) HandleTimerAction(action, meetName string) {
	// Convert routine timer actions to DEBUG level
	logContext := logger.NewTimerContext("timer_action_received", meetName, action, "")
	logger.LogDebugWithContext(logContext, "Processing timer action")

	meetState := tm.Provider.GetMeetState(meetName)
	if meetState == nil {
		// Keep ERROR level for missing meet state
		logContext := logger.NewTimerContext("meet_state_missing", meetName, action, "")
		logger.LogErrorWithContext(logContext, "Meet state not found for timer action")
		return
	}

	// Convert routine state pointer logging to DEBUG level
	logContext = logger.NewTimerContext("meet_state_accessed", meetName, action, "")
	logContext["statePointer"] = fmt.Sprintf("%p", meetState)
	logger.LogDebugWithContext(logContext, "Accessed MeetState for timer action")

	switch action {
	case "startTimer":
		// clear previous decisions and notify clients to clear results
		// Convert routine decision clearing to DEBUG level
		logContext = logger.NewTimerContext("decisions_cleared", meetName, "startTimer", "")
		logger.LogDebugWithContext(logContext, "Clearing old decisions and sending clearResults")
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			// Keep ERROR level for marshaling failures
			logContext = logger.NewTimerContext("clear_message_marshal_error", meetName, "startTimer", "")
			logContext = logger.AddError(logContext, err)
			logger.LogErrorWithContext(logContext, "Failed to marshal clearResults message")
			return
		}
		tm.Messenger.BroadcastRaw(clearJSON)

		// explicitly cancel any active platform ready timer
		CancelPlatformReadyTimer(meetName)

		// start the platform ready timer
		tm.Messenger.BroadcastMessage(meetName, map[string]interface{}{"action": "startTimer"})
		// Convert routine timer start to DEBUG level
		logContext = logger.NewTimerContext("platform_ready_starting", meetName, "startTimer", "")
		logger.LogDebugWithContext(logContext, "Starting platform ready timer")
		tm.startPlatformReadyTimer(meetState)

	case "resetTimer":
		// Convert routine timer reset to DEBUG level
		logContext = logger.NewTimerContext("timer_reset", meetName, "resetTimer", "")
		logger.LogDebugWithContext(logContext, "Processing resetTimer action")
		tm.resetPlatformReadyTimer(meetState)
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			// Keep ERROR level for marshaling failures
			logContext = logger.NewTimerContext("clear_message_marshal_error", meetName, "resetTimer", "")
			logContext = logger.AddError(logContext, err)
			logger.LogErrorWithContext(logContext, "Failed to marshal clearResults message")
			return
		}
		tm.Messenger.BroadcastRaw(clearJSON)

	case "startNextAttemptTimer":
		// Convert routine next attempt timer start to DEBUG level
		logContext = logger.NewTimerContext("next_attempt_starting", meetName, "startNextAttemptTimer", "")
		logger.LogDebugWithContext(logContext, "Starting next attempt timer")
		tm.startNextAttemptTimer(meetState)

	case "updatePlatformReadyTime":
		// the UI might be echoing updates; we typically ignore or no-op here
		// Keep as DEBUG level for routine update echoes
		logContext = logger.NewTimerContext("update_echo_ignored", meetName, "updatePlatformReadyTime", "")
		logger.LogDebugWithContext(logContext, "Ignoring timer update echo from client")
		return

	default:
		// Keep as DEBUG level for unrecognized actions
		logContext = logger.NewTimerContext("unrecognized_action", meetName, action, "")
		logger.LogDebugWithContext(logContext, "Timer action not recognized")
	}

	// Convert routine action completion to DEBUG level
	logContext = logger.NewTimerContext("timer_action_completed", meetName, action, "")
	logger.LogDebugWithContext(logContext, "Finished processing timer action")
}

// -------------------- platform ready timer management --------------------

// startPlatformReadyTimer
/*
Begins a 60-second countdown for the platform readiness phase,
cancelling any existing platform-ready timer. The function broadcasts remaining
time to connected clients until time runs out or the timer is reset/cancelled.
*/
func (tm *TimerManager) startPlatformReadyTimer(meetState *MeetState) {
	// Convert routine timer start to DEBUG level
	logContext := logger.NewTimerContext("platform_ready_timer_called", meetState.MeetName, "platform_ready", "")
	logger.LogDebugWithContext(logContext, "Platform ready timer function called")

	tm.platformReadyMutex.Lock()
	// cancel existing timer if running
	if meetState.PlatformReadyCancel != nil {
		meetState.PlatformReadyCancel()
		// Convert routine timer cancellation to DEBUG level
		logContext := logger.NewTimerContext("existing_timer_cancelled", meetState.MeetName, "platform_ready", meetState.PlatformReadyTimerID)
		logger.LogDebugWithContext(logContext, "Cancelled existing platform ready timer")
	}

	// create a new cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	meetState.PlatformReadyCtx = ctx
	meetState.PlatformReadyCancel = cancel

	// increment the timer ID for tracking
	meetState.PlatformReadyTimerID++
	localTimerID := meetState.PlatformReadyTimerID

	// set the single timer to active and store its end time
	meetState.PlatformReadyActive = true
	meetState.PlatformReadyEnd = time.Now().Add(60 * time.Second)
	// Convert routine timer setup to DEBUG level
	logContext = logger.NewTimerContext("platform_ready_timer_set", meetState.MeetName, "platform_ready", meetState.PlatformReadyTimerID)
	logContext["duration"] = "60s"
	logContext["endTime"] = meetState.PlatformReadyEnd.Format(time.RFC3339)
	logger.LogDebugWithContext(logContext, "Platform ready timer configured")
	tm.platformReadyMutex.Unlock()

	// clear lights and broadcast initial time left
	clearMsg := map[string]string{"action": "clearResults"}
	clearJSON, err := json.Marshal(clearMsg)
	if err != nil {
		// Keep ERROR level for marshaling failures
		logContext := logger.NewTimerContext("clear_message_marshal_error", meetState.MeetName, "platform_ready", localTimerID)
		logContext = logger.AddError(logContext, err)
		logger.LogErrorWithContext(logContext, "Failed to marshal clearResults message in platform ready timer")
		return
	}
	tm.Messenger.BroadcastRaw(clearJSON)

	timeLeft := int(time.Until(meetState.PlatformReadyEnd).Seconds())
	tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)

	// timer countdown using a ticker
	ticker := time.NewTicker(tm.interval())

	go func(ctx context.Context, timerID int) {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tm.platformReadyMutex.Lock()

				// if a new timer started, exit this one
				if meetState.PlatformReadyTimerID != timerID {
					// Convert routine timer ID mismatch to DEBUG level
					logContext := logger.NewTimerContext("timer_id_mismatch", meetState.MeetName, "platform_ready", timerID)
					logContext["currentTimerID"] = meetState.PlatformReadyTimerID
					logger.LogDebugWithContext(logContext, "Timer ID mismatch detected, exiting old timer")
					tm.platformReadyMutex.Unlock()
					return
				}

				// if the timer is no longer active, exit
				if !meetState.PlatformReadyActive {
					// Convert routine early timer stop to DEBUG level
					logContext := logger.NewTimerContext("timer_stopped_early", meetState.MeetName, "platform_ready", timerID)
					logger.LogDebugWithContext(logContext, "Timer was stopped early")
					tm.platformReadyMutex.Unlock()
					return
				}

				// calculate time left
				timeLeft := int(time.Until(meetState.PlatformReadyEnd).Seconds())
				if timeLeft < 0 {
					timeLeft = 0
				}

				// if time is up, broadcast and reset
				if timeLeft <= 0 {
					// Convert routine timer expiration to DEBUG level
					logContext := logger.NewTimerContext("timer_expired", meetState.MeetName, "platform_ready", timerID)
					logger.LogDebugWithContext(logContext, "Platform ready timer reached 0, marking expired")
					tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", 0, 0, meetState.MeetName)

					// Broadcast expiration message with error handling
					expirationMsg := []byte(`{"action":"platformReadyExpired"}`)
					tm.Messenger.BroadcastRaw(expirationMsg)

					meetState.PlatformReadyActive = false
					meetState.PlatformReadyEnd = time.Time{}
					tm.platformReadyMutex.Unlock()
					return
				}

				// otherwise, broadcast the updated time (routine countdown updates - no logging needed)
				tm.Messenger.BroadcastTimeUpdate("updatePlatformReadyTime", timeLeft, 0, meetState.MeetName)
				tm.platformReadyMutex.Unlock()

			case <-ctx.Done():
				// Convert routine context cancellation to DEBUG level
				logContext := logger.NewTimerContext("timer_context_cancelled", meetState.MeetName, "platform_ready", timerID)
				logger.LogDebugWithContext(logContext, "Timer context cancelled")
				return
			}
		}
	}(ctx, localTimerID)
}

// resetPlatformReadyTimer
/*
stops any ongoing platform readiness timer immediately and prevents further
time updates from being broadcast.
*/
func (tm *TimerManager) resetPlatformReadyTimer(meetState *MeetState) {
	tm.platformReadyMutex.Lock()
	defer tm.platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		// Keep WARN level for attempting to reset non-active timer
		context := logger.NewTimerContext("reset_inactive_timer", meetState.MeetName, "platform_ready", meetState.PlatformReadyTimerID)
		logger.LogWarnWithContext(context, "No active platform ready timer to reset")
		return
	}

	// Convert routine timer reset to DEBUG level
	context := logger.NewTimerContext("timer_reset_successful", meetState.MeetName, "platform_ready", meetState.PlatformReadyTimerID)
	logger.LogDebugWithContext(context, "Platform ready timer reset successfully")

	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// -------------------- next attempt timer management --------------------

// startNextAttemptTimer
/*
Creates a timer for the next attempt (default 60 seconds). It regularly
updates connected clients on the countdown, then marks the timer as inactive
once time expires or is reset.
*/
func (tm *TimerManager) startNextAttemptTimer(meetState *MeetState) {
	tm.nextAttemptMutex.Lock()
	tm.nextAttemptIDCounter++
	timerID := tm.nextAttemptIDCounter

	startVal := 60
	if tm.NextAttemptStartValue > 0 {
		startVal = tm.NextAttemptStartValue
	}

	// Convert routine timer creation to DEBUG level
	logContext := logger.NewTimerContext("next_attempt_timer_created", meetState.MeetName, "next_attempt", timerID)
	logContext["duration"] = fmt.Sprintf("%ds", startVal)
	logger.LogDebugWithContext(logContext, "Creating next attempt timer")

	deadline := time.Now().Add(time.Duration(startVal) * time.Second)
	newTimer := NextAttemptTimer{
		ID:       timerID,
		TimeLeft: startVal,
		Active:   true,
		EndTime:  deadline,
	}

	// add the new timer to the list
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	tm.nextAttemptMutex.Unlock()

	// broadcast the updated list of timers
	broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

	// start the countdown in a separate goroutine
	ticker := time.NewTicker(tm.interval())
	go func(id int) {
		defer ticker.Stop()
		for range ticker.C {
			tm.nextAttemptMutex.Lock()
			idx := findTimerIndex(meetState.NextAttemptTimers, id)

			// check if the timer is still in the list
			if idx == -1 {
				// timer not found; must've been removed or ended (routine - no logging needed)
				tm.nextAttemptMutex.Unlock()
				return
			}

			// check if the timer is still active
			if !meetState.NextAttemptTimers[idx].Active {
				// already inactive (routine - no logging needed)
				tm.nextAttemptMutex.Unlock()
				return
			}

			// recalc time left from EndTime
			timeLeft := int(time.Until(meetState.NextAttemptTimers[idx].EndTime).Seconds())
			if timeLeft < 0 {
				timeLeft = 0
			}

			// update the time left
			meetState.NextAttemptTimers[idx].TimeLeft = timeLeft

			// broadcast updated timers (routine countdown updates - no logging needed)
			broadcastAllNextAttemptTimersFunc(meetState.NextAttemptTimers, meetState.MeetName)

			if timeLeft <= 0 {
				// timer is done - convert to DEBUG level
				logContext := logger.NewTimerContext("next_attempt_timer_expired", meetState.MeetName, "next_attempt", id)
				logger.LogDebugWithContext(logContext, "Next attempt timer expired")
				meetState.NextAttemptTimers[idx].Active = false
				tm.nextAttemptMutex.Unlock()
				return
			}

			// still active; update the time left (routine - no logging needed)
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

// broadcastAllNextAttemptTimers
/*
packages the current list of next-attempt timers for a given meet into JSON
and sends it to all connected clients, ensuring the UI reflects accurate
countdown states.
*/
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
		// Keep ERROR level for marshaling failures
		context := logger.NewTimerContext("next_attempt_marshal_error", "", "next_attempt", "")
		context = logger.AddError(context, err)
		logger.LogErrorWithContext(context, "Failed to marshal next attempt timers")
		return
	}

	broadcastToMeet(meetName, out)
}

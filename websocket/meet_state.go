// Package websocket manages referee connections, meet state tracking, and real-time data updates.
// file: websocket/meet_state.go
package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// ------------- meet state structure --------------------------------

// NextAttemptTimer represents a timer tracking the countdown for the next attempt.
type NextAttemptTimer struct {
	ID       int  // Unique ID for identifying the timer
	TimeLeft int  // Time remaining in seconds
	Active   bool // Indicates if the timer is currently active
}

// MeetState manages real-time, in-memory data for an active meet.
type MeetState struct {
	MeetName              string                     // Name of the meet
	RefereeSessions       map[string]*websocket.Conn // Active referee WebSocket connections
	JudgeDecisions        map[string]string          // Stores decisions from judges (left, center, right)
	PlatformReadyActive   bool                       // Indicates if the "Platform Ready" timer is active
	PlatformReadyTimeLeft int                        // Remaining time for platform readiness
	PlatformReadyEnd      time.Time                  // Timestamp when platform readiness ends
	NextAttemptTimers     []NextAttemptTimer         // List of next attempt timers

	// Context for managing the platform readiness timer
	PlatformReadyCtx     context.Context
	PlatformReadyCancel  context.CancelFunc
	PlatformReadyTimerID int
}

// ------------- meet state storage --------------------------------

// Global storage for meet states.
var (
	meets      = make(map[string]*MeetState)
	meetsMutex = &sync.Mutex{}
)

// ------------- meet state management --------------------------------

// getMeetState retrieves or initialises a MeetState for the given meetName
func getMeetState(meetName string) *MeetState {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	state, exists := meets[meetName]
	if !exists {
		logger.Info.Printf("[getMeetState] Creating new MeetState for meet: %s", meetName)
		state = &MeetState{
			MeetName:              meetName,
			RefereeSessions:       make(map[string]*websocket.Conn),
			JudgeDecisions:        make(map[string]string),
			PlatformReadyActive:   false,
			PlatformReadyTimeLeft: 60, // default to 60s todo: move to const
			NextAttemptTimers:     []NextAttemptTimer{},
			PlatformReadyCancel:   nil,
		}
		meets[meetName] = state
	} else {
		logger.Debug.Printf("[getMeetState] Retrieved existing MeetState for meet: %s", meetName)
	}

	// log before cancelling old timers
	logger.Debug.Printf("[getMeetState] Active Timer: %v, CancelFunc Exists: %v", state.PlatformReadyActive, state.PlatformReadyCancel != nil)

	// ensure no duplicate timers are running
	if state.PlatformReadyCancel != nil {
		logger.Info.Printf("[getMeetState] Cancelling old timer for meet: %s", meetName)
		state.PlatformReadyCancel()
		state.PlatformReadyCancel = nil
		state.PlatformReadyActive = false
	}
	return state
}

// ClearMeetState removes the MeetState for a given meetName.
// Used when a meet is completed or in case of an error requiring clean-up.
func ClearMeetState(meetName string) {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	if _, exists := meets[meetName]; exists {
		delete(meets, meetName)
		logger.Info.Printf("Cleared MeetState for meet: %s", meetName)
	} else {
		logger.Warn.Printf("Attempted to clear non-existent MeetState for meet: %s", meetName)
	}
}

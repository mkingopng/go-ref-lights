// Package websocket unified-state.go provides a unified state provider for the WebSocket server
// File: websocket/unified_state.go
package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// StateProvider is an interface for fetching MeetState objects
type StateProvider interface {
	GetMeetState(meetName string) *MeetState
}

// MeetState holds all state for a meet including timer information and judge decisions
type MeetState struct {
	MeetName              string                     // name of the meet
	RefereeSessions       map[string]*websocket.Conn // active referee WebSocket connections
	JudgeDecisions        map[string]string          // judge decisions (e.g., left, center, right)
	PlatformReadyActive   bool                       // is the Platform Ready timer active?
	PlatformReadyTimeLeft int                        // remaining seconds on the timer
	PlatformReadyEnd      time.Time                  // time when the timer expires
	NextAttemptTimers     []NextAttemptTimer         // next attempt timers
	PlatformReadyCtx      context.Context            // context for the Platform Ready timer
	PlatformReadyCancel   context.CancelFunc         // cancel function for the timer
	PlatformReadyTimerID  int                        // unique timer ID to help cancel stale timers
}

// NextAttemptTimer represents a timer for the next attempt
type NextAttemptTimer struct {
	ID       int       // unique ID for the timer
	TimeLeft int       // time remaining in seconds
	Active   bool      // is the timer active?
	EndTime  time.Time // for convenience, we store the end time
}

// global map and mutex to store MeetState instances
var (
	meets      = make(map[string]*MeetState)
	meetsMutex = &sync.Mutex{}
)

// GetMeetState returns the MeetState for a given meetName. If none exists, it creates a new one
func GetMeetState(meetName string) *MeetState {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	state, exists := meets[meetName]
	if !exists {
		logger.Info.Printf("[GetMeetState] Creating new MeetState for meet=%s", meetName)
		state = &MeetState{
			MeetName:              meetName,
			RefereeSessions:       make(map[string]*websocket.Conn),
			JudgeDecisions:        make(map[string]string),
			NextAttemptTimers:     []NextAttemptTimer{},
			PlatformReadyTimeLeft: 60, // Default (60 seconds)
		}
		meets[meetName] = state
	} else {
		logger.Debug.Printf("[GetMeetState] Retrieved existing MeetState for meet=%s", meetName)
	}

	return state
}

// CancelPlatformReadyTimer explicitly cancels any active platform ready timer for the given meet
func CancelPlatformReadyTimer(meetName string) {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	if state, exists := meets[meetName]; exists {
		if state.PlatformReadyCancel != nil {
			logger.Info.Printf("[CancelPlatformReadyTimer] Cancelling existing platform ready timer for meet=%s", meetName)
			state.PlatformReadyCancel()
			state.PlatformReadyCancel = nil
			state.PlatformReadyActive = false
		}
	}
}

// ClearMeetState removes a MeetState for a given meetName
func ClearMeetState(meetName string) {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	if _, exists := meets[meetName]; exists {
		delete(meets, meetName)
		logger.Info.Printf("[ClearMeetState] Cleared MeetState for meet=%s", meetName)
	} else {
		logger.Warn.Printf("[ClearMeetState] Attempted to clear non-existent MeetState for meet=%s", meetName)
	}
}

// UnifiedStateProvider implements the StateProvider interface using the global meets map
type UnifiedStateProvider struct{}

// GetMeetState returns the MeetState for the given meetName using our unified method
func (usp *UnifiedStateProvider) GetMeetState(meetName string) *MeetState {
	return GetMeetState(meetName)
}

// DefaultStateProvider is the unified state provider that all components should use
var DefaultStateProvider StateProvider = &UnifiedStateProvider{}

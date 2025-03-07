// Package websocket manages referee connections, meet state tracking, and real-time data updates.
// file: websocket/meet_state.go
package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// NextAttemptTimer structure used to track next attempt timers.
type NextAttemptTimer struct {
	ID       int
	TimeLeft int
	Active   bool
}

// MeetState holds per-meet, in-memory data.
type MeetState struct {
	MeetName              string
	RefereeSessions       map[string]*websocket.Conn
	JudgeDecisions        map[string]string
	PlatformReadyActive   bool
	PlatformReadyTimeLeft int
	PlatformReadyEnd      time.Time
	NextAttemptTimers     []NextAttemptTimer
}

// a global map storing meetName -> *MeetState
var (
	meets      = make(map[string]*MeetState)
	meetsMutex = &sync.Mutex{}
)

// getMeetState fetches or creates a MeetState for the given meetName.
func getMeetState(meetName string) *MeetState {
	meetsMutex.Lock()
	defer meetsMutex.Unlock()

	state, exists := meets[meetName]
	if !exists {
		logger.Info.Printf("Creating new MeetState for meet: %s", meetName)
		state = &MeetState{
			MeetName:              meetName, // ← This is the added field!
			RefereeSessions:       make(map[string]*websocket.Conn),
			JudgeDecisions:        make(map[string]string),
			PlatformReadyActive:   false,
			PlatformReadyTimeLeft: 60,
			NextAttemptTimers:     []NextAttemptTimer{},
		}
		meets[meetName] = state
	} else {
		logger.Debug.Printf("Retrieved existing MeetState for meet: %s", meetName)
	}
	return state
}

// ClearMeetState removes the MeetState for the given meetName.
// This can be used when a meet is finished, or if an error condition warrants
// clean-up.
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

// Package websocket: This file contains the MeetState struct, which holds per-meet, in-memory data
package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
	"go-ref-lights/logger"
)

// NextAttemptTimer structure used to track next attempt timers.
type NextAttemptTimer struct {
	ID       int // Unique identifier for this timer.
	TimeLeft int
	Active   bool
}

// MeetState holds per-meet, in-memory data.
type MeetState struct {
	MeetName string
	// judgeID -> WebSocket connection (e.g. "left" -> conn)
	RefereeSessions map[string]*websocket.Conn
	// judgeID -> decision string (e.g. "left" -> "white")
	JudgeDecisions map[string]string
	// whether the Platform Ready timer is active, and the time left
	PlatformReadyActive   bool
	PlatformReadyTimeLeft int

	// nextAttempt timers, multiple can run concurrently
	NextAttemptTimers []NextAttemptTimer
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
// This can be used when a meet is finished or if an error condition warrants
// cleanup.  // todo: where should i use this?
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

// DecisionMessage now includes MeetName so we can handle multiple meets.
type DecisionMessage struct {
	MeetName string `json:"meetName,omitempty"` // e.g. "STATE_CHAMPS_2025"
	JudgeID  string `json:"judgeId,omitempty"`
	Decision string `json:"decision,omitempty"`
	Action   string `json:"action,omitempty"`
}

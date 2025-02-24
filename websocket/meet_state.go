// Package websocket: This file contains the MeetState struct, which holds per-meet, in-memory data
package websocket

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// NextAttemptTimer structure (moved here from handler.go to be used by MeetState)
type NextAttemptTimer struct {
	TimeLeft int
	Active   bool
}

// MeetState holds per-meet, in-memory data.
type MeetState struct {
	// judgeID -> WebSocket connection (e.g. "left" -> conn)
	RefereeSessions map[string]*websocket.Conn

	// judgeID -> decision string (e.g. "left" -> "white")
	JudgeDecisions map[string]string

	// whether the Platform Ready timer is active, and the time left
	PlatformReadyActive   bool
	PlatformReadyTimeLeft int

	// nextAttempt timers, if multiple can run concurrently
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
		log.Printf("Creating new MeetState for meet: %s", meetName)
		state = &MeetState{
			RefereeSessions:       make(map[string]*websocket.Conn),
			JudgeDecisions:        make(map[string]string),
			PlatformReadyActive:   false,
			PlatformReadyTimeLeft: 60,
			NextAttemptTimers:     []NextAttemptTimer{},
		}
		meets[meetName] = state
	}
	return state
}

// DecisionMessage now includes MeetName so we can handle multiple meets.
type DecisionMessage struct {
	MeetName string `json:"meetName,omitempty"` // e.g. "STATE_CHAMPS_2025"
	JudgeID  string `json:"judgeId,omitempty"`
	Decision string `json:"decision,omitempty"`
	Action   string `json:"action,omitempty"`
}

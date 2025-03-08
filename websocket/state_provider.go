// Package websocket - this file contains the implementation of the
// realStateProvider struct and its methods.
// file: websocket/state_provider.go
package websocket

import "sync"

// StateProvider is an interface for fetching MeetState objects.
type StateProvider interface {
	GetMeetState(meetName string) *MeetState
}

// realStateProvider is an implementation of the StateProvider that persists
type realStateProvider struct {
	mu    sync.Mutex
	state map[string]*MeetState
}

// Use the realStateProvider instead of the dummy one.
var defaultStateProvider StateProvider = &realStateProvider{
	state: make(map[string]*MeetState),
}

// --------------- Methods on realStateProvider -----------------

// GetMeetState returns the persistent MeetState for the given meet name.
func (r *realStateProvider) GetMeetState(meetName string) *MeetState {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.state[meetName]; ok {
		return s
	}

	// create a new MeetState if none exists.
	s := &MeetState{
		MeetName:          meetName,
		JudgeDecisions:    make(map[string]string),
		NextAttemptTimers: []NextAttemptTimer{},
	}
	r.state[meetName] = s
	return s
}

// websocket/meet_state_test.go

//go:build unit
// +build unit

package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

// Test `getMeetState` creates a new MeetState when none exists
func TestGetMeetState_CreatesNewState(t *testing.T) {
	InitTest()
	meetName := "TestMeet1"

	// Ensure the state is cleared before test
	ClearMeetState(meetName)

	state := getMeetState(meetName)

	// Ensure a new MeetState was created
	assert.NotNil(t, state, "MeetState should not be nil")
	assert.Equal(t, meetName, state.MeetName, "MeetName should match requested name")
	assert.False(t, state.PlatformReadyActive, "PlatformReadyActive should be false by default")
	assert.Equal(t, 60, state.PlatformReadyTimeLeft, "PlatformReadyTimeLeft should default to 60")
	assert.Empty(t, state.RefereeSessions, "RefereeSessions should be empty initially")
	assert.Empty(t, state.JudgeDecisions, "JudgeDecisions should be empty initially")
	assert.Empty(t, state.NextAttemptTimers, "NextAttemptTimers should be empty initially")
}

// Test `getMeetState` retrieves an existing MeetState without creating a new one
func TestGetMeetState_RetrievesExistingState(t *testing.T) {
	InitTest()
	meetName := "TestMeet2"

	// Ensure the state is cleared before test
	ClearMeetState(meetName)

	// Create a state
	initialState := getMeetState(meetName)
	initialState.PlatformReadyActive = true
	initialState.JudgeDecisions["judge1"] = "white"

	// Retrieve the state again
	retrievedState := getMeetState(meetName)

	// Ensure the same instance is returned
	assert.Equal(t, initialState, retrievedState, "Should return the same MeetState instance")
	assert.True(t, retrievedState.PlatformReadyActive, "PlatformReadyActive should be true")
	assert.Equal(t, "white", retrievedState.JudgeDecisions["judge1"], "JudgeDecisions should persist")
}

// Test `ClearMeetState` removes the MeetState correctly
func TestClearMeetState_RemovesMeetState(t *testing.T) {
	InitTest()
	meetName := "TestMeet3"

	// Ensure the state is cleared before test
	ClearMeetState(meetName)

	// Create state
	getMeetState(meetName)

	// Ensure state exists
	meetsMutex.Lock()
	_, exists := meets[meetName]
	meetsMutex.Unlock()
	assert.True(t, exists, "MeetState should exist after creation")

	// Clear state
	ClearMeetState(meetName)

	// Ensure state is removed
	meetsMutex.Lock()
	_, exists = meets[meetName]
	meetsMutex.Unlock()
	assert.False(t, exists, "MeetState should be removed after ClearMeetState is called")
}

// Test `getMeetState` is thread-safe under concurrent access
func TestGetMeetState_ThreadSafety(t *testing.T) {
	InitTest()
	meetName := "TestMeet4"
	ClearMeetState(meetName)

	var wg sync.WaitGroup
	numRoutines := 100
	states := make([]*MeetState, numRoutines)

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			states[index] = getMeetState(meetName)
		}(i)
	}

	wg.Wait()

	// Ensure all goroutines return the same MeetState instance
	for i := 1; i < numRoutines; i++ {
		assert.Equal(t, states[0], states[i], "All goroutines should return the same MeetState instance")
	}
}

// Test `RefereeSessions` and `JudgeDecisions` behave as expected
func TestMeetState_RefereeSessionsAndJudgeDecisions(t *testing.T) {
	InitTest()
	meetName := "TestMeet5"
	ClearMeetState(meetName)

	state := getMeetState(meetName)

	// Simulate adding referee sessions
	var mockConn *websocket.Conn // Properly handle WebSocket connection
	state.RefereeSessions["ref1"] = mockConn
	state.RefereeSessions["ref2"] = mockConn

	// Simulate judge decisions
	state.JudgeDecisions["judge1"] = "white"
	state.JudgeDecisions["judge2"] = "red"

	assert.Equal(t, 2, len(state.RefereeSessions), "RefereeSessions should contain 2 refs")
	assert.Equal(t, 2, len(state.JudgeDecisions), "JudgeDecisions should contain 2 decisions")
	assert.Equal(t, "white", state.JudgeDecisions["judge1"], "Judge1 decision should be white")
	assert.Equal(t, "red", state.JudgeDecisions["judge2"], "Judge2 decision should be red")
}

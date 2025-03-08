// file: websocket/unified_state_test.go
//go:build unit
// +build unit

package websocket

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestGetMeetStateCreatesNewState verifies that a new MeetState is created with default values.
func TestGetMeetStateCreatesNewState(t *testing.T) {
	// Ensure we start fresh.
	ClearMeetState("TestMeet1")
	state := GetMeetState("TestMeet1")
	require.NotNil(t, state)
	assert.Equal(t, "TestMeet1", state.MeetName, "MeetName should be set")
	assert.Equal(t, 60, state.PlatformReadyTimeLeft, "Default PlatformReadyTimeLeft should be 60")
	assert.Empty(t, state.JudgeDecisions, "JudgeDecisions map should be empty")
	assert.Empty(t, state.NextAttemptTimers, "NextAttemptTimers slice should be empty")
	assert.NotNil(t, state.RefereeSessions, "RefereeSessions map should be non-nil")
}

// TestGetMeetStateRetrievesExistingState verifies that calling GetMeetState twice returns the same state.
func TestGetMeetStateRetrievesExistingState(t *testing.T) {
	ClearMeetState("TestMeet2")
	state1 := GetMeetState("TestMeet2")
	// Modify the state.
	state1.JudgeDecisions["left"] = "good"
	state2 := GetMeetState("TestMeet2")
	assert.Equal(t, state1, state2, "Expected the same state instance")
	assert.Equal(t, "good", state2.JudgeDecisions["left"], "Modified JudgeDecision should persist")
}

// TestGetMeetStateCancelsExistingTimer verifies that an active timer is cancelled on a later call.
func TestGetMeetStateCancelsExistingTimer(t *testing.T) {
	// Ensure no state exists for "TestMeet3"
	ClearMeetState("TestMeet3")
	state := GetMeetState("TestMeet3")

	// Set a dummy cancel function to simulate an active timer.
	cancelled := false
	state.PlatformReadyCancel = func() { cancelled = true }
	state.PlatformReadyActive = true

	// Instead of calling GetMeetState (which no longer cancels timers),
	// we now explicitly cancel the timer.
	CancelPlatformReadyTimer("TestMeet3")

	assert.True(t, cancelled, "Existing timer should have been cancelled")
	assert.False(t, state.PlatformReadyActive, "PlatformReadyActive should be false after cancellation")
	assert.Nil(t, state.PlatformReadyCancel, "PlatformReadyCancel should be nil after cancellation")
}

// TestClearMeetState verifies that ClearMeetState removes a MeetState.
func TestClearMeetState(t *testing.T) {
	// Create a state.
	state1 := GetMeetState("TestMeet4")
	require.NotNil(t, state1)
	// Clear the state.
	ClearMeetState("TestMeet4")
	// Get a new state; a pointer should differ.
	state2 := GetMeetState("TestMeet4")
	// Use NotSame to verify that the two pointers are different.
	assert.NotSame(t, state1, state2, "After clearing, GetMeetState should create a new instance")
}

// TestUnifiedStateProvider_GetMeetState verifies that the unified provider returns the same state as GetMeetState.
func TestUnifiedStateProvider_GetMeetState(t *testing.T) {
	ClearMeetState("TestMeet5")
	provider := DefaultStateProvider
	state1 := provider.GetMeetState("TestMeet5")
	state2 := GetMeetState("TestMeet5")
	assert.Equal(t, state1, state2, "UnifiedStateProvider should return the same state as GetMeetState")
}

// TestGetMeetStateConcurrency verifies that concurrent calls to GetMeetState return the same state.
func TestGetMeetStateConcurrency(t *testing.T) {
	ClearMeetState("ConcurrentMeet")
	const count = 100
	var wg sync.WaitGroup
	results := make([]*MeetState, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = GetMeetState("ConcurrentMeet")
		}(i)
	}
	wg.Wait()

	first := results[0]
	for i, state := range results {
		assert.Equal(t, first, state, "Result at index %d should be equal", i)
	}
}

// Optional: TestClearMeetStateForNonexistentMeet verifies that clearing a non-existent state does not panic.
func TestClearMeetStateForNonexistentMeet(t *testing.T) {
	// Ensure that calling ClearMeetState on a meet name that doesn't exist does not panic.
	assert.NotPanics(t, func() {
		ClearMeetState("NonexistentMeet")
	}, "Clearing a nonexistent meet state should not panic")
}

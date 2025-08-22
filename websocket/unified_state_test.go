//go:build unit
// +build unit

// file: websocket/unified_state_test.go
package websocket

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetMeetStateCreatesNewState verifies that a new MeetState is created with default values
func TestGetMeetStateCreatesNewState(t *testing.T) {
	ClearMeetState("TestMeet1")
	state := GetMeetState("TestMeet1")
	require.NotNil(t, state)
	assert.Equal(t, "TestMeet1", state.MeetName, "MeetName should be set")
	assert.Equal(t, 60, state.PlatformReadyTimeLeft, "Default PlatformReadyTimeLeft should be 60")
	assert.Empty(t, state.JudgeDecisions, "JudgeDecisions map should be empty")
	assert.Empty(t, state.NextAttemptTimers, "NextAttemptTimers slice should be empty")
	assert.NotNil(t, state.RefereeSessions, "RefereeSessions map should be non-nil")
}

// TestGetMeetStateRetrievesExistingState verifies that calling GetMeetState twice returns the same state
func TestGetMeetStateRetrievesExistingState(t *testing.T) {
	ClearMeetState("TestMeet2")
	state1 := GetMeetState("TestMeet2")
	state1.JudgeDecisions["left"] = "good"
	state2 := GetMeetState("TestMeet2")
	assert.Equal(t, state1, state2, "Expected the same state instance")
	assert.Equal(t, "good", state2.JudgeDecisions["left"], "Modified JudgeDecision should persist")
}

// TestGetMeetStateCancelsExistingTimer verifies that an active timer is cancelled on a later call.
func TestGetMeetStateCancelsExistingTimer(t *testing.T) {
	ClearMeetState("TestMeet3")
	state := GetMeetState("TestMeet3")

	// set a dummy cancel function to simulate an active timer
	cancelled := false
	state.PlatformReadyCancel = func() { cancelled = true }
	state.PlatformReadyActive = true

	// instead of calling GetMeetState (which no longer cancels timers), we now explicitly cancel the timer.
	CancelPlatformReadyTimer("TestMeet3")

	assert.True(t, cancelled, "Existing timer should have been cancelled")
	assert.False(t, state.PlatformReadyActive, "PlatformReadyActive should be false after cancellation")
	assert.Nil(t, state.PlatformReadyCancel, "PlatformReadyCancel should be nil after cancellation")
}

// TestClearMeetState verifies that ClearMeetState removes a MeetState.
func TestClearMeetState(t *testing.T) {
	state1 := GetMeetState("TestMeet4")
	require.NotNil(t, state1)
	ClearMeetState("TestMeet4")
	state2 := GetMeetState("TestMeet4")
	assert.NotSame(t, state1, state2, "After clearing, GetMeetState should create a new instance")
}

// TestUnifiedStateProvider_GetMeetState verifies that the unified provider returns the same state as GetMeetState
func TestUnifiedStateProvider_GetMeetState(t *testing.T) {
	ClearMeetState("TestMeet5")
	provider := DefaultStateProvider
	state1 := provider.GetMeetState("TestMeet5")
	state2 := GetMeetState("TestMeet5")
	assert.Equal(t, state1, state2, "UnifiedStateProvider should return the same state as GetMeetState")
}

// TestGetMeetStateConcurrency verifies that concurrent calls to GetMeetState return the same state
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

//// Optional: TestClearMeetStateForNonexistentMeet verifies that clearing a non-existent state does not panic.
//func TestClearMeetStateForNonexistentMeet(t *testing.T) {
//	ClearMeetState("NonexistentMeet")
//}

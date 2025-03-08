// file: websocket/state_provider_test.go
//go:build unit
// +build unit

package websocket

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMeetState_New(t *testing.T) {
	provider := &realStateProvider{
		state: make(map[string]*MeetState),
	}
	meet := provider.GetMeetState("TestMeet")
	assert.NotNil(t, meet, "Expected a non-nil MeetState")
	assert.Equal(t, "TestMeet", meet.MeetName, "MeetName should match the provided name")
	assert.NotNil(t, meet.JudgeDecisions, "JudgeDecisions map should be initialized")
	assert.Equal(t, 0, len(meet.JudgeDecisions), "JudgeDecisions should be empty")
	assert.NotNil(t, meet.NextAttemptTimers, "NextAttemptTimers slice should be initialized")
	assert.Equal(t, 0, len(meet.NextAttemptTimers), "NextAttemptTimers should be empty")
}

func TestGetMeetState_Existing(t *testing.T) {
	provider := &realStateProvider{
		state: make(map[string]*MeetState),
	}
	meet1 := provider.GetMeetState("TestMeet")
	meet2 := provider.GetMeetState("TestMeet")
	assert.Equal(t, meet1, meet2, "Expected the same MeetState instance for the same meet name")
}

func TestGetMeetState_Concurrent(t *testing.T) {
	provider := &realStateProvider{
		state: make(map[string]*MeetState),
	}
	const numRoutines = 20
	var wg sync.WaitGroup
	meetName := "ConcurrentMeet"
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			provider.GetMeetState(meetName)
		}()
	}
	wg.Wait()

	// After concurrent calls, all instances should be identical.
	meet1 := provider.GetMeetState(meetName)
	meet2 := provider.GetMeetState(meetName)
	assert.Equal(t, meet1, meet2, "Concurrent GetMeetState should return the same instance")
}

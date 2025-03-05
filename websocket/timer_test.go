// websocket/timer_test.go

//go:build unit
// +build unit

package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock implementations using testify/mock ---

type MockStateProvider struct {
	mock.Mock
}

func (m *MockStateProvider) GetMeetState(meetName string) *MeetState {
	args := m.Called(meetName)
	return args.Get(0).(*MeetState)
}

type MockMessenger struct {
	mock.Mock
}

func (m *MockMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	m.Called(meetName, msg)
}

func (m *MockMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	m.Called(action, timeLeft, index, meetName)
}

func (m *MockMessenger) BroadcastRaw(msg []byte) {
	m.Called(msg)
}

// Test: startTimer action should clear JudgeDecisions and start the platform timer.
func TestTimerManager_HandleTimerAction_StartTimer(t *testing.T) {
	InitTest()
	meetState := &MeetState{
		MeetName:       "TestMeet",
		JudgeDecisions: map[string]string{"initial": "value"},
	}

	mockProvider := new(MockStateProvider)
	mockMessenger := new(MockMessenger)
	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)

	// Expect a clearResults broadcast.
	mockMessenger.
		On("BroadcastRaw", []byte(`{"action":"clearResults"}`)).
		Once()
	// Expect a BroadcastMessage with action "startTimer".
	mockMessenger.
		On("BroadcastMessage", "TestMeet", mock.MatchedBy(func(msg map[string]interface{}) bool {
			return msg["action"] == "startTimer"
		})).
		Once()
	// Expect a platformReadyExpired broadcast when the timer expires.
	mockMessenger.
		On("BroadcastRaw", []byte(`{"action":"platformReadyExpired"}`)).
		Once()
	// Optionally allow BroadcastTimeUpdate calls.
	mockMessenger.
		On("BroadcastTimeUpdate", "updatePlatformReadyTime", mock.Anything, 0, "TestMeet").
		Maybe()

	tm := &TimerManager{
		Provider:             mockProvider,
		Messenger:            mockMessenger,
		nextAttemptIDCounter: 0,
	}

	tm.HandleTimerAction("startTimer", "TestMeet")

	assert.Equal(t, 0, len(meetState.JudgeDecisions), "JudgeDecisions should be cleared")
	assert.True(t, meetState.PlatformReadyActive, "PlatformReadyActive should be true")

	// Force timer expiry.
	meetState.PlatformReadyEnd = time.Now().Add(-1 * time.Second)
	time.Sleep(1100 * time.Millisecond)

	mockProvider.AssertExpectations(t)
	mockMessenger.AssertExpectations(t)
}

// Test: startNextAttemptTimer with fast ticker and short start value.
func TestTimerManager_HandleTimerAction_StartNextAttemptTimer(t *testing.T) {
	InitTest()
	// Override broadcastAllNextAttemptTimersFunc to no-op to prevent delays.
	oldBroadcast := broadcastAllNextAttemptTimersFunc
	broadcastAllNextAttemptTimersFunc = func(timers []NextAttemptTimer, meetName string) {}
	defer func() { broadcastAllNextAttemptTimersFunc = oldBroadcast }()

	meetState := &MeetState{
		MeetName:          "TestMeet",
		NextAttemptTimers: []NextAttemptTimer{},
	}
	mockProvider := new(MockStateProvider)
	mockMessenger := new(MockMessenger)
	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)

	// Create TimerManager with fast ticker and a very low starting value.
	tm := &TimerManager{
		Provider:              mockProvider,
		Messenger:             mockMessenger,
		NextAttemptStartValue: 1,                     // start at 1 second
		TickerInterval:        10 * time.Millisecond, // tick every 10ms
		nextAttemptIDCounter:  0,
	}

	tm.HandleTimerAction("startNextAttemptTimer", "TestMeet")

	assert.Equal(t, 1, len(meetState.NextAttemptTimers), "Expected one NextAttemptTimer")
	timer := meetState.NextAttemptTimers[0]
	assert.Equal(t, 1, timer.TimeLeft, "Timer should start at 1 second")
	assert.True(t, timer.Active, "Timer should be active")

	// Wait enough time for the timer goroutine to expire the timer.
	time.Sleep(50 * time.Millisecond)

	tm.nextAttemptMutex.Lock()
	updatedTimer := meetState.NextAttemptTimers[0]
	tm.nextAttemptMutex.Unlock()
	assert.False(t, updatedTimer.Active, "Expected timer to be inactive after expiration")
	mockProvider.AssertExpectations(t)
}

// Test: resetTimer action should clear JudgeDecisions and stop the platform timer.
func TestTimerManager_HandleTimerAction_ResetTimer(t *testing.T) {
	InitTest()
	meetState := &MeetState{
		MeetName:              "TestMeet",
		JudgeDecisions:        map[string]string{"decision": "value"},
		PlatformReadyActive:   true,
		PlatformReadyTimeLeft: 30,
	}
	mockProvider := new(MockStateProvider)
	mockMessenger := new(MockMessenger)
	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)

	mockMessenger.
		On("BroadcastRaw", mock.MatchedBy(func(msg []byte) bool {
			var m map[string]string
			_ = json.Unmarshal(msg, &m)
			return m["action"] == "clearResults"
		})).
		Once()

	tm := &TimerManager{
		Provider:  mockProvider,
		Messenger: mockMessenger,
	}

	tm.HandleTimerAction("resetTimer", "TestMeet")

	assert.Equal(t, 0, len(meetState.JudgeDecisions), "JudgeDecisions should be cleared after resetTimer")
	assert.False(t, meetState.PlatformReadyActive, "PlatformReadyActive should be false after resetTimer")
	assert.Equal(t, 60, meetState.PlatformReadyTimeLeft, "PlatformReadyTimeLeft should be 60 after resetTimer")

	mockProvider.AssertExpectations(t)
	mockMessenger.AssertExpectations(t)
}

// Test: updatePlatformReadyTime action should leave JudgeDecisions unchanged.
func TestTimerManager_HandleTimerAction_UpdatePlatformReadyTime(t *testing.T) {
	InitTest()
	meetState := &MeetState{
		MeetName:       "TestMeet",
		JudgeDecisions: map[string]string{"initial": "value"},
	}
	mockProvider := new(MockStateProvider)
	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)

	tm := &TimerManager{
		Provider:  mockProvider,
		Messenger: new(MockMessenger),
	}

	initialCount := len(meetState.JudgeDecisions)
	tm.HandleTimerAction("updatePlatformReadyTime", "TestMeet")
	assert.Equal(t, initialCount, len(meetState.JudgeDecisions), "updatePlatformReadyTime should not modify JudgeDecisions")
	mockProvider.AssertExpectations(t)
}

// Test: invalid action should not modify JudgeDecisions.
func TestTimerManager_HandleTimerAction_InvalidAction(t *testing.T) {
	InitTest()
	meetState := &MeetState{
		MeetName:       "TestMeet",
		JudgeDecisions: map[string]string{"initial": "value"},
	}
	mockProvider := new(MockStateProvider)
	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)

	tm := &TimerManager{
		Provider:  mockProvider,
		Messenger: new(MockMessenger),
	}

	tm.HandleTimerAction("invalidAction", "TestMeet")
	assert.Equal(t, 1, len(meetState.JudgeDecisions), "Invalid action should not modify JudgeDecisions")
	mockProvider.AssertExpectations(t)
}

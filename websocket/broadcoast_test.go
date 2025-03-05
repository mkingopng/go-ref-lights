// file: websocket/broadcast_test.go

//go:build unit
// +build unit

package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockBroadcast is a buffered channel that we use to override the global broadcast.
var mockBroadcast = make(chan []byte, 10)

// In init, override the global broadcast channel.
func init() {
	broadcast = mockBroadcast
}

// Helper function to flush the broadcast channel.
func flushBroadcastChannel() {
	for len(mockBroadcast) > 0 {
		<-mockBroadcast
	}
}

// TestBroadcastMessage_Success verifies that BroadcastMessage correctly marshals and sends a message.
func TestBroadcastMessage_Success(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	message := map[string]interface{}{
		"action": "testAction",
		"data":   "testData",
	}

	BroadcastMessage("APL Test Meet", message)

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]interface{}
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "testAction", decoded["action"])
		assert.Equal(t, "testData", decoded["data"])
	default:
		t.Fatal("Expected message in broadcast channel, but got none")
	}
}

// TestBroadcastFinalResults verifies that broadcastFinalResults sends a displayResults message.
func TestBroadcastFinalResults(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	// Set up a MeetState with predefined JudgeDecisions.
	mockMeetState := getMeetState("APL Test Meet")
	mockMeetState.JudgeDecisions = map[string]string{
		"left":   "good",
		"center": "no lift",
		"right":  "good",
	}

	broadcastFinalResults("APL Test Meet")

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "displayResults", decoded["action"])
		assert.Equal(t, "good", decoded["leftDecision"])
	default:
		t.Fatal("Expected final results broadcast, but got none")
	}
}

// TestBroadcastFinalResults_ClearsAfterTimeout verifies that broadcastFinalResults
// sends a clearResults message after the timeout.
func TestBroadcastFinalResults_ClearsAfterTimeout(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	// Set a short display duration.
	resultsDisplayDuration = 1

	// Create a controlled MeetState.
	mockState := &MeetState{
		JudgeDecisions: map[string]string{
			"left":   "good",
			"center": "bad",
			"right":  "good",
		},
	}
	// Override getMeetStateFunc to return our controlled MeetState.
	origGetMeetState := getMeetStateFunc
	getMeetStateFunc = func(meetName string) *MeetState {
		return mockState
	}
	defer func() { getMeetStateFunc = origGetMeetState }()

	// Override sleepFunc to simulate an immediate timeout.
	origSleep := sleepFunc
	sleepFunc = func(d time.Duration) {}
	defer func() { sleepFunc = origSleep }()

	broadcastFinalResults("APL Test Meet")

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "displayResults", decoded["action"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected displayResults broadcast, but got none")
	}

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "clearResults", decoded["action"])
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected clearResults broadcast after simulated timeout, but got none")
	}
}

// TestBroadcastTimeUpdateWithIndex verifies that broadcastTimeUpdateWithIndex sends the correct message.
func TestBroadcastTimeUpdateWithIndex(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	broadcastTimeUpdateWithIndex("updateTime", 30, 1, "APL Test Meet")

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]interface{}
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "updateTime", decoded["action"])
		assert.Equal(t, float64(30), decoded["timeLeft"])
		assert.Equal(t, float64(1), decoded["index"])
	default:
		t.Fatal("Expected time update broadcast, but got none")
	}
}

// TestBroadcastAllNextAttemptTimers verifies that only active timers are broadcasted.
func TestBroadcastAllNextAttemptTimers(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	timers := []NextAttemptTimer{
		{Active: true, TimeLeft: 30},
		{Active: false, TimeLeft: 20}, // Should be skipped.
		{Active: true, TimeLeft: 10},
	}

	broadcastAllNextAttemptTimers(timers, "APL Test Meet")

	// Expect exactly 2 messages (one for each active timer).
	count := 0
	for i := 0; i < 2; i++ {
		select {
		case <-mockBroadcast:
			count++
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected active timers to be broadcasted, but timeout occurred")
		}
	}
	assert.Equal(t, 2, count, "Expected exactly 2 active timers to be broadcasted")
}

// TestSendBroadcastMessage verifies that SendBroadcastMessage sends raw data.
func TestSendBroadcastMessage(t *testing.T) {
	InitTest()
	flushBroadcastChannel()

	rawData := []byte(`{"action":"rawMessage"}`)
	SendBroadcastMessage(rawData)

	select {
	case msg := <-mockBroadcast:
		assert.Equal(t, rawData, msg)
	default:
		t.Fatal("Expected raw message in broadcast channel, but got none")
	}
}

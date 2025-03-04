// file: websocket/broadcast_test.go
package websocket

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ✅ Mock WebSocket Broadcast Channel
var mockBroadcast = make(chan []byte, 10) // Buffered channel to prevent blocking

// ✅ Override `broadcast` with mock
func init() {
	broadcast = mockBroadcast
}

// ✅ Test helper function to override `getMeetState` during tests
var mockGetMeetState = getMeetState // Default to real function

// ✅ Wrapper function to use mock in tests
func getMeetStateWrapper(meetName string) *MeetState {
	return mockGetMeetState(meetName)
}

// ✅ Helper function to **flush** the mock broadcast channel before each test
func flushBroadcastChannel() {
	for len(mockBroadcast) > 0 {
		<-mockBroadcast
	}
}

// ✅ Test: BroadcastMessage marshals and sends messages
func TestBroadcastMessage_Success(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

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

// ✅ Test: broadcastFinalResults sends final decisions
func TestBroadcastFinalResults(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

	// ✅ Force override the meet state before calling broadcastFinalResults
	mockMeetState := getMeetState("APL Test Meet")
	mockMeetState.JudgeDecisions = map[string]string{
		"left":   "good",
		"center": "no lift",
		"right":  "good",
	}

	// ✅ Call broadcastFinalResults now that meetState is modified
	broadcastFinalResults("APL Test Meet")

	// ✅ Validate message was broadcasted
	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "displayResults", decoded["action"])
		assert.Equal(t, "good", decoded["leftDecision"]) // ✅ Now this should pass
	default:
		t.Fatal("Expected final results broadcast, but got none")
	}
}

// ✅ Test: broadcastFinalResults sends **clearResults** after timeout
func TestBroadcastFinalResults_ClearsAfterTimeout(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

	resultsDisplayDuration = 1 // ✅ Set short timeout for test

	mockState := &MeetState{
		JudgeDecisions: map[string]string{
			"left":   "good",
			"center": "bad",
			"right":  "good",
		},
	}
	mockGetMeetState = func(meetName string) *MeetState {
		return mockState
	}

	// ✅ Call broadcastFinalResults
	broadcastFinalResults("APL Test Meet")

	// ✅ Read **both** messages from the broadcast channel
	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "displayResults", decoded["action"]) // ✅ First message

	case <-time.After(1 * time.Second):
		t.Fatal("Expected displayResults broadcast, but got none")
	}

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]string
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "clearResults", decoded["action"]) // ✅ Second message (after timeout)

	case <-time.After(2 * time.Second):
		t.Fatal("Expected clearResults broadcast after timeout, but got none")
	}
}

// ✅ Test: broadcastTimeUpdateWithIndex sends time updates
func TestBroadcastTimeUpdateWithIndex(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

	broadcastTimeUpdateWithIndex("updateTime", 30, 1, "APL Test Meet")

	select {
	case msg := <-mockBroadcast:
		var decoded map[string]interface{}
		err := json.Unmarshal(msg, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "updateTime", decoded["action"]) // ✅ Correct message
		assert.Equal(t, float64(30), decoded["timeLeft"])
		assert.Equal(t, float64(1), decoded["index"])
	default:
		t.Fatal("Expected time update broadcast, but got none")
	}
}

// ✅ Test: broadcastAllNextAttemptTimers sends all active timers
func TestBroadcastAllNextAttemptTimers(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

	timers := []NextAttemptTimer{
		{Active: true, TimeLeft: 30},
		{Active: false, TimeLeft: 20}, // ❌ This should be skipped
		{Active: true, TimeLeft: 10},
	}

	broadcastAllNextAttemptTimers(timers, "APL Test Meet")

	// ✅ Expect only active timers to be broadcasted (2 messages)
	count := 0
	for i := 0; i < 2; i++ {
		select {
		case <-mockBroadcast:
			count++
		case <-time.After(1 * time.Second):
			t.Fatal("Expected active timers to be broadcasted, but timeout occurred")
		}
	}
	assert.Equal(t, 2, count, "Expected exactly 2 active timers to be broadcasted")
}

// ✅ Test: SendBroadcastMessage sends raw data
func TestSendBroadcastMessage(t *testing.T) {
	flushBroadcastChannel() // ✅ Ensure clean state before test

	rawData := []byte(`{"action":"rawMessage"}`)
	SendBroadcastMessage(rawData)

	select {
	case msg := <-mockBroadcast:
		assert.Equal(t, rawData, msg)
	default:
		t.Fatal("Expected raw message in broadcast channel, but got none")
	}
}

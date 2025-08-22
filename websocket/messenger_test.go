//go:build unit
// +build unit

// file: websocket/messenger_test.go
package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestRealMessenger_BroadcastMessage tests the BroadcastMessage method of the realMessenger.
func TestRealMessenger_BroadcastMessage(t *testing.T) {
	// set up a fake broadcast collector.
	var captured []byte
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	// override broadcast with a buffered channel.
	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	testMsg := map[string]interface{}{"action": "testAction"}
	rm.BroadcastMessage("TestMeet", testMsg)

	// read from the channel.
	captured = <-broadcast
	var result map[string]interface{}
	err := json.Unmarshal(captured, &result)
	assert.NoError(t, err)
	assert.Equal(t, "testAction", result["action"])
	assert.Equal(t, "TestMeet", result["meetName"])
}

// TestRealMessenger_BroadcastTimeUpdate tests the BroadcastTimeUpdate method of the realMessenger.
func TestRealMessenger_BroadcastTimeUpdate(t *testing.T) {
	// set up a fake broadcast collector.
	var captured []byte
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	action := "updateTime"
	timeLeft := 42
	index := 3
	meetName := "TestMeet"

	rm.BroadcastTimeUpdate(action, timeLeft, index, meetName)

	// read from the channel.
	captured = <-broadcast
	var result map[string]interface{}
	err := json.Unmarshal(captured, &result)
	assert.NoError(t, err)
	// JSON numbers become float64 by default.
	assert.Equal(t, action, result["action"])
	assert.Equal(t, float64(timeLeft), result["timeLeft"])
	assert.Equal(t, float64(index), result["index"])
	assert.Equal(t, meetName, result["meetName"])
}

// TestRealMessenger_BroadcastRaw tests the BroadcastRaw method of the realMessenger
func TestRealMessenger_BroadcastRaw(t *testing.T) {
	// set up a dummy broadcast collector.
	var captured []byte
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	rawMsg := []byte(`{"action":"rawTest"}`)
	rm.BroadcastRaw(rawMsg)

	// read from the channel.
	captured = <-broadcast
	assert.Equal(t, rawMsg, captured)
}

// TestRealMessenger_BroadcastMessage_EmptyMeetName tests that BroadcastMessage rejects empty meetName
func TestRealMessenger_BroadcastMessage_EmptyMeetName(t *testing.T) {
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	// override broadcast with a buffered channel.
	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	testMsg := map[string]interface{}{"action": "testAction"}
	rm.BroadcastMessage("", testMsg)

	// Should not send any message when meetName is empty
	select {
	case <-broadcast:
		t.Fatal("Expected no message in broadcast channel when meetName is empty, but got one")
	default:
		// This is expected - no message should be sent
	}
}

// TestRealMessenger_BroadcastTimeUpdate_EmptyMeetName tests that BroadcastTimeUpdate rejects empty meetName
func TestRealMessenger_BroadcastTimeUpdate_EmptyMeetName(t *testing.T) {
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	rm.BroadcastTimeUpdate("updateTime", 42, 3, "")

	// Should not send any message when meetName is empty
	select {
	case <-broadcast:
		t.Fatal("Expected no message in broadcast channel when meetName is empty, but got one")
	default:
		// This is expected - no message should be sent
	}
}

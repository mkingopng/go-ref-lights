// file: websocket/messenger_test.go
//go:build unit
// +build unit

package websocket

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRealMessenger_BroadcastMessage(t *testing.T) {
	// Set up a dummy broadcast collector.
	var captured []byte
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	// Override broadcast with a buffered channel.
	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	testMsg := map[string]interface{}{"action": "testAction"}
	rm.BroadcastMessage("TestMeet", testMsg)

	// Read from the channel.
	captured = <-broadcast
	var result map[string]interface{}
	err := json.Unmarshal(captured, &result)
	assert.NoError(t, err)
	assert.Equal(t, "testAction", result["action"])
}

func TestRealMessenger_BroadcastTimeUpdate(t *testing.T) {
	// Set up a dummy broadcast collector.
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

	// Read from the channel.
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

func TestRealMessenger_BroadcastRaw(t *testing.T) {
	// Set up a dummy broadcast collector.
	var captured []byte
	originalBroadcast := broadcast
	defer func() { broadcast = originalBroadcast }()

	broadcast = make(chan []byte, 1)

	rm := &realMessenger{}
	rawMsg := []byte(`{"action":"rawTest"}`)
	rm.BroadcastRaw(rawMsg)

	// Read from the channel.
	captured = <-broadcast
	assert.Equal(t, rawMsg, captured)
}

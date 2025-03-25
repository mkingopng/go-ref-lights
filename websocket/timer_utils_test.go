//go:build unit
// +build unit

// file: websocket/timer_utils_test.go
//
package websocket

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindTimerIndex(t *testing.T) {
	timers := []NextAttemptTimer{
		{ID: 1, TimeLeft: 30, Active: true},
		{ID: 2, TimeLeft: 20, Active: true},
	}
	index := findTimerIndex(timers, 2)
	assert.Equal(t, 1, index)

	index = findTimerIndex(timers, 3)
	assert.Equal(t, -1, index)
}

// TestBroadcastAllNextAttemptTimers tests the broadcastAllNextAttemptTimers function.
func TestBroadcastAllNextAttemptTimers(t *testing.T) {
	// override broadcastToMeet to capture output
	var captured []byte
	originalFunc := broadcastToMeet
	defer func() { broadcastToMeet = originalFunc }()

	broadcastToMeet = func(meetName string, msg []byte) {
		captured = msg
	}

	timers := []NextAttemptTimer{
		{ID: 1, TimeLeft: 30, Active: true},
	}
	broadcastAllNextAttemptTimers(timers, "TestMeet")

	var result map[string]interface{}
	err := json.Unmarshal(captured, &result)
	assert.NoError(t, err)
	assert.Equal(t, "updateNextAttemptTime", result["action"])
	assert.Equal(t, "TestMeet", result["meetName"])
}

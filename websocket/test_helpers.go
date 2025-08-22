// Package websocket test_helpers.go
// File: websocket/test_helpers.go
package websocket

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// InitTest sets up the test environment for WebSocket-based meet state handling.
func InitTest() {
	// flush the broadcast channel if necessary.
	for len(broadcast) > 0 {
		<-broadcast
	}

	// reset the results display duration if needed.
	resultsDisplayDuration = 15

	// reset the sleep function to the standard one.
	sleepFunc = time.Sleep

	// reset the timer manager if it exists.
	if defaultTimerManager != nil {
		defaultTimerManager.nextAttemptIDCounter = 0
	}
}

// MockStateProvider is a fake implementation of StateProvider for testing.
type MockStateProvider struct {
	mock.Mock
	meetStates map[string]*MeetState
}

// NewMockStateProvider creates a new MockStateProvider with empty or pre-seeded MeetStates.
func NewMockStateProvider() *MockStateProvider {
	return &MockStateProvider{
		meetStates: make(map[string]*MeetState),
	}
}

func (msp *MockStateProvider) GetMeetState(meetName string) *MeetState {
	args := msp.Called(meetName)
	return args.Get(0).(*MeetState)
}

// MockMessenger is a fake implementation of Messenger for testing.
type MockMessenger struct {
	mock.Mock
	Broadcasts []string
}

// BroadcastMessage simulates sending a message to clients, storing the message in Broadcasts.
func (mm *MockMessenger) BroadcastMessage(meetName string, msg map[string]interface{}) {
	// let testify track that BroadcastMessage was called with these arguments
	mm.Called(meetName, msg)

	// also do your own optional logic
	mm.Broadcasts = append(mm.Broadcasts, "[BroadcastMessage] meet="+meetName)
}

// BroadcastTimeUpdate simulates sending a time update, also storing the data.
func (mm *MockMessenger) BroadcastTimeUpdate(action string, timeLeft int, index int, meetName string) {
	// let testify track that BroadcastTimeUpdate was called with these arguments
	mm.Called(action, timeLeft, index, meetName)

	// also do your own optional logic
	mm.Broadcasts = append(mm.Broadcasts,
		"[BroadcastTimeUpdate] action="+action+
			" timeLeft="+string(rune(timeLeft))+
			" index="+string(rune(index))+
			" meet="+meetName)
}

// BroadcastRaw simulates sending raw bytes, storing them as a string in Broadcasts.
func (mm *MockMessenger) BroadcastRaw(msg []byte) {
	// let testify track that BroadcastRaw was called with these arguments
	mm.Called(msg)

	// also do your own optional logic
	mm.Broadcasts = append(mm.Broadcasts, "[BroadcastRaw] "+string(msg))
}

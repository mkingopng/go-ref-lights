// websocket/timer_test.go

//go:build unit
// +build unit

package websocket

import (
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

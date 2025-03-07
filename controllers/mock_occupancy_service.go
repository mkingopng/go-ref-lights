package controllers

import (
	"github.com/stretchr/testify/mock"
	"go-ref-lights/services"
)

// MockOccupancyService implements the OccupancyServiceInterface for testing.
type MockOccupancyService struct {
	mock.Mock
}

// UnsetPosition removes the position assignment for a given referee.
func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// GetOccupancy retrieves the current occupancy status for a given meet.
func (m *MockOccupancyService) GetOccupancy(meetName string) services.Occupancy {
	args := m.Called(meetName)
	return args.Get(0).(services.Occupancy)
}

// ResetOccupancyForMeet clears all referee positions for a specific meet.
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

// SetPosition assigns a referee to a specific position in a meet.
func (m *MockOccupancyService) SetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

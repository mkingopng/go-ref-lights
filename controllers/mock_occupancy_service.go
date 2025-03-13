// Package controllers provides mock implementations for testing.
// File: controllers/mock_occupancy_service.go
package controllers

import (
	"github.com/stretchr/testify/mock"
	"go-ref-lights/services"
)

// -------------------- mock service for testing --------------------

// MockOccupancyService is a mock implementation of the OccupancyServiceInterface.
// It is used in unit tests to simulate interactions with the occupancy service.
type MockOccupancyService struct {
	mock.Mock
}

// ---------------------- mock method implementations ----------------------

// UnsetPosition removes the position assignment for a given referee.
// This function simulates the behavior of unsetting a referee’s assigned position in a meet.
func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// GetOccupancy retrieves the current occupancy status for a given meet.
// This function returns a mock response based on predefined test cases.
func (m *MockOccupancyService) GetOccupancy(meetName string) services.Occupancy {
	args := m.Called(meetName)
	return args.Get(0).(services.Occupancy)
}

// ResetOccupancyForMeet clears all referee positions for a specific meet.
// This function ensures that all referee positions are reset to vacant.
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

// SetPosition assigns a referee to a specific position in a meet.
// This function mimics the process of assigning a referee to a seat.
func (m *MockOccupancyService) SetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

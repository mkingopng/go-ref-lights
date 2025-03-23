package services

import (
	"github.com/stretchr/testify/mock"
)

// Ensure MockOccupancyService implements OccupancyServiceInterface
var _ OccupancyServiceInterface = (*MockOccupancyService)(nil)

// MockOccupancyService is a mock implementation for testing and extends `mock.Mock`
type MockOccupancyService struct {
	mock.Mock
}

// GetOccupancy is a mocked function that returns a mock Occupancy struct
func (m *MockOccupancyService) GetOccupancy(meetName string) Occupancy {
	args := m.Called(meetName)
	return args.Get(0).(Occupancy)
}

// SetPosition is a mocked function that returns an error
func (m *MockOccupancyService) SetPosition(meetName, position, userEmail string) error {
	args := m.Called(meetName, position, userEmail)
	return args.Error(0)
}

// UnsetPosition removes a user's position from the occupancy service (mocked)
func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// ResetOccupancyForMeet is a mocked function that resets the occupancy for a given meet
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

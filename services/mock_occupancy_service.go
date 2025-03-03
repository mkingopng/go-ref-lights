package services

import (
	"fmt"
	"github.com/stretchr/testify/mock"
)

// ✅ Ensure MockOccupancyService implements OccupancyServiceInterface
var _ OccupancyServiceInterface = (*MockOccupancyService)(nil)

// MockOccupancyService is a mock implementation for testing and extends `mock.Mock`
type MockOccupancyService struct {
	mock.Mock
}

// GetOccupancy (Mocked)
func (m *MockOccupancyService) GetOccupancy(meetName string) Occupancy {
	fmt.Println("Mock GetOccupancy Called with:", meetName) // 🛠 Debugging Output
	args := m.Called(meetName)
	return args.Get(0).(Occupancy)
}

// SetPosition (Mocked)
func (m *MockOccupancyService) SetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// UnsetPosition (Mocked)
func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

// ResetOccupancyForMeet (Mocked)
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

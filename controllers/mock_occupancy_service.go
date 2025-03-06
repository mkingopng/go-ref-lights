package controllers

import (
	"github.com/stretchr/testify/mock"
	"go-ref-lights/services"
)

// MockOccupancyService implements the OccupancyServiceInterface for testing.
type MockOccupancyService struct {
	mock.Mock
}

func (m *MockOccupancyService) UnsetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

func (m *MockOccupancyService) GetOccupancy(meetName string) services.Occupancy {
	args := m.Called(meetName)
	return args.Get(0).(services.Occupancy)
}

func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

func (m *MockOccupancyService) SetPosition(meetName, position, user string) error {
	args := m.Called(meetName, position, user)
	return args.Error(0)
}

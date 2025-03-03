package services

// ✅ Ensure MockOccupancyService implements OccupancyServiceInterface
var _ OccupancyServiceInterface = (*MockOccupancyService)(nil)

// MockOccupancyService is a mock implementation for testing
type MockOccupancyService struct{}

// GetOccupancy returns the current occupancy state for a meet
func (m *MockOccupancyService) GetOccupancy(meetName string) Occupancy {
	return Occupancy{
		LeftUser:   "user1",
		CenterUser: "",
		RightUser:  "user2",
	}
}

// SetPosition assigns a user to a position
func (m *MockOccupancyService) SetPosition(meetName, position, userEmail string) error {
	return nil
}

// UnsetPosition removes a user from a position
func (m *MockOccupancyService) UnsetPosition(meetName, position, userEmail string) error {
	return nil
}

// ResetOccupancyForMeet clears all positions for a meet
func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	// No-op (mock function does nothing)
}

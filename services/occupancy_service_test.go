//go:build unit
// +build unit

// file: services/occupancy_service_test.go
package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-ref-lights/websocket"
)

// resetGlobalOccupancy clears the global occupancy map (used between tests)
func resetGlobalOccupancy() {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	for k := range occupancyMap {
		delete(occupancyMap, k)
	}
}

// TestGetOccupancy_NewMeet ensures that a new meet has no users assigned to any positions
func TestGetOccupancy_NewMeet(t *testing.T) {
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL State Championship"

	// expect an empty occupancy state for a new meet
	occupancy := service.GetOccupancy(meetName)

	assert.Empty(t, occupancy.LeftUser)
	assert.Empty(t, occupancy.CenterUser)
	assert.Empty(t, occupancy.RightUser)
}

// TestSetPosition_Success ensures that a user can take a position that is not already occupied
func TestSetPosition_Success(t *testing.T) {
	resetGlobalOccupancy()
	service := NewOccupancyService()

	// call the existing signature
	err := service.SetPosition("APL Nationals", "left", "referee1@example.com")
	require.NoError(t, err)

	occupancy := service.GetOccupancy("APL Nationals")
	assert.Equal(t, "referee1@example.com", occupancy.LeftUser)
}

// TestSetPosition_FailsIfTaken ensures that a user cannot take a position that is already occupied
func TestSetPosition_FailsIfTaken(t *testing.T) {
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Regionals"

	// first referee takes the left position
	_ = service.SetPosition(meetName, "left", "ref1@example.com")

	// second referee should be blocked from taking the same position
	err := service.SetPosition(meetName, "left", "ref2@example.com")
	assert.Error(t, err)
	assert.Equal(t, "left position is already taken", err.Error())

	// ensure the original assignment is unchanged
	occupancy := service.GetOccupancy(meetName)
	assert.Equal(t, "ref1@example.com", occupancy.LeftUser)
}

// TestSetPosition_ClearsOldSeatBeforeAssigningNewOne ensures that a user can only hold one position at a time
func TestSetPosition_ClearsOldSeatBeforeAssigningNewOne(t *testing.T) {
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Qualifiers"

	// assign user to left
	_ = service.SetPosition(meetName, "left", "ref1@example.com")

	// move the same user to center
	err := service.SetPosition(meetName, "center", "ref1@example.com")
	assert.NoError(t, err)

	// verify they moved
	occupancy := service.GetOccupancy(meetName)
	assert.Empty(t, occupancy.LeftUser) // Old position should be empty
	assert.Equal(t, "ref1@example.com", occupancy.CenterUser)
}

// TestResetOccupancyForMeet ensures that all positions are cleared when a meet is reset
func TestResetOccupancyForMeet(t *testing.T) {
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Open"

	// assign positions
	_ = service.SetPosition(meetName, "left", "ref1@example.com")
	_ = service.SetPosition(meetName, "center", "ref2@example.com")

	// reset occupancy
	service.ResetOccupancyForMeet(meetName)

	// expect an empty occupancy state
	occupancy := service.GetOccupancy(meetName)
	assert.Empty(t, occupancy.LeftUser)
	assert.Empty(t, occupancy.CenterUser)
	assert.Empty(t, occupancy.RightUser)
}

func TestUnsetPosition(t *testing.T) {
	resetGlobalOccupancy()
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Grand Finals"

	// assign a user to right
	_ = service.SetPosition(meetName, "right", "ref3@example.com")

	// unset the position
	err := service.UnsetPosition(meetName, "right", "ref3@example.com")
	assert.NoError(t, err)

	// verify position is cleared
	occupancy := service.GetOccupancy(meetName)
	assert.Empty(t, occupancy.RightUser)
}

// TestUnsetPosition_FailsIfPositionDoesNotMatchUser ensures that a user cannot unset a position that they do not hold
func TestUnsetPosition_FailsIfPositionDoesNotMatchUser(t *testing.T) {
	resetGlobalOccupancy()
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Regionals"

	// assign a user to a position
	_ = service.SetPosition(meetName, "center", "ref2@example.com")

	// attempt to unset the position with a different user (should fail)
	err := service.UnsetPosition(meetName, "center", "wronguser@example.com")

	// expect an error
	assert.Error(t, err, "Expected an error when an incorrect user tries to unset a position")
	assert.Equal(t, "user does not hold this position", err.Error())

	// ensure the original assignment remains unchanged
	occupancy := service.GetOccupancy(meetName)
	assert.Equal(t, "ref2@example.com", occupancy.CenterUser)
}

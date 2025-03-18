// file: services/occupancy_service_test.go

//go:build unit
// +build unit

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

func TestGetOccupancy_NewMeet(t *testing.T) {
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL State Championship"

	// expect an empty occupancy state for a new meet
	occupancy := service.GetOccupancy(meetName)

	assert.Empty(t, occupancy.LeftUser)
	assert.Empty(t, occupancy.CenterUser)
	assert.Empty(t, occupancy.RightUser)
}

func TestSetPosition_Success(t *testing.T) {
	websocket.InitTest()
	service := &OccupancyService{}
	meetName := "APL Nationals"

	// assign a user to the left position
	err := service.SetPosition(meetName, "left", "referee1@example.com")
	assert.NoError(t, err)

	// verify that the position is correctly assigned
	occupancy := service.GetOccupancy(meetName)
	assert.Equal(t, "referee1@example.com", occupancy.LeftUser)
}

func TestSetPosition_FailsIfTaken(t *testing.T) {
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

func TestSetPosition_ClearsOldSeatBeforeAssigningNewOne(t *testing.T) {
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

func TestResetOccupancyForMeet(t *testing.T) {
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

func TestUnsetPosition_FailsIfPositionDoesNotMatchUser(t *testing.T) {
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

// file: services/occupancy_service_test.go
package services_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

// Test concurrent users claiming the same position
func TestSetPosition_ConcurrentUsers(t *testing.T) {
	svc := services.OccupancyService{}
	svc.ResetOccupancy()

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error

	go func() {
		defer wg.Done()
		err1 = svc.SetPosition("left", "user1@example.com")
	}()

	go func() {
		defer wg.Done()
		err2 = svc.SetPosition("left", "user2@example.com")
	}()

	wg.Wait()

	// ✅ Ensure exactly **one** user succeeded
	if err1 == nil {
		assert.Error(t, err2, "Second user should fail")
	} else {
		assert.NoError(t, err2, "One user must succeed")
	}

	// ✅ Ensure only **one** user holds the "left" position
	occ := svc.GetOccupancy()
	if occ.LeftUser != "user1@example.com" && occ.LeftUser != "user2@example.com" {
		t.Fatalf("Unexpected user assigned to 'left': got %s, expected 'user1@example.com' or 'user2@example.com'", occ.LeftUser)
	}
}

// Test setting an invalid position
func TestSetPosition_InvalidPosition(t *testing.T) {
	svc := services.OccupancyService{}
	svc.ResetOccupancy()

	err := svc.SetPosition("invalid_position", "test@example.com")
	assert.Error(t, err, "Should return an error for an invalid position")
}

// Test switching positions mid-meet
func TestSetPosition_SwitchPosition(t *testing.T) {
	svc := services.OccupancyService{}
	svc.ResetOccupancy()

	// ✅ User claims "centre"
	err := svc.SetPosition("centre", "referee@example.com")
	assert.NoError(t, err)

	// ✅ Same user moves to "left"
	err = svc.SetPosition("left", "referee@example.com")
	assert.NoError(t, err)

	// ✅ Ensure "centre" is now empty and "left" is assigned
	occ := svc.GetOccupancy()
	assert.Equal(t, "referee@example.com", occ.LeftUser)
	assert.Empty(t, occ.CentreUser, "Previous position should be cleared")
}

// Test ResetOccupancy clears all assignments
func TestResetOccupancy(t *testing.T) {
	svc := services.OccupancyService{}
	svc.ResetOccupancy()

	// Assign all positions
	err := svc.SetPosition("left", "ref1@example.com")
	assert.NoError(t, err, "Setting position 'left' should not fail")

	err = svc.SetPosition("centre", "ref2@example.com")
	assert.NoError(t, err, "Setting position 'centre' should not fail")

	err = svc.SetPosition("right", "ref3@example.com")
	assert.NoError(t, err, "Setting position 'right' should not fail")

	// Reset everything
	svc.ResetOccupancy()

	// ✅ Ensure all positions are empty
	occ := svc.GetOccupancy()
	assert.Empty(t, occ.LeftUser, "Left position should be cleared after reset")
	assert.Empty(t, occ.CentreUser, "Centre position should be cleared after reset")
	assert.Empty(t, occ.RightUser, "Right position should be cleared after reset")
}

func TestSetPosition(t *testing.T) {
	// Create a new instance of OccupancyService
	svc := services.OccupancyService{}
	svc.ResetOccupancy() // ✅ Reset before running the test

	// Start with a clean occupancy
	occ := svc.GetOccupancy()
	assert.Empty(t, occ.LeftUser, "LeftUser should be empty at start")
	assert.Empty(t, occ.CentreUser, "CentreUser should be empty at start")
	assert.Empty(t, occ.RightUser, "RightUser should be empty at start")

	// Attempt to set "left" to "test@example.com"
	err := svc.SetPosition("left", "test@example.com")
	assert.NoError(t, err)

	occ = svc.GetOccupancy()
	assert.Equal(t, "test@example.com", occ.LeftUser)
	// Others should still be empty
	assert.Empty(t, occ.CentreUser)
	assert.Empty(t, occ.RightUser)
}

func TestSetPosition_AlreadyTaken(t *testing.T) {
	// Create a new instance of OccupancyService
	svc := services.OccupancyService{}
	svc.ResetOccupancy() // ✅ Reset before running the test

	// Set "centre" position to user1
	err := svc.SetPosition("centre", "user1@example.com")
	assert.NoError(t, err)

	// Now user2 tries to claim "centre" as well
	err = svc.SetPosition("centre", "user2@example.com")
	assert.Error(t, err, "Expected an error since 'centre' was already occupied")
}

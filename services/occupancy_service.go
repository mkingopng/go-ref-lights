// file: services/occupancy_service.go

package services

import (
	"errors"
	"sync"
)

// Occupancy defines the struct to track referee positions
type Occupancy struct {
	LeftUser   string
	CentreUser string
	RightUser  string
}

// Mutex for thread safety
var occupancyMutex sync.Mutex

// Global variable to track current referee occupancy
var currentOccupancy = Occupancy{}

// OccupancyServiceInterface defines an interface for dependency injection
type OccupancyServiceInterface interface {
	GetOccupancy() Occupancy
	SetPosition(position, userEmail string) error
	ResetOccupancy()
}

// OccupancyService struct that implements the interface
type OccupancyService struct{}

// GetOccupancy retrieves the current occupancy state (thread-safe)
func (s *OccupancyService) GetOccupancy() Occupancy {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	return currentOccupancy
}

// SetPosition assigns a user to a referee position (thread-safe)
func (s *OccupancyService) SetPosition(position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	// ✅ Reject invalid positions
	validPositions := map[string]bool{"left": true, "centre": true, "right": true}
	if !validPositions[position] {
		return errors.New("invalid position selected")
	}

	// ✅ Prevent multiple assignments of the same position
	switch position {
	case "left":
		if currentOccupancy.LeftUser != "" {
			return errors.New("left position is already taken")
		}
	case "centre":
		if currentOccupancy.CentreUser != "" {
			return errors.New("centre position is already taken")
		}
	case "right":
		if currentOccupancy.RightUser != "" {
			return errors.New("right position is already taken")
		}
	}

	// ✅ Clear any previous position assigned to this user
	if currentOccupancy.LeftUser == userEmail {
		currentOccupancy.LeftUser = ""
	}
	if currentOccupancy.CentreUser == userEmail {
		currentOccupancy.CentreUser = ""
	}
	if currentOccupancy.RightUser == userEmail {
		currentOccupancy.RightUser = ""
	}

	// ✅ Assign the new position
	switch position {
	case "left":
		currentOccupancy.LeftUser = userEmail
	case "centre":
		currentOccupancy.CentreUser = userEmail
	case "right":
		currentOccupancy.RightUser = userEmail
	}

	return nil
}

// ResetOccupancy resets the global occupancy state
func (s *OccupancyService) ResetOccupancy() {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	currentOccupancy = Occupancy{}
}

// UnsetPosition frees up a referee position
func UnsetPosition(position, userEmail string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	switch position {
	case "left":
		if currentOccupancy.LeftUser == userEmail {
			currentOccupancy.LeftUser = ""
		}
	case "centre":
		if currentOccupancy.CentreUser == userEmail {
			currentOccupancy.CentreUser = ""
		}
	case "right":
		if currentOccupancy.RightUser == userEmail {
			currentOccupancy.RightUser = ""
		}
	}
}

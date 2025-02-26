// Package services file: services/occupancy_service.go
package services

import (
	"errors"
	"sync"

	"go-ref-lights/logger"
)

// Occupancy defines the struct to track referee positions.
type Occupancy struct {
	LeftUser   string
	CentreUser string
	RightUser  string
}

// Mutex for thread safety.
var occupancyMutex sync.Mutex

// Global variable to track current referee occupancy.
var currentOccupancy = Occupancy{}

// OccupancyServiceInterface defines an interface for dependency injection.
type OccupancyServiceInterface interface {
	GetOccupancy() Occupancy
	SetPosition(position, userEmail string) error
	ResetOccupancy()
}

// OccupancyService struct that implements the interface.
type OccupancyService struct{}

// GetOccupancy retrieves the current occupancy state (thread-safe).
func (s *OccupancyService) GetOccupancy() Occupancy {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	logger.Debug.Printf("Getting occupancy state: %+v", currentOccupancy)
	return currentOccupancy
}

// SetPosition assigns a user to a referee position (thread-safe).
func (s *OccupancyService) SetPosition(position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	logger.Info.Printf("Attempting to assign position '%s' to user '%s'", position, userEmail)

	// Reject invalid positions.
	validPositions := map[string]bool{"left": true, "centre": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected")
		logger.Error.Printf("SetPosition failed: %v", err)
		return err
	}

	// Prevent multiple assignments of the same position.
	switch position {
	case "left":
		if currentOccupancy.LeftUser != "" {
			err := errors.New("left position is already taken")
			logger.Error.Printf("SetPosition failed: %v", err)
			return err
		}
	case "centre":
		if currentOccupancy.CentreUser != "" {
			err := errors.New("centre position is already taken")
			logger.Error.Printf("SetPosition failed: %v", err)
			return err
		}
	case "right":
		if currentOccupancy.RightUser != "" {
			err := errors.New("right position is already taken")
			logger.Error.Printf("SetPosition failed: %v", err)
			return err
		}
	}

	// Clear any previous position assigned to this user.
	if currentOccupancy.LeftUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to left", userEmail)
		currentOccupancy.LeftUser = ""
	}
	if currentOccupancy.CentreUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to centre", userEmail)
		currentOccupancy.CentreUser = ""
	}
	if currentOccupancy.RightUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to right", userEmail)
		currentOccupancy.RightUser = ""
	}

	// Assign the new position.
	switch position {
	case "left":
		currentOccupancy.LeftUser = userEmail
	case "centre":
		currentOccupancy.CentreUser = userEmail
	case "right":
		currentOccupancy.RightUser = userEmail
	}

	logger.Info.Printf("Position '%s' successfully assigned to user '%s'. Current occupancy: %+v", position, userEmail, currentOccupancy)
	return nil
}

// ResetOccupancy resets the global occupancy state.
func (s *OccupancyService) ResetOccupancy() {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	currentOccupancy = Occupancy{}
	logger.Info.Println("Occupancy state has been reset.")
}

// UnsetPosition frees up a referee position.
func UnsetPosition(position, userEmail string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	switch position {
	case "left":
		if currentOccupancy.LeftUser == userEmail {
			logger.Info.Printf("Unsetting left position for user '%s'", userEmail)
			currentOccupancy.LeftUser = ""
		}
	case "centre":
		if currentOccupancy.CentreUser == userEmail {
			logger.Info.Printf("Unsetting centre position for user '%s'", userEmail)
			currentOccupancy.CentreUser = ""
		}
	case "right":
		if currentOccupancy.RightUser == userEmail {
			logger.Info.Printf("Unsetting right position for user '%s'", userEmail)
			currentOccupancy.RightUser = ""
		}
	}
}

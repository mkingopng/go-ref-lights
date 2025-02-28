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
	CenterUser string
	RightUser  string
}

// Mutex for thread safety.
var occupancyMutex sync.Mutex
var occupancyMap = make(map[string]*Occupancy)

// OccupancyServiceInterface defines an interface for dependency injection.
type OccupancyServiceInterface interface {
	GetOccupancy(meetName string) Occupancy
	SetPosition(meetName, position, userEmail string) error
	ResetOccupancyForMeet(meetName string)
}

// OccupancyService struct that implements the interface.
type OccupancyService struct{}

// GetOccupancy retrieves the current occupancy state (thread-safe).
func (s *OccupancyService) GetOccupancy(meetName string) Occupancy {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetName]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}
	logger.Debug.Printf("Getting occupancy state: %s %+v", meetName, occ)
	return *occ
}

// SetPosition assigns a user to a referee position (thread-safe).
func (s *OccupancyService) SetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetName]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}
	logger.Info.Printf("Attempting to assign position '%s' to user '%s' for meet %s", position, userEmail, meetName)

	// validate position and check if already taken
	validPositions := map[string]bool{"left": true, "center": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected")
		logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
		return err
	}

	// prevent multiple assignments of the same position.
	switch position {
	case "left":
		if occ.LeftUser != "" {
			err := errors.New("left position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	case "center":
		if occ.CenterUser != "" {
			err := errors.New("center position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	case "right":
		if occ.RightUser != "" {
			err := errors.New("right position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	}

	// clear any previous position assigned to this user
	if occ.LeftUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to left in meet %s", userEmail, meetName)
		occ.LeftUser = ""
	}
	if occ.CenterUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to center in meet %s", userEmail, meetName)
		occ.CenterUser = ""
	}
	if occ.RightUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to right in meet %s", userEmail, meetName)
		occ.RightUser = ""
	}

	// assign new position.
	switch position {
	case "left":
		occ.LeftUser = userEmail
	case "center":
		occ.CenterUser = userEmail
	case "right":
		occ.RightUser = userEmail
	}

	logger.Info.Printf("Position '%s' assigned to user '%s' for meet %s. Current occupancy: %+v", position, userEmail, meetName, occ)
	return nil
}

// ResetOccupancyForMeet ResetOccupancy resets the global occupancy state.
func (s *OccupancyService) ResetOccupancyForMeet(meetName string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occupancyMap[meetName] = &Occupancy{}
	logger.Info.Printf("Occupancy state for meet %s has been reset.", meetName)
}

// UnsetPosition frees up a referee position for a specific meet
func UnsetPosition(meetName, position, userEmail string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetName]
	if !exists {
		return
	}
	switch position {
	case "left":
		if occ.LeftUser == userEmail {
			logger.Info.Printf("Unsetting left position for user '%s' in meet %s", userEmail, meetName)
			occ.LeftUser = ""
		}
	case "center":
		if occ.CenterUser == userEmail {
			logger.Info.Printf("Unsetting center position for user '%s' in meet %s", userEmail, meetName)
			occ.CenterUser = ""
		}
	case "right":
		if occ.RightUser == userEmail {
			logger.Info.Printf("Unsetting right position for user '%s' in meet %s", userEmail, meetName)
			occ.RightUser = ""
		}
	}
}

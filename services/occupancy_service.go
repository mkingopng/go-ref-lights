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
var occupancyMap = make(map[string]*Occupancy)

// OccupancyServiceInterface defines an interface for dependency injection.
type OccupancyServiceInterface interface {
	GetOccupancy() Occupancy
	SetPosition(meetId, position, userEmail string) error
	ResetOccupancyForMeet(meetId string)
}

// OccupancyService struct that implements the interface.
type OccupancyService struct{}

// GetOccupancy retrieves the current occupancy state (thread-safe).
func (s *OccupancyService) GetOccupancy(meetId string) Occupancy {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetId]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetId] = occ
	}
	logger.Debug.Printf("Getting occupancy state: %s %+v", meetId, occ)
	return *occ
}

// SetPosition assigns a user to a referee position (thread-safe).
func (s *OccupancyService) SetPosition(meetId, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetId]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetId] = occ
	}
	logger.Info.Printf("Attempting to assign position '%s' to user '%s' for meet %s", position, userEmail, meetId)

	// validate position and check if already taken
	validPositions := map[string]bool{"left": true, "centre": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected")
		logger.Error.Printf("SetPosition failed for meet %s: %v", meetId, err)
		return err
	}

	// prevent multiple assignments of the same position.
	switch position {
	case "left":
		if occ.LeftUser != "" {
			err := errors.New("left position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetId, err)
			return err
		}
	case "centre":
		if occ.CentreUser != "" {
			err := errors.New("centre position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetId, err)
			return err
		}
	case "right":
		if occ.RightUser != "" {
			err := errors.New("right position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetId, err)
			return err
		}
	}

	// clear any previous position assigned to this user
	if occ.LeftUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to left in meet %s", userEmail, meetId)
		occ.LeftUser = ""
	}
	if occ.CentreUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to centre in meet %s", userEmail, meetId)
		occ.CentreUser = ""
	}
	if occ.RightUser == userEmail {
		logger.Debug.Printf("Clearing previous assignment: user '%s' was assigned to right in meet %s", userEmail, meetId)
		occ.RightUser = ""
	}

	// assign new position.
	switch position {
	case "left":
		occ.LeftUser = userEmail
	case "centre":
		occ.CentreUser = userEmail
	case "right":
		occ.RightUser = userEmail
	}

	logger.Info.Printf("Position '%s' assigned to user '%s' for meet %s. Current occupancy: %+v", position, userEmail, meetId, occ)
	return nil
}

// ResetOccupancy resets the global occupancy state.
func (s *OccupancyService) ResetOccupancyForMeet(meetId string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occupancyMap[meetId] = &Occupancy{}
	logger.Info.Println("Occupancy state for meet %s has been reset.", meetId)
}

// UnsetPosition frees up a referee position.
func UnsetPosition(meetId, position, userEmail string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetId]
	if !exists {
		return
	}
	switch position {
	case "left":
		if occ.LeftUser == userEmail {
			logger.Info.Printf("Unsetting left position for user '%s' in meet %s", userEmail, meetId)
			occ.LeftUser = ""
		}
	case "centre":
		if occ.CentreUser == userEmail {
			logger.Info.Printf("Unsetting centre position for user '%s' in meet %s", userEmail, meetId)
			occ.CentreUser = ""
		}
	case "right":
		if occ.RightUser == userEmail {
			logger.Info.Printf("Unsetting right position for user '%s' in meet %s", userEmail, meetId)
			occ.RightUser = ""
		}
	}
}

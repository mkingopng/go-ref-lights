// Package services handles the business logic of the application, including referee position tracking.
// File: services/occupancy_service.go
package services

import (
	"errors"
	"sync"

	"go-ref-lights/logger"
)

// ---------------- global occupancy state ---------------------------

// Global mutex to protect access to occupancyMap.
var occupancyMutex sync.Mutex

// Global map to store the current occupancy state for each meet.
var occupancyMap = make(map[string]*Occupancy)

// ------------------ occupancy structures ----------------------------

// Occupancy defines the struct to track referee positions
type Occupancy struct {
	LeftUser   string // Referee assigned to the left position
	CenterUser string // Referee assigned to the center position
	RightUser  string // Referee assigned to the right position
}

// OccupancyServiceInterface defines the methods required for managing referee assignments.
type OccupancyServiceInterface interface {
	GetOccupancy(meetName string) Occupancy
	SetPosition(meetName, position, userEmail string) error
	ResetOccupancyForMeet(meetName string)
	UnsetPosition(meetName, position, userEmail string) error
}

// OccupancyService provides methods to manage referee assignments for meets.
type OccupancyService struct {
	mu        sync.Mutex
	occupancy map[string]*Occupancy
}

// ------------------ service initialisation ----------------------------

// NewOccupancyService creates and returns a new OccupancyService instance.
func NewOccupancyService() *OccupancyService {
	return &OccupancyService{
		occupancy: make(map[string]*Occupancy), // ✅ Initialize the map
	}
}

// ------------------ occupancy management ----------------------------

// GetOccupancy retrieves the current referee occupancy state for a given meet.
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

// SetPosition assigns a referee to a specific position in a meet.
func (s *OccupancyService) SetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	occ, exists := occupancyMap[meetName]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}
	logger.Info.Printf("Attempting to assign position '%s' to user '%s' for meet %s", position, userEmail, meetName)

	// validate position input
	validPositions := map[string]bool{"left": true, "center": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected, please choose left, center, or right")
		logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
		return err
	}

	// ensure the position isn't already taken
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

	// remove the user from any previous position
	if occ.LeftUser == userEmail {
		occ.LeftUser = ""
	}
	if occ.CenterUser == userEmail {
		occ.CenterUser = ""
	}
	if occ.RightUser == userEmail {
		occ.RightUser = ""
	}

	// assign the user to the new position
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

// UnsetPosition removes a referee from their assigned position in a meet.
func (s *OccupancyService) UnsetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	occ, exists := occupancyMap[meetName]
	if !exists {
		logger.Warn.Printf("UnsetPosition: no occupancy record for meet %s", meetName)
		return errors.New("no occupancy found for that meet")
	}

	switch position {
	case "left":
		if occ.LeftUser == userEmail {
			logger.Info.Printf("Unsetting left position for user '%s' in meet %s", userEmail, meetName)
			occ.LeftUser = ""
		} else {
			return errors.New("user does not hold this position")
		}

	case "center":
		if occ.CenterUser == userEmail {
			logger.Info.Printf("Unsetting center position for user '%s' in meet %s", userEmail, meetName)
			occ.CenterUser = ""
		} else {
			return errors.New("user does not hold this position")
		}

	case "right":
		if occ.RightUser == userEmail {
			logger.Info.Printf("Unsetting right position for user '%s' in meet %s", userEmail, meetName)
			occ.RightUser = ""
		} else {
			return errors.New("user does not hold this position")
		}

	default:
		err := errors.New("invalid position")
		logger.Error.Printf("UnsetPosition: %v", err)
		return err
	}

	logger.Info.Printf("Position '%s' was vacated by user '%s' for meet %s. Current occupancy: %+v", position, userEmail, meetName, occ)
	return nil
}

// ResetOccupancyForMeet clears all assigned referee positions for a given meet.
func (s *OccupancyService) ResetOccupancyForMeet(meetName string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	logger.Info.Printf("ResetOccupancyForMeet: Clearing all positions for meet '%s'", meetName)

	if occ, exists := occupancyMap[meetName]; exists {
		occ.LeftUser = ""
		occ.CenterUser = ""
		occ.RightUser = ""
	}
}

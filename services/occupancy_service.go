// Package services: services/occupancy_service.go
package services

import (
	"errors"
	"sync"

	"go-ref-lights/logger"
)

var occupancyMutex sync.Mutex
var occupancyMap = make(map[string]*Occupancy)

// Occupancy defines the struct to track referee positions
type Occupancy struct {
	LeftUser   string
	CenterUser string
	RightUser  string
}

type OccupancyServiceInterface interface {
	GetOccupancy(meetName string) Occupancy
	SetPosition(meetName, position, userEmail string) error
	ResetOccupancyForMeet(meetName string)
	// todo: ADD: We'll explicitly reference UnsetPosition in the interface too:
	UnsetPosition(meetName, position, userEmail string) error
}

type OccupancyService struct {
	mu        sync.Mutex
	occupancy map[string]*Occupancy
}

// NewOccupancyService creates a new OccupancyService instance.
func NewOccupancyService() *OccupancyService {
	return &OccupancyService{
		occupancy: make(map[string]*Occupancy), // ✅ Initialize the map
	}
}

// GetOccupancy returns the current occupancy state for a given meet
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

// SetPosition assigns a referee to a position for a given meet
func (s *OccupancyService) SetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	occ, exists := occupancyMap[meetName]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}
	logger.Info.Printf("Attempting to assign position '%s' to user '%s' for meet %s", position, userEmail, meetName)

	// validate position
	validPositions := map[string]bool{"left": true, "center": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected, please choose left, center, or right")
		logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
		return err
	}

	// check if already taken
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

	// clear any old seat this user had
	if occ.LeftUser == userEmail {
		occ.LeftUser = ""
	}
	if occ.CenterUser == userEmail {
		occ.CenterUser = ""
	}
	if occ.RightUser == userEmail {
		occ.RightUser = ""
	}

	// assign new seat
	switch position {
	case "left":
		occ.LeftUser = userEmail
	case "center":
		occ.CenterUser = userEmail
	case "right":
		occ.RightUser = userEmail
	}

	logger.Info.Printf("Position '%s' assigned to user '%s' for meet %s. Current occupancy: %+v",
		position, userEmail, meetName, occ)
	return nil
}

// ResetOccupancyForMeet clears all assigned referee positions for a given meet
func (s *OccupancyService) ResetOccupancyForMeet(meetName string) {
	occupancyMutex.Lock() // ✅ Use the global mutex
	defer occupancyMutex.Unlock()

	logger.Info.Printf("ResetOccupancyForMeet: Clearing all positions for meet '%s'", meetName)

	if occ, exists := occupancyMap[meetName]; exists {
		occ.LeftUser = ""
		occ.CenterUser = ""
		occ.RightUser = ""
	}
}

// UnsetPosition We define UnsetPosition as part of our service interface.
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

	logger.Info.Printf("Position '%s' was vacated by user '%s' for meet %s. Current occupancy: %+v",
		position, userEmail, meetName, occ)
	return nil
}

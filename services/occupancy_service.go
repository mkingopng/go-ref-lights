// Package services handles the business logic of the application, including referee position tracking.
package services

import (
	"errors"
	"sync"
	"time"

	"go-ref-lights/logger"
)

// Global mutex + map remain the same
var occupancyMutex sync.Mutex
var occupancyMap = make(map[string]*Occupancy)

type Occupancy struct {
	LeftUser    string
	CenterUser  string
	RightUser   string
	LastUpdated time.Time
}

type OccupancyServiceInterface interface {
	GetOccupancy(meetName string) Occupancy
	SetPosition(meetName, position, userEmail string) error
	UnsetPosition(meetName, position, userEmail string) error
	ResetOccupancyForMeet(meetName string)
}

type OccupancyService struct {
	mu        sync.Mutex
	occupancy map[string]*Occupancy
}

func NewOccupancyService() *OccupancyService {
	return &OccupancyService{
		occupancy: make(map[string]*Occupancy),
	}
}

// GetOccupancy is unchanged
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

// SetPosition now allows the same user to re‐enter the seat without error.
func (s *OccupancyService) SetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	occ, exists := occupancyMap[meetName]
	if !exists {
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}

	logger.Info.Printf("Attempting to assign position '%s' to user '%s' for meet %s", position, userEmail, meetName)

	// Validate position
	validPositions := map[string]bool{"left": true, "center": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected, please choose left, center, or right")
		logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
		return err
	}

	// Decide how to handle occupant vs. userEmail:
	// If occupant is "" or occupant == userEmail => allow
	// If occupant is some other user => error
	switch position {
	case "left":
		if occ.LeftUser != "" && occ.LeftUser != userEmail {
			err := errors.New("left position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	case "center":
		if occ.CenterUser != "" && occ.CenterUser != userEmail {
			err := errors.New("center position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	case "right":
		if occ.RightUser != "" && occ.RightUser != userEmail {
			err := errors.New("right position is already taken")
			logger.Error.Printf("SetPosition failed for meet %s: %v", meetName, err)
			return err
		}
	}

	// Remove the user from other positions if they're currently seated
	if occ.LeftUser == userEmail {
		occ.LeftUser = ""
	}
	if occ.CenterUser == userEmail {
		occ.CenterUser = ""
	}
	if occ.RightUser == userEmail {
		occ.RightUser = ""
	}

	// Now seat them in the chosen position
	switch position {
	case "left":
		occ.LeftUser = userEmail
	case "center":
		occ.CenterUser = userEmail
	case "right":
		occ.RightUser = userEmail
	}

	s.TouchActivity(meetName)
	logger.Info.Printf("Position '%s' assigned to user '%s' for meet %s. Current occupancy: %+v",
		position, userEmail, meetName, occ)
	return nil
}

// UnsetPosition is unchanged
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

// ResetOccupancyForMeet unchanged
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

// TouchActivity is unchanged
func (s *OccupancyService) TouchActivity(meetName string) {
	if occ, exists := occupancyMap[meetName]; exists {
		occ.LastUpdated = time.Now()
		logger.Debug.Printf("TouchActivity: Updated LastUpdated for meet %s to %v", meetName, occ.LastUpdated)
	}
}

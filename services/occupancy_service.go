// Package services handles the business logic of the application, including referee position tracking.
package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-ref-lights/models"
	"os"
	"sync"
	"time"

	"go-ref-lights/logger"
)

// Global mutex + map remain the same
var occupancyMutex sync.Mutex

// occupancyMap holds the occupancy state for each meetName.
var occupancyMap = make(map[string]*Occupancy)

// GlobalMeetCredentials holds the credentials for the meet.
var GlobalMeetCredentials *models.MeetCreds

// Occupancy holds occupant data for each position plus a timestamp.
type Occupancy struct {
	LeftUser    string
	CenterUser  string
	RightUser   string
	LastUpdated time.Time
}

// OccupancyServiceInterface defines the methods for managing occupancy states.
type OccupancyServiceInterface interface {
	GetOccupancy(meetName string) Occupancy
	SetPosition(meetName, position, userEmail string) error
	UnsetPosition(meetName, position, userEmail string) error
	ResetOccupancyForMeet(meetName string)
}

// OccupancyService is a concrete implementation of OccupancyServiceInterface.
type OccupancyService struct{}

// NewOccupancyService returns a pointer to a new OccupancyService.
func NewOccupancyService() *OccupancyService {
	return &OccupancyService{}
}

// GetOccupancy retrieves the occupancy state for a given meetName,
// creating a new empty Occupancy if it doesn’t exist yet.
func (s *OccupancyService) GetOccupancy(meetName string) Occupancy {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()
	occ, exists := occupancyMap[meetName]
	if !exists {
		logger.Info.Printf("[GetOccupancy] No occupancy found for meet=%s, creating new entry", meetName)
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	} else {
		logger.Debug.Printf("[GetOccupancy] Found existing occupancy for meet=%s", meetName)
	}
	logger.Debug.Printf("[GetOccupancy] Final occupancy state: meet=%s → Left=%s, Center=%s, Right=%s", meetName, occ.LeftUser, occ.CenterUser, occ.RightUser)
	return *occ
}

// SetPosition seats a user at a given position, allowing them to re-enter the seat if they’re already occupant.
func (s *OccupancyService) SetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	// fetch or create occupancy record
	occ, exists := occupancyMap[meetName]
	if !exists {
		logger.Info.Printf("[SetPosition] No occupancy for meet=%s — creating new", meetName)
		occ = &Occupancy{}
		occupancyMap[meetName] = occ
	}

	logger.Info.Printf("[SetPosition] ➡️ Attempting seat assignment: meet=%s, position=%s, user=%s",
		meetName, position, userEmail)

	// validate seat
	validPositions := map[string]bool{"left": true, "center": true, "right": true}
	if !validPositions[position] {
		err := errors.New("invalid position selected — must be left, center, or right")
		logger.Error.Printf("[SetPosition] ❌ Invalid position: position=%s, user=%s, meet=%s", position, userEmail, meetName)
		return err
	}

	// reject if position already occupied by someone else
	switch position {
	case "left":
		if occ.LeftUser != "" && occ.LeftUser != userEmail {
			err := errors.New("left position is already taken")
			logger.Warn.Printf("[SetPosition] 🚫 Conflict: left seat taken by %s — user=%s, meet=%s",
				occ.LeftUser, userEmail, meetName)
			return err
		}
	case "center":
		if occ.CenterUser != "" && occ.CenterUser != userEmail {
			err := errors.New("center position is already taken")
			logger.Warn.Printf("[SetPosition] 🚫 Conflict: center seat taken by %s — user=%s, meet=%s",
				occ.CenterUser, userEmail, meetName)
			return err
		}
	case "right":
		if occ.RightUser != "" && occ.RightUser != userEmail {
			err := errors.New("right position is already taken")
			logger.Warn.Printf("[SetPosition] 🚫 Conflict: right seat taken by %s — user=%s, meet=%s",
				occ.RightUser, userEmail, meetName)
			return err
		}
	}

	// clear user from any other seats
	if occ.LeftUser == userEmail {
		logger.Debug.Printf("[SetPosition] ↩️ Removing user=%s from left (already seated)", userEmail)
		occ.LeftUser = ""
	}
	if occ.CenterUser == userEmail {
		logger.Debug.Printf("[SetPosition] ↩️ Removing user=%s from center (already seated)", userEmail)
		occ.CenterUser = ""
	}
	if occ.RightUser == userEmail {
		logger.Debug.Printf("[SetPosition] ↩️ Removing user=%s from right (already seated)", userEmail)
		occ.RightUser = ""
	}

	// assign user to chosen seat
	switch position {
	case "left":
		occ.LeftUser = userEmail
	case "center":
		occ.CenterUser = userEmail
	case "right":
		occ.RightUser = userEmail
	}

	s.TouchActivity(meetName)
	logger.Info.Printf("[SetPosition] ✅ Assigned user=%s to %s seat in meet=%s. Final occupancy → Left=%s, Center=%s, Right=%s", userEmail, position, meetName, occ.LeftUser, occ.CenterUser, occ.RightUser)
	return nil
}

// UnsetPosition removes the occupant from a specified position
func (s *OccupancyService) UnsetPosition(meetName, position, userEmail string) error {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	// fetch occupancy record
	occ, exists := occupancyMap[meetName]
	if !exists {
		logger.Warn.Printf("[UnsetPosition] 🚫 No occupancy record for meet=%s — cannot unset position", meetName)
		return errors.New("no occupancy found for that meet")
	}

	logger.Info.Printf("[UnsetPosition] ➡️ Attempting to unset: meet=%s, position=%s, user=%s", meetName, position, userEmail)

	switch position {
	case "left":
		if occ.LeftUser == userEmail {
			logger.Info.Printf("[UnsetPosition] ✅ Clearing LEFT seat for user=%s in meet=%s", userEmail, meetName)
			occ.LeftUser = ""
		} else {
			logger.Warn.Printf("[UnsetPosition] ❌ LEFT seat mismatch: expected=%s, actual=%s", userEmail, occ.LeftUser)
			return errors.New("user does not hold this position")
		}

	case "center":
		if occ.CenterUser == userEmail {
			logger.Info.Printf("[UnsetPosition] ✅ Clearing CENTER seat for user=%s in meet=%s", userEmail, meetName)
			occ.CenterUser = ""
		} else {
			logger.Warn.Printf("[UnsetPosition] ❌ CENTER seat mismatch: expected=%s, actual=%s", userEmail, occ.CenterUser)
			return errors.New("user does not hold this position")
		}

	case "right":
		if occ.RightUser == userEmail {
			logger.Info.Printf("[UnsetPosition] ✅ Clearing RIGHT seat for user=%s in meet=%s", userEmail, meetName)
			occ.RightUser = ""
		} else {
			logger.Warn.Printf("[UnsetPosition] ❌ RIGHT seat mismatch: expected=%s, actual=%s", userEmail, occ.RightUser)
			return errors.New("user does not hold this position")
		}

	default:
		err := errors.New("invalid position")
		logger.Error.Printf("[UnsetPosition] ❌ Invalid position specified: %s", position)
		return err
	}

	logger.Debug.Printf("[UnsetPosition] Final occupancy state for meet=%s: Left=%s, Center=%s, Right=%s", meetName, occ.LeftUser, occ.CenterUser, occ.RightUser)
	return nil
}

// ResetOccupancyForMeet clears all occupant fields for the specified meet
func (s *OccupancyService) ResetOccupancyForMeet(meetName string) {
	occupancyMutex.Lock()
	defer occupancyMutex.Unlock()

	logger.Info.Printf("[ResetOccupancyForMeet] 🔄 Request to clear all positions for meet=%s", meetName)

	occ, exists := occupancyMap[meetName]
	if !exists {
		logger.Warn.Printf("[ResetOccupancyForMeet] ⚠️ No occupancy record found for meet=%s — nothing to reset", meetName)
		return
	}

	logger.Debug.Printf("[ResetOccupancyForMeet] Current occupancy before reset: %+v", occ)

	// clear all positions
	occ.LeftUser = ""
	occ.CenterUser = ""
	occ.RightUser = ""

	logger.Info.Printf("[ResetOccupancyForMeet] ✅ Reset successful for meet=%s. All positions cleared.", meetName)
}

// TouchActivity updates the LastUpdated timestamp for the given meet
// NOTE: Caller must hold occupancyMutex.
func (s *OccupancyService) TouchActivity(meetName string) {
	if occ, exists := occupancyMap[meetName]; exists {
		occ.LastUpdated = time.Now()
		logger.Debug.Printf("[TouchActivity] Updated LastUpdated for meet=%s to %v", meetName, occ.LastUpdated)
	}
}

// LoadJSONFile is a generic helper that reads a file and unmarshals JSON into 'target'
func LoadJSONFile(path string, target interface{}) error {
	logger.Debug.Printf("[LoadJSONFile] Attempting to read file: %s", path)

	// #nosec G304
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error.Printf("[LoadJSONFile] Failed to read file %s: %v", path, err)
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	logger.Debug.Printf("[LoadJSONFile] Successfully read file: %s (size=%d bytes)", path, len(data))

	// Unmarshal the JSON data into the target struct
	if err := json.Unmarshal(data, target); err != nil {
		logger.Error.Printf("[LoadJSONFile] Failed to unmarshal JSON from file %s: %v", path, err)
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	logger.Info.Printf("[LoadJSONFile] Successfully loaded JSON from file: %s", path)
	return nil
}

// LoadBasicMeets loads the basic meets info from config/meets.json
func LoadBasicMeets() (*models.BasicMeets, error) {
	path := "config/meets.json" // #nosec G304
	if env := os.Getenv("MEETS_PATH"); env != "" {
		path = env
		logger.Debug.Printf("[LoadBasicMeets] Overriding path from MEETS_PATH env var: %s", path)
	} else {
		logger.Debug.Printf("[LoadBasicMeets] Using default path: %s", path)
	}

	// load the JSON file
	var basic models.BasicMeets
	if err := LoadJSONFile(path, &basic); err != nil {
		logger.Error.Printf("[LoadBasicMeets] Failed to load basic meets from %s: %v", path, err)
		return nil, err
	}

	logger.Info.Printf("[LoadBasicMeets] Loaded %d meets from %s", len(basic.Meets), path)
	return &basic, nil
}

// LoadMeetCredentials loads the meet credentials from config/meet_creds.json
func LoadMeetCredentials() (*models.MeetCreds, error) {
	path := "config/meet_creds.json"
	if env := os.Getenv("MEET_CREDS_PATH"); env != "" {
		path = env
		logger.Debug.Printf("[LoadMeetCredentials] Overriding path from MEET_CREDS_PATH env var: %s", path)
	} else {
		logger.Debug.Printf("[LoadMeetCredentials] Using default path: %s", path)
	}

	var creds models.MeetCreds
	if err := LoadJSONFile(path, &creds); err != nil {
		logger.Error.Printf("[LoadMeetCredentials] Failed to load credentials from %s: %v", path, err)
		return nil, err
	}

	logger.Info.Printf("[LoadMeetCredentials] Loaded credentials from %s", path)
	return &creds, nil
}

// SetGlobalMeetCredentials sets the global meet credentials
func SetGlobalMeetCredentials(c *models.MeetCreds) {
	GlobalMeetCredentials = c
	logger.Info.Printf("[SetGlobalMeetCredentials] Global meet credentials set. Meets count: %d", len(c.Meets))
}

// GetGlobalMeetCredentials returns the global meet credentials
func GetGlobalMeetCredentials() *models.MeetCreds {
	if GlobalMeetCredentials == nil {
		logger.Warn.Println("[GetGlobalMeetCredentials] No global meet credentials have been set")
	} else {
		logger.Debug.Printf("[GetGlobalMeetCredentials] Returning global meet credentials. Meets count: %d", len(GlobalMeetCredentials.Meets))
	}
	return GlobalMeetCredentials
}

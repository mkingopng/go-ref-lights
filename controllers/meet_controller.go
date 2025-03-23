// Package controllers handles meet selection and configuration management.
// File: controllers/meet_controller.go
package controllers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
)

// --------------- global variables ---------------

// loadMeetsFunc allows dependency injection for testing.
var loadMeetsFunc = LoadMeets

// ------------- meet configuration management -------------

// LoadMeets loads the meet configuration from `./config/meets.json`.
// This function retrieves the available meets and their details from the JSON file.
func LoadMeets() (*models.MeetCreds, error) {
	// read the config file
	data, err := os.ReadFile("./config/meets.json")
	if err != nil {
		return nil, err
	}

	var meets models.MeetCreds
	if err := json.Unmarshal(data, &meets); err != nil {
		return nil, err
	}

	logger.Info.Printf("[LoadMeets] Successfully loaded %d meets", len(meets.Meets))
	return &meets, nil
}

// -------------- meet selection handling --------------

// ShowMeets renders the meet selection page.
// It fetches the list of available meets and passes them to the template.
// If loading fails, it returns an HTTP 500 response.
func ShowMeets(c *gin.Context) {
	// retrieve meet data using a mockable function for easier testing
	meetsData, err := loadMeetsFunc()
	if err != nil {
		logger.Error.Printf("[ShowMeets] Failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	// render the meet selection page with available meets
	c.HTML(http.StatusOK, "choose_meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

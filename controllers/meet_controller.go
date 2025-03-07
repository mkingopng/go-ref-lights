// Package controllers handles user authentication and session management
// file: controllers/meet_controller.go
package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"net/http"
	"os"
)

var loadMeetsFunc = LoadMeets // ✅ Use a variable for easy mocking

// LoadMeets loads the meet configuration from ./config/meets.json.
func LoadMeets() (*models.MeetCreds, error) {
	data, err := os.ReadFile("./config/meets.json")
	if err != nil {
		return nil, err
	}
	var meets models.MeetCreds
	if err := json.Unmarshal(data, &meets); err != nil {
		return nil, err
	}
	return &meets, nil
}

// ShowMeets renders the meeting selection page.
func ShowMeets(c *gin.Context) {
	meetsData, err := loadMeetsFunc() // ✅ Use mockable function
	if err != nil {
		logger.Error.Printf("ShowMeets: failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}
	c.HTML(http.StatusOK, "choose_meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

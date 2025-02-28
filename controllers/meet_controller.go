// Package controllers: controllers/meet_controller.go
package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"net/http"
	"os"
)

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
	meetsData, err := LoadMeets()
	if err != nil {
		logger.Error.Printf("ShowMeets: failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}
	c.HTML(http.StatusOK, "choose_meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

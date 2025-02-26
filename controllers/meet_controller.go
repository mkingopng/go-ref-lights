// Package controllers controllers/meet_controller.go
package controllers

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// Meet represents a single meet entry.
type Meet struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Date string `json:"date"`
}

// Meets is a wrapper for multiple meets.
type Meets struct {
	Meets []Meet `json:"meets"`
}

// ShowMeets renders the meeting selection page.
func ShowMeets(c *gin.Context) {
	// Adjust the path as necessary. Here we assume the config file is in "config/meets.json"
	data, err := os.ReadFile("config/meets.json")
	if err != nil {
		logger.Error.Printf("ShowMeets: failed to read config file: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	var meets Meets
	if err := json.Unmarshal(data, &meets); err != nil {
		logger.Error.Printf("ShowMeets: failed to parse config file: %v", err)
		c.String(http.StatusInternalServerError, "Failed to parse meets")
		return
	}

	// Render the choose_meet.html template with the meets data.
	c.HTML(http.StatusOK, "choose_meet.html", meets)
}

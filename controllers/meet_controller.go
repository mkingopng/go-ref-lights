package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"

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

// LoadMeets loads the meets configuration from config/meets.json.
func LoadMeets() (*Meets, error) {
	data, err := os.ReadFile("config/meets.json")
	if err != nil {
		return nil, err
	}
	var meets Meets
	if err := json.Unmarshal(data, &meets); err != nil {
		return nil, err
	}
	return &meets, nil
}

// Package controllers config/meets.json
package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"net/http"
	"os"
)

// Meet represents a single meet entry.
type Meet struct {
	Name string `json:"name"`
	Date string `json:"date"`
}

// Meets is a wrapper for multiple meets.
type Meets struct {
	Meets []Meet `json:"meets"`
}

// LoadMeets loads the meet configuration from ./config/meets.json.
func LoadMeets() (*Meets, error) {
	data, err := os.ReadFile("./config/meets.json")
	if err != nil {
		return nil, err
	}
	var meets Meets
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

// Package controllers: controllers/meet_controller.go
package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

// MeetsData structure to hold the data from meets.json
type MeetsData struct {
	Meets []struct {
		Name  string `json:"name"`
		Date  string `json:"date"`
		Users []struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"users"`
	} `json:"meets"`
}

// LoadMeets reads config/meets.json and returns the parsed data
func LoadMeets() (*MeetsData, error) {
	// Use a relative path so it works in Docker and local:
	filePath := "config/meets.json"

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var data MeetsData
	if err := json.Unmarshal(fileBytes, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ShowMeets loads the meets data and renders the meets.html template
func ShowMeets(c *gin.Context) {
	meets, err := LoadMeets()
	if err != nil {
		log.Printf("ShowMeets: failed to load meets: %v", err)
		// Make sure "error.html" actually exists or choose a real template
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrorMessage": "Failed to load meets data.",
		})
		return
	}
	c.HTML(http.StatusOK, "meets.html", gin.H{
		"Meets": meets,
	})
}

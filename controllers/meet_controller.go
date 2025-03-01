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

// LoadMeets reads ./config/meets.json and returns the parsed data
func LoadMeets() (*MeetsData, error) {
	// EXACT PATH reference
	filePath := "./config/meets.json"

	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		// Could log an error or just return
		return nil, err
	}
	var data MeetsData
	if err := json.Unmarshal(fileBytes, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ShowMeets is an example endpoint that reads the meets data
// and renders it or returns JSON
func ShowMeets(c *gin.Context) {
	meets, err := LoadMeets()
	if err != nil {
		// Log the error and show a user-friendly message
		log.Printf("ShowMeets: failed to load meets: %v", err)
		// You might want to do c.HTML(...) or c.JSON(...) with an error
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"ErrorMessage": "Failed to load meets data.",
		})
		return
	}

	// If successful, you could pass the data to your template or JSON
	c.HTML(http.StatusOK, "meets.html", gin.H{
		"Meets": meets,
	})
}

// Package controllers controllers/auth_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
)

var activeUsers = make(map[string]bool)

var loadMeetCredsFunc = LoadMeetCreds // Assign to a variable for easier testing

// ComparePasswords checks if the given password matches the hashed password
func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// SetMeetHandler saves the selected meetName in session.
func SetMeetHandler(c *gin.Context) {
	session := sessions.Default(c)

	meetName := c.PostForm("meetName")
	if meetName == "" {
		c.HTML(http.StatusBadRequest, "choose_meet.html", gin.H{"Error": "Please select a meet."})
		return
	}

	session.Set("meetName", meetName)
	if err := session.Save(); err != nil {
		logger.Error.Println("Failed to save meet session:", err)
		c.HTML(http.StatusInternalServerError, "choose_meet.html", gin.H{"Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("Meet %s selected, redirecting to login.", meetName)
	c.Redirect(http.StatusFound, "/login")
}

// LoadMeetCreds loads meet credentials from JSON file
// LoadMeetCreds loads meet credentials from JSON file
func LoadMeetCreds() (*models.MeetCreds, error) {
	credPath := "./config/meet_creds.json" // #nosec G101

	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{} // Load raw JSON first
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse meet_creds.json: %w", err)
	}

	// Convert "isadmin" from string to boolean if needed
	meetsData, _ := raw["meets"].([]interface{})
	for _, meet := range meetsData {
		meetMap := meet.(map[string]interface{})
		usersData, _ := meetMap["users"].([]interface{})

		for _, user := range usersData {
			userMap := user.(map[string]interface{})
			if isAdminStr, ok := userMap["isadmin"].(string); ok {
				userMap["isadmin"] = (isAdminStr == "True" || isAdminStr == "true")
			}
		}
	}

	// Convert back to models.MeetCreds
	parsedData, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encode JSON: %w", err)
	}

	var creds models.MeetCreds
	if err := json.Unmarshal(parsedData, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse corrected JSON: %w", err)
	}

	// ✅ Debug print to confirm "isadmin" is correctly parsed
	for _, meet := range creds.Meets {
		for _, user := range meet.Users {
			fmt.Printf("Loaded user: %s (Admin: %t)\n", user.Username, user.IsAdmin)
		}
	}

	return &creds, nil
}

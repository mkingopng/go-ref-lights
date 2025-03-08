// Package controllers provides authentication and session management for users.
// File: controllers/auth_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"golang.org/x/crypto/bcrypt"
)

// ---------- global variables ----------

// activeUsers tracks currently logged-in users.
var activeUsers = make(map[string]bool)

// loadMeetCredsFunc allows dependency injection for testing.
var loadMeetCredsFunc = LoadMeetCreds // Assign to a variable for easier testing

// ----------------------- authentication utilities -----------------------

// ComparePasswords checks if the given password matches the hashed password
func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// ----------------------- meet selection ----------------------------------

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

// ----------------------- credentials management ---------------------------

// LoadMeetCreds loads meet credentials from a JSON file.
// This function ensures that the `isadmin` field is properly converted
// from string to boolean when necessary.
func LoadMeetCreds() (*models.MeetCreds, error) {
	credPath := "./config/meet_creds.json" // #nosec G101 - This is a known, controlled file path.

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

	// Debug print to confirm "isadmin" is correctly parsed
	for _, meet := range creds.Meets {
		for _, user := range meet.Users {
			fmt.Printf("Loaded user: %s (Admin: %t)\n", user.Username, user.IsAdmin)
		}
	}

	return &creds, nil
}

// ----------------------- admin actions -----------------------------------

// ForceLogoutHandler forcibly logs out a user (admin action).
// Requires:
// - `username` from the POST request body.
// - The user to have admin privileges.
func ForceLogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	// Only admins can force logout users
	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to force logout a user.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	// Extract username from request
	username := c.PostForm("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	// Ensure the user exists in activeUsers
	if _, exists := activeUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	// Remove user session and mark them as logged out
	delete(activeUsers, username)
	logger.Info.Printf("Admin forcibly logged out user: %s", username)

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// --------------------- active user tracking ------------------------------

// ActiveUsersHandler returns a list of active users (admin action).
// Requires admin privileges.
func ActiveUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	// Only admins can see active users
	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to view active users.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	// Convert activeUsers map keys to a list
	var userList []string
	for user := range activeUsers {
		userList = append(userList, user)
	}

	c.JSON(http.StatusOK, gin.H{"users": userList})
}

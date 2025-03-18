// Package controllers provides authentication and session management for users.
// File: controllers/auth_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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
// LoadMeetCreds loads meet credentials from a JSON file
func LoadMeetCreds() (*models.MeetCreds, error) {
	credPath := "./config/meet_creds.json" // #nosec G101

	// Read JSON file
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read meet credentials file: %w", err)
	}

	// Unmarshal JSON directly into MeetCreds struct
	var creds models.MeetCreds
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse meet_creds.json: %w", err)
	}

	// Ensure all admin credentials are correctly loaded
	for _, meet := range creds.Meets {
		if meet.Admin.Username == "" {
			return nil, fmt.Errorf("error: Meet '%s' is missing an admin username", meet.Name)
		}
		if meet.Admin.Password == "" || !strings.HasPrefix(meet.Admin.Password, "$2b$12$") {
			return nil, fmt.Errorf("error: Meet '%s' is missing a valid hashed password", meet.Name)
		}
		// Debug log for confirmation
		fmt.Printf("✅ Loaded Meet: %s (Admin: %s, IsAdmin: %t)\n", meet.Name, meet.Admin.Username, meet.Admin.IsAdmin)
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

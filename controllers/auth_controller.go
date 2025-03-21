// Package controllers provides authentication and session management for users.
// File: controllers/auth_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"golang.org/x/crypto/bcrypt"
)

// ---------- global variables ----------

// ActiveUsers tracks currently logged-in users.
var ActiveUsers = make(map[string]bool)
var ActiveUsersMu sync.RWMutex

// loadMeetCredsFunc allows dependency injection for testing.
var loadMeetCredsFunc = LoadMeetCreds // Assign to a variable for easier testing

// ----------------------- authentication utilities -----------------------

// In auth_controller.go (or in a _test.go file in the same package)
// Provide a helper so your test can lock/unlock or set users as needed:

func lockActiveUsers() {
	ActiveUsersMu.Lock()
}

func unlockActiveUsers() {
	ActiveUsersMu.Unlock()
}

func setUserActive(username string) {
	ActiveUsers[username] = true
}

func clearUserActive(username string) {
	delete(ActiveUsers, username)
}

// ComparePasswords checks if the given password matches the hashed password
func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// SetMeetHandler sets the selected meet in the session and redirects to the meet page.
func SetMeetHandler(c *gin.Context) {
	meetName := c.PostForm("meetName")
	if meetName == "" {
		c.HTML(http.StatusBadRequest, "choose_meet.html", gin.H{"Error": "Please select a meet."})
		return
	}

	session := sessions.Default(c)
	session.Set("meetName", meetName)
	if err := session.Save(); err != nil {
		logger.Error.Printf("Failed to save meet session: %v", err)
		c.HTML(http.StatusInternalServerError, "choose_meet.html", gin.H{"Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("Meet %s selected, redirecting to meet page.", meetName)
	c.Redirect(http.StatusFound, "/login")
}

// ----------------------- meet selection ----------------------------------

// MeetHandler retrieves the meet details from session and renders the home page with the appropriate logo.
func MeetHandler(c *gin.Context) {
	session := sessions.Default(c)
	storedMeet := session.Get("meetName")
	if storedMeet == nil {
		c.HTML(http.StatusBadRequest, "choose_meet.html", gin.H{"Error": "No meet selected."})
		return
	}
	meetName := storedMeet.(string)

	// load meet credentials using the injectable function.
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Printf("Failed to load meets: %v", err)
		c.HTML(http.StatusInternalServerError, "choose_meet.html", gin.H{"Error": "Internal error loading meets."})
		return
	}

	// find the meet with the matching name.
	var currentMeet *models.Meet
	for _, meet := range creds.Meets {
		if meet.Name == meetName {
			currentMeet = &meet
			break
		}
	}
	if currentMeet == nil {
		c.HTML(http.StatusNotFound, "choose_meet.html", gin.H{"Error": "Meet not found."})
		return
	}

	// prepare data for the template.
	data := gin.H{
		"meetName": currentMeet.Name,
		"logo":     currentMeet.Logo,
	}

	// render the template with the correct logo.
	c.HTML(http.StatusOK, "index.html", data)
}

// ----------------------- credentials management ---------------------------

// LoadMeetCreds loads meet credentials from a JSON file
func LoadMeetCreds() (*models.MeetCreds, error) {
	// define the path to the credentials file.
	credPath := "./config/meet_creds.json" // #nosec G101

	// read the JSON file
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read meet credentials file: %w", err)
	}

	// unmarshal JSON into MeetCreds struct
	var creds models.MeetCreds
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse meet_creds.json: %w", err)
	}

	// validate admin credentials for each meet.
	for _, meet := range creds.Meets {
		if meet.Admin.Username == "" {
			return nil, fmt.Errorf("error: Meet '%s' is missing an admin username", meet.Name)
		}
		if meet.Admin.Password == "" || !strings.HasPrefix(meet.Admin.Password, "$2b$12$") {
			return nil, fmt.Errorf("error: Meet '%s' is missing a valid hashed password", meet.Name)
		}
		// replaced the direct fmt.Printf with a logger call
		logger.Debug.Printf("Loaded Meet: %s (Admin: %s, IsAdmin: %t)",
			meet.Name, meet.Admin.Username, meet.Admin.IsAdmin)
	}
	return &creds, nil
}

// ----------------------- admin actions -----------------------------------

// ForceLogoutHandler forcibly logs out a user (admin action).
// Requires: `username` from the POST request body.
func ForceLogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to force logout a user.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	username := c.PostForm("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	// Acquire the write lock for read-check + deletion
	ActiveUsersMu.Lock()
	defer ActiveUsersMu.Unlock()

	if _, exists := ActiveUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	delete(ActiveUsers, username)
	logger.Info.Printf("Admin forcibly logged out user: %s", username)

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// --------------------- active user tracking ------------------------------

// ActiveUsersHandler returns a list of currently active users (admin action).
func ActiveUsersHandler(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Println("Unauthorized attempt to view active users.")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	var userList []string

	// Acquire read lock for iteration
	ActiveUsersMu.RLock()
	for user := range ActiveUsers {
		userList = append(userList, user)
	}
	ActiveUsersMu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"users": userList})
}

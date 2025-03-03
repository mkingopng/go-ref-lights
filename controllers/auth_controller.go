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
	"runtime"
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
func LoadMeetCreds() (*models.MeetCreds, error) {
	_, _, _, _ = runtime.Caller(0) // Unused variable fix
	credPath := "./config/meet_creds.json"

	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, err
	}

	var creds models.MeetCreds
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	// Debug print to confirm meets are loaded correctly
	fmt.Println("Loaded meets:", creds.Meets)

	return &creds, nil
}

// LoginHandler verifies the username and password, enforces single login,
// and stores session data if successful.
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		logger.Warn.Println("LoginHandler: No meet selected, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		logger.Warn.Println("LoginHandler: Missing username or password")

		// ✅ If in test mode, return JSON instead of HTML
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please fill in all fields."})
			return
		}

		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// Load credentials
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)

		// ✅ Return JSON in test mode
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// Validate credentials
	var valid bool
	for _, m := range creds.Meets {
		if m.Name == meetName {
			for _, user := range m.Users {
				if user.Username == username && ComparePasswords(user.Password, password) {
					valid = true
					break
				}
			}
		}
	}

	if !valid {
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)

		// ✅ Return JSON in test mode
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// Single login enforcement
	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)

		// ✅ Return JSON in test mode
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "This username is already logged in on another device."})
			return
		}

		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "This username is already logged in on another device.",
		})
		return
	}

	// Mark user as logged in
	activeUsers[username] = true
	session.Set("user", username)
	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)

		// ✅ Return JSON in test mode
		if gin.Mode() == gin.TestMode {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
			return
		}

		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s", username, meetName)
	c.Redirect(http.StatusFound, "/dashboard")
}

// LogoutHandler Provide a logout function that removes the user from activeUsers, so they can log in again from another device.
func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)

	// Attempt to get the username from session
	user, hasUser := session.Get("user").(string)
	if hasUser && user != "" {
		// Remove the user from the activeUsers map
		delete(activeUsers, user)
		logger.Info.Printf("LogoutHandler: Removed user %s from active list", user)
	}

	// Clear out all session data
	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("LogoutHandler: Error saving session during logout: %v", err)
	} else {
		logger.Info.Println("LogoutHandler: Session cleared successfully")
	}

	// You can redirect to a "logged out" page or back to choose-meet, etc.
	c.Redirect(http.StatusFound, "/choose-meet")
}

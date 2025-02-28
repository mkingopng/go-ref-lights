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

// ComparePasswords checks if the given password matches the hashed password
func ComparePasswords(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// SetMeetHandler saves the selected meetName in session.
// We’ve removed the old check that prevented the user from changing the meet.
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
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// Load credentials from config
	creds, err := LoadMeetCreds()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// Validate user credentials against the chosen meet
	var valid bool
	for _, m := range creds.Meets {
		if m.Name == meetName {
			for _, user := range m.Users {
				// If we find a username match and the password check passes, it's valid
				if user.Username == username && ComparePasswords(user.Password, password) {
					valid = true
					break
				}
			}
		}
	}

	// If no match found, return an error
	if !valid {
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// -------------------------------------------------------------------------
	// Single-login enforcement:
	// If the user is already in activeUsers, that means they have a live session.
	// We deny the new session attempt.  (They can log out to free up the old session.)
	// -------------------------------------------------------------------------
	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "This username is already logged in on another device.",
		})
		return
	}

	// Mark the user as active. If we reach here, they either:
	// 1) haven't logged in before, or
	// 2) they have logged out from the old session properly.
	activeUsers[username] = true

	// Save user info in session
	session.Set("user", username)
	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s", username, meetName)
	c.Redirect(http.StatusFound, "/dashboard")
}

// -----------------------------------------------------------------------------
//  2. Provide a logout function that removes the user from activeUsers,
//     so they can log in again from another device.
//     Note: If your existing logout is in a different file (e.g. page_controller.go),
//     you can adapt the code below and call it from there. The key is to delete
//     the username from activeUsers so they can log in again later.
//
// -----------------------------------------------------------------------------
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

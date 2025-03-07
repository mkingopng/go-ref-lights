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
	_, _, _, _ = runtime.Caller(0)         // Unused variable fix
	credPath := "./config/meet_creds.json" // #nosec G101

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
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"MeetName": meetName, "Error": "Please fill in all fields."})
		return
	}

	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	var valid bool

	if username == adminUser && password == adminPassword {
		valid = true
	} else {
		creds, err := loadMeetCredsFunc()
		if err != nil {
			logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
			c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again later."})
			return
		}

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
	}

	if !valid {
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"MeetName": meetName, "Error": "Invalid username or password."})
		return
	}

	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"MeetName": meetName, "Error": "This username is already logged in on another device."})
		return
	}

	// Mark user as logged in and set session user
	activeUsers[username] = true
	session.Set("user", username)

	// ✅ Check if user is admin and explicitly log the result
	isAdmin := (username == adminUser)
	session.Set("isAdmin", isAdmin)

	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)

	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)
	c.Redirect(http.StatusFound, "/dashboard")
}

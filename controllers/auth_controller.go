// Package controllers controllers/auth_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/models"
	"net/http"
	"os"
	"runtime"
)

// SetMeetHandler saves the selected meetName in session
func SetMeetHandler(c *gin.Context) {
	session := sessions.Default(c)

	// Prevent selecting a different meet after already setting one
	if session.Get("meetName") != nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

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

// LoginHandler verifies the username and password
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"MeetName": meetName, "Error": "Please fill in all fields."})
		return
	}

	creds, err := LoadMeetCreds()
	if err != nil {
		logger.Error.Println("Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again later."})
		return
	}

	var valid bool
	for _, m := range creds.Meets {
		if m.Name == meetName {
			for _, user := range m.Users {
				if user.Username == username && user.Password == password {
					valid = true
					break
				}
			}
		}
	}

	if !valid {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"MeetName": meetName, "Error": "Invalid username or password."})
		return
	}

	// Save user info in session
	session.Set("user", username)
	if err := session.Save(); err != nil {
		logger.Error.Println("Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("User %s authenticated for meet %s", username, meetName)
	c.Redirect(http.StatusFound, "/dashboard")
}

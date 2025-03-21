// Package controllers handles user authentication and session management.
// File: controllers/loginHandler.go
package controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"golang.org/x/crypto/bcrypt"
)

var occupancyService services.OccupancyServiceInterface

// ------------------ authentication utilities ------------------

// checkPasswordHash verifies if the provided plain-text password matches the stored hashed password.
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// PerformLogin captures meetName & position from query params for the login page
// Called when user first arrives at /login?meetName=foo&position=left, for example
func PerformLogin(c *gin.Context) {
	session := sessions.Default(c)

	// grab from the query string
	meetNameParam := c.Query("meetName")
	posParam := c.Query("position")

	// if present, store them in session
	if meetNameParam != "" {
		session.Set("meetName", meetNameParam)
	}
	if posParam != "" {
		session.Set("desiredPosition", posParam)
	}

	// persist session changes
	if err := session.Save(); err != nil {
		logger.Error.Printf("[PerformLogin] Failed to save session: %v", err)
	}

	// finally, render the login form
	meetName := session.Get("meetName")
	logo := "" // default empty

	// Try to load logo from meet config
	if meetNameStr, ok := meetName.(string); ok && meetNameStr != "" {
		if creds, err := loadMeetCredsFunc(); err == nil {
			for _, meet := range creds.Meets {
				if meet.Name == meetNameStr {
					logo = meet.Logo
					break
				}
			}
		}
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"MeetName": meetName,
		"Logo":     logo,
	})
}

// ------------------ login handling ------------------

// LoginHandler authenticates the user, prevents duplicate logins, and manages session storage.
// If successful, it redirects:
// - Admin users → `/admin`
// - Regular users → `/index`
// If authentication fails, it returns an appropriate error message.
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c)

	// retrieve meet name from session
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		logger.Warn.Println("[LoginHandler] No meet selected, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	// extract username and password from the POST form
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || password == "" {
		logger.Warn.Println("[LoginHandler] Missing username or password")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// load meet credentials
	creds, err := loadMeetCredsFunc()

	if err != nil {
		logger.Error.Printf("[LoginHandler] Failed to load meet credentials: %v", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// check for superuser login
	if creds.Superuser != nil &&
		creds.Superuser.Username == username &&
		checkPasswordHash(password, creds.Superuser.Password) {
		session.Set("sudo", true)
		session.Set("isAdmin", true)
		session.Set("user", username)
		_ = session.Save()

		logger.Info.Printf("[LoginHandler] Superuser %s authenticated", username)
		c.Redirect(http.StatusFound, "/sudo")
		return
	}

	// validate the provided credentials against the selected meet
	var isAdmin bool
	var authenticated bool
	for _, m := range creds.Meets {
		if m.Name != meetName {
			continue
		}

		// primary admin
		if m.Admin.Username == username && checkPasswordHash(password, m.Admin.Password) {
			isAdmin = m.Admin.IsAdmin
			authenticated = true
			break
		}

		// secondary admins
		for _, sa := range m.SecondaryAdmins {
			if sa.Username == username && checkPasswordHash(password, sa.Password) {
				isAdmin = sa.IsAdmin
				authenticated = true
				break
			}
		}

		break // stop after checking this meet
	}

	if !authenticated {
		logger.Warn.Printf("[LoginHandler] Invalid login attempt for user=%s at meet=%s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// prevent duplicate logins
	ActiveUsersMu.Lock()
	if ActiveUsers[username] {
		logger.Warn.Printf("[LoginHandler] User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
			"Logo":     getLogoForMeet(meetName), // helper function
		})
		ActiveUsersMu.Unlock()
		return
	}
	ActiveUsers[username] = true
	ActiveUsersMu.Unlock()

	session.Set("user", username)
	session.Set("isAdmin", isAdmin)
	logger.Debug.Printf("[LoginHandler] Setting isAdmin=%v for user=%s", isAdmin, username)

	if err := session.Save(); err != nil {
		logger.Error.Printf("[LoginHandler] Failed to save session: %v", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("[LoginHandler] User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// ------------------ auto-claim desired position ------------------
	desiredPos := session.Get("desiredPosition")
	if desiredPos != nil {
		logger.Info.Printf("[LoginHandler] Attempting to auto-claim position=%s for user=%s", desiredPos, username)
		posString := desiredPos.(string)
		if err := occupancyService.SetPosition(meetName, posString, username); err != nil {
			logger.Warn.Printf("[LoginHandler] Auto-claim failed for user=%s on position=%s: %v", username, posString, err)
			c.HTML(http.StatusForbidden, "positions.html", gin.H{
				"Error":    "Position is already taken or invalid. Please choose another.",
				"meetName": meetName,
			})
			return
		}
		session.Set("refPosition", posString)
		_ = session.Save()

		switch posString {
		case "left":
			c.Redirect(http.StatusFound, "/left")
		case "center":
			c.Redirect(http.StatusFound, "/center")
		case "right":
			c.Redirect(http.StatusFound, "/right")
		default:
			c.Redirect(http.StatusFound, "/positions")
		}
		return
	}

	// ------------------ default redirect on success ------------------
	c.Redirect(http.StatusFound, "/index")
}

// Helper function to retrieve logo for meet
func getLogoForMeet(meetName string) string {
	creds, err := loadMeetCredsFunc()
	if err != nil {
		return ""
	}
	for _, meet := range creds.Meets {
		if meet.Name == meetName {
			return meet.Logo
		}
	}
	return ""
}

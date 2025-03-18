// Package controllers handles user authentication and session management.
// File: controllers/loginHandler.go
package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"golang.org/x/crypto/bcrypt"
	"net/http"
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

	// Grab from the query string
	meetNameParam := c.Query("meetName")
	posParam := c.Query("position")

	// If present, store them in session
	if meetNameParam != "" {
		session.Set("meetName", meetNameParam)
	}
	if posParam != "" {
		session.Set("desiredPosition", posParam)
	}

	// Persist session changes
	if err := session.Save(); err != nil {
		logger.Error.Println("PerformLogin: Failed to save session:", err)
	}

	// Finally, render the login form
	c.HTML(http.StatusOK, "login.html", gin.H{
		"MeetName": session.Get("meetName"),
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

	// Retrieve meet name from session.
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		// No meet selected: redirect to the choose-meet page.
		logger.Warn.Println("LoginHandler: No meet selected, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	// Extract username and password from the POST form.
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate that both fields are provided.
	if username == "" || password == "" {
		logger.Warn.Println("LoginHandler: Missing username or password")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName, // preserve the meet name
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// Load meet credentials using your helper.
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// Validate the provided credentials against the stored admin credentials for the meet.
	var valid bool
	var isAdmin bool
	for _, m := range creds.Meets {
		if m.Name == meetName {
			// Compare the given password with the stored hash.
			if m.Admin.Username == username && checkPasswordHash(password, m.Admin.Password) {
				valid = true
				isAdmin = m.Admin.IsAdmin
				break
			}
		}
	}

	if !valid {
		// Credentials are invalid.
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// Prevent duplicate logins.
	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "This username is already logged in on another device.",
		})
		return
	}
	// Mark the user as logged in.
	activeUsers[username] = true
	session.Set("user", username)
	session.Set("isAdmin", isAdmin)

	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)

	// Attempt to save the session.
	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}
	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// ------------------ Auto-Claim Desired Position ------------------
	// If a desired position was set in the session, attempt to claim that seat.
	desiredPos := session.Get("desiredPosition")
	if desiredPos != nil {
		logger.Info.Printf("LoginHandler: Attempting to auto-claim position %s for user %s", desiredPos, username)
		posString := desiredPos.(string)
		if err := occupancyService.SetPosition(meetName, posString, username); err != nil {
			// If auto-claim fails (position taken or invalid), render the positions page with an error.
			logger.Warn.Printf("LoginHandler: Auto-claim failed for %s on %s: %v", username, posString, err)
			c.HTML(http.StatusForbidden, "positions.html", gin.H{
				"Error":    "Position is already taken or invalid. Please choose another.",
				"meetName": meetName,
			})
			return
		}
		// Save the claimed position in session.
		session.Set("refPosition", posString)
		_ = session.Save() // Ignoring error for brevity

		// Redirect to the corresponding referee view.
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

	// ------------------ Default Redirect on Success ------------------
	// If no desired position is set, redirect to /index.
	c.Redirect(http.StatusFound, "/index")
}

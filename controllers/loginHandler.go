// Package controllers handles user authentication and session management.
// File: controllers/loginHandler.go
package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

// ------------------ authentication utilities ------------------

// checkPasswordHash verifies if the provided plain-text password matches the stored hashed password.
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ------------------ login handling ------------------

// LoginHandler authenticates the user, prevents duplicate logins, and manages session storage.
// If successful, it redirects:
// - Admin users → `/admin`
// - Regular users → `/dashboard`
// If authentication fails, it returns an appropriate error message.
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c)

	// Extract login credentials from form input
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Handle missing fields before checking meet selection
	if username == "" || password == "" {
		logger.Warn.Println("LoginHandler: Missing username or password")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": "",
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// Retrieve meet name from session
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)

	// Handle missing meet selection
	if !ok || meetName == "" {
		logger.Warn.Println("LoginHandler: No meet selected, returning 400 Bad Request")
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": "",
			"Error":    "Please select a meet before logging in.",
		})
		return
	}

	// Load meet credentials
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// Validate credentials
	var valid bool
	var isAdmin bool

	for _, m := range creds.Meets {
		if m.Name == meetName {
			if m.Admin.Username == username && ComparePasswords(m.Admin.Password, password) {
				valid = true
				isAdmin = m.Admin.IsAdmin
				break
			}
		}
	}

	// Handle invalid login attempt (wrong username/password)
	if !valid {
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// Prevent duplicate logins
	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "This username is already logged in on another device.",
		})
		return
	}

	// Session management
	activeUsers[username] = true
	session.Set("user", username)
	session.Set("isAdmin", isAdmin)

	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)

	// Save session
	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// Redirect based on user role
	if isAdmin {
		c.Redirect(http.StatusFound, "/index")
	} else {
		c.Redirect(http.StatusFound, "/index")
	}
}

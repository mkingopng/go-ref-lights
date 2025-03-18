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
	meetName := session.Get("meetName")
	username := c.PostForm("username")
	password := c.PostForm("password")

	// validate input fields.
	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Please fill in all fields.",
		})
		return
	}

	// load meet credentials from JSON.
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// ------------------ user authentication ------------------

	var valid bool
	var isAdmin bool

	for _, m := range creds.Meets {
		if m.Name == meetName {
			if m.Admin.Username == username && ComparePasswords(m.Admin.Password, password) {
				valid = true
				isAdmin = m.Admin.IsAdmin
			}
			break
		}
	}

	// handle invalid login attempts.
	if !valid {
		logger.Warn.Printf("LoginHandler: Invalid login attempt for user %s at meet %s", username, meetName)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Invalid username or password.",
		})
		return
	}

	// prevent duplicate logins (only one session per user).
	if activeUsers[username] {
		logger.Warn.Printf("LoginHandler: User %s already logged in, denying second login", username)
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "This username is already logged in on another device.",
		})
		return
	}

	// ------------------ session management ------------------

	// Mark user as logged in and set session user
	activeUsers[username] = true
	session.Set("user", username)
	session.Set("isAdmin", isAdmin) // store admin status in session
	if err := session.Save(); err != nil {
		logger.Error.Println("Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"Error": "Internal error, please try again."})
		return
	}

	c.Redirect(http.StatusFound, "/index")
	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)
}

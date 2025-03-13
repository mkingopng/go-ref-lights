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

	// retrieve the selected meet from session.
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		logger.Warn.Println("LoginHandler: No meet selected, redirecting to /choose-meet")
		c.Redirect(http.StatusFound, "/choose-meet")
		return
	}

	// extract credentials from form input.
	username := c.PostForm("username")
	password := c.PostForm("password")

	// validate input fields.
	if username == "" || password == "" {
		logger.Warn.Println("LoginHandler: Missing username or password")
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

	// iterate through meets and users to validate credentials.
	for _, m := range creds.Meets {
		if m.Name == meetName {
			for _, user := range m.Users {
				if user.Username == username && ComparePasswords(user.Password, password) {
					valid = true
					isAdmin = user.IsAdmin // Capture admin status
					break
				}
			}
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

	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)

	// save session state.
	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again.",
		})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// ---------------- role based redirection ----------------

	// Redirect users based on their role
	if isAdmin {
		c.Redirect(http.StatusFound, "/admin") // admin panel
	} else {
		c.Redirect(http.StatusFound, "/dashboard") // user dashboard
	}
}

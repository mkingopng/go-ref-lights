// Package controllers handles user authentication and session management
// file: controllers/loginHandler.go
package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"golang.org/x/crypto/bcrypt"
	"net/http"
)

// compare hashed password
func checkPasswordHash(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
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

	// Load credentials from JSON
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Println("LoginHandler: Failed to load meet credentials:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again later."})
		return
	}

	var valid bool
	var isAdmin bool

	// Check if username and password match
	for _, m := range creds.Meets {
		if m.Name == meetName {
			for _, user := range m.Users {
				if user.Username == username && ComparePasswords(user.Password, password) {
					valid = true
					isAdmin = user.IsAdmin // ✅ Identify if user is an admin
					break
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
	session.Set("isAdmin", isAdmin) // ✅ Store admin status in session

	logger.Info.Printf("DEBUG: Setting isAdmin=%v for user=%s", isAdmin, username)

	if err := session.Save(); err != nil {
		logger.Error.Println("LoginHandler: Failed to save session:", err)
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{"MeetName": meetName, "Error": "Internal error, please try again."})
		return
	}

	logger.Info.Printf("LoginHandler: User %s authenticated for meet %s (isAdmin=%v)", username, meetName, isAdmin)

	// ✅ Redirect Admins to /admin, Regular Users to /dashboard
	if isAdmin {
		c.Redirect(http.StatusFound, "/admin")
	} else {
		c.Redirect(http.StatusFound, "/dashboard")
	}
}

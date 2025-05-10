// Package middleware provides request filters and security checks for the application.
// File: middleware/auth.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// -------------- authentication middleware --------------

// AuthRequired is a middleware that ensures the user is logged in.
/*How it works:
- Retrieves the session from the request context.
- Checks if the "user" session variable is set.
- If no user is found, redirects to "/choose-meet" and aborts execution.
- Otherwise, the request proceeds.*/
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	logger.Debug.Printf("[AuthRequired] Checking session for user. user=%v (type=%T), remoteAddr=%s",
		user, user, c.Request.RemoteAddr)

	// block request if user session is missing
	if user == nil {
		logger.Warn.Printf("[AuthRequired] user is nil => redirecting to /choose-meet. Possibly missing cookie.")
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort() // prevents further execution
		return
	}

	logger.Debug.Println("[AuthRequired] User is present in session - proceeding with request")
	c.Next()
}

// AdminRequired is a middleware that restricts access to admin-only routes.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isAdmin, ok := session.Get("isAdmin").(bool)

		logger.Debug.Printf("[AdminRequired] isAdmin=%v, ok=%v", isAdmin, ok)

		// block request if the user is not an admin
		if !ok || !isAdmin {
			logger.Warn.Printf("[AdminRequired] Unauthorized access attempt from %s", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
			return
		}

		logger.Debug.Println("[AdminRequired] Authorized, continuing request")
		c.Next()
	}
}

// SudoRequired ensures the user has superuser (sudo) privileges.
func SudoRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isSudo, ok := session.Get("sudo").(bool)

		if !ok || !isSudo {
			logger.Warn.Println("SudoRequired: user is not superuser; blocking access")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Superuser privileges required"})
			c.Abort()
			return
		}

		// pass through if superuser
		c.Next()
	}
}

// MeetRequired ensures a valid meetName is present in the session.
// Redirects to / if missing.
func MeetRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		meetName, ok := session.Get("meetName").(string)

		if !ok || meetName == "" {
			logger.Warn.Printf("[MeetRequired] No meetName in session — redirecting %s to /", c.ClientIP())
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		logger.Debug.Printf("[MeetRequired] meetName=%s found in session — proceeding", meetName)
		c.Next()
	}
}

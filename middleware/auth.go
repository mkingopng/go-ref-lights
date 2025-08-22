// Package middleware provides request filters and security checks for the application.
// File: middleware/auth.go
package middleware

import (
	"net/http"

	"go-ref-lights/logger"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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
	clientIP := c.ClientIP()
	user := session.Get("user")

	logger.Debug.Printf("[AuthRequired] Checking session for user. user=%v (type=%T), IP=%s",
		user, user, clientIP)

	if user == nil {
		logger.Warn.Printf("[AuthRequired] Unauthorized: user session missing or nil. Redirecting to /choose-meet. IP=%s", clientIP)
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort()
		return
	}

	logger.Info.Printf("[AuthRequired] Authorized: user session exists (%v). Proceeding with request. IP=%s", user, clientIP)
	c.Next()
}

// AdminRequired is a middleware that restricts access to admin-only routes.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		clientIP := c.ClientIP()
		isAdmin, ok := session.Get("isAdmin").(bool)

		logger.Debug.Printf("[AdminRequired] Checking admin status: isAdmin=%v, ok=%v, IP=%s", isAdmin, ok, clientIP)

		if !ok || !isAdmin {
			logger.Warn.Printf("[AdminRequired] Unauthorized access attempt. isAdmin=%v, ok=%v, IP=%s", isAdmin, ok, clientIP)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
			return
		}

		logger.Info.Printf("[AdminRequired] Admin access granted. IP=%s", clientIP)
		c.Next()
	}
}

// SudoRequired ensures the user has superuser (sudo) privileges.
func SudoRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		clientIP := c.ClientIP()
		isSudo, ok := session.Get("sudo").(bool)

		logger.Debug.Printf("[SudoRequired] Checking sudo status: isSudo=%v, ok=%v, IP=%s", isSudo, ok, clientIP)

		if !ok || !isSudo {
			logger.Warn.Printf("[SudoRequired] Access denied — not a superuser. isSudo=%v, ok=%v, IP=%s", isSudo, ok, clientIP)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Superuser privileges required"})
			c.Abort()
			return
		}

		logger.Info.Printf("[SudoRequired] Superuser access granted. IP=%s", clientIP)
		c.Next()
	}
}

// MeetRequired ensures a valid meetName is present in the session.
// Redirects to / if missing.
func MeetRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		meetName, ok := session.Get("meetName").(string)
		clientIP := c.ClientIP()

		logger.Debug.Printf("[MeetRequired] Checking meetName in session: ok=%v, meetName=%q, IP=%s", ok, meetName, clientIP)

		if !ok || meetName == "" {
			logger.Warn.Printf("[MeetRequired] Missing or empty meetName — redirecting client %s to /", clientIP)
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		logger.Info.Printf("[MeetRequired] meetName=%s found — access allowed for client %s", meetName, clientIP)
		c.Next()
	}
}

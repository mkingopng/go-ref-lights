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
// How it works:
// - Retrieves the session from the request context.
// - Checks if the "user" session variable is set.
// - If no user is found, redirects to "/choose-meet" and aborts execution.
// - Otherwise, the request proceeds.
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	// block request if user session is missing
	if user == nil {
		logger.Warn.Printf("[AuthRequired] No user found in session (user=%v). Redirecting to /choose-meet",
			session.Get("user"))
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
			logger.Warn.Println("[AdminRequired] Unauthorized attempt blocked")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort() // prevents further execution
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

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

// PositionRequired ensures that a user has the correct referee position to access specific paths.
func PositionRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")

		// if the user is not authenticated, redirect to /login
		if user == nil {
			logger.Warn.Printf("[PositionRequired] Unauthenticated access attempt to %s. Redirecting to /login",
				c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/login")
			c.Abort() // prevents further execution
			return
		}

		// retrieve the user's assigned referee position
		refPos := session.Get("refPosition")

		// determine the required position based on the request path
		path := c.Request.URL.Path
		var requiredPos string
		switch path {
		case "/left":
			requiredPos = "left"
		case "/center":
			requiredPos = "center"
		case "/right":
			requiredPos = "right"
		default:
			logger.Debug.Printf("[PositionRequired] No specific role required for path: %s", path)
		}

		// if no specific role is required, proceed
		if requiredPos == "" {
			logger.Debug.Printf("[PositionRequired] Proceeding without role restriction on path: %s", path)
			c.Next()
			return
		}

		// if user’s position does not match the required position, redirect
		if requiredPos != "" && refPos != requiredPos {
			logger.Warn.Printf("[PositionRequired] User=%v does not have the required position for %s. Expected=%s, got=%v. Redirecting to /positions",
				user, path, requiredPos, refPos)
			c.Redirect(http.StatusFound, "/positions")
			c.Abort()
			return
		}

		logger.Debug.Printf("[PositionRequired] User=%v authorized for position=%s on path=%s", user, requiredPos, path)
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

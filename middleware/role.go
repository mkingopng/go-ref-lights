// Package middleware provides request filters and access control mechanisms for the application.
// File: middleware/role.go

package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// PositionRequired ensures that a user has the correct referee position to access specific paths.
//
// Usage:
//
//	router.Use(PositionRequired())
func PositionRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")

		// If the user is not authenticated, redirect to /login
		if user == nil {
			logger.Warn.Printf("[PositionRequired] Unauthenticated access attempt to %s. Redirecting to /login",
				c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/login")
			c.Abort() // prevents further execution
			return
		}

		// Retrieve the user's assigned referee position
		refPos := session.Get("refPosition")

		// Determine the required position based on the request path
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

		// If no specific role is required, proceed
		if requiredPos == "" {
			logger.Debug.Printf("[PositionRequired] Proceeding without role restriction on path: %s", path)
			c.Next()
			return
		}

		// If user’s position does not match the required position, redirect
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

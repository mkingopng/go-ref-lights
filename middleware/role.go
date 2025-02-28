// Package middleware file: middleware/role.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// PositionRequired is a middleware that checks if the user has the required position to access the path.
func PositionRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			logger.Warn.Printf("Unauthenticated access attempt to %s. Redirecting to /login", c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		refPos := session.Get("refPosition")
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
			logger.Debug.Printf("No specific role required for path: %s", path)
		}

		// If there's a mismatch between expected and actual position, log a warning and redirect.
		if requiredPos != "" && refPos != requiredPos {
			logger.Warn.Printf("User %v does not have the required position for %s. Expected: %s, got: %v. Redirecting to /positions", user, path, requiredPos, refPos)
			c.Redirect(http.StatusFound, "/positions")
			c.Abort()
			return
		}

		logger.Debug.Printf("User %v authorized for position %s on path %s", user, requiredPos, path)
		c.Next()
	}
}

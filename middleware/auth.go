// Package middleware middleware/auth.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// AuthRequired is a middleware that checks if the user is authenticated
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		logger.Warn.Printf("Unauthenticated access attempt to %s. Redirecting to /meets", c.Request.URL.Path)
		c.Redirect(http.StatusFound, "/meets")
		c.Abort()
		return
	}
	c.Next()
}

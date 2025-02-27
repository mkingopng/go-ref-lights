// Package middleware middleware/auth.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// AuthRequired middleware to ensure user is logged in
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName := session.Get("meetName")

	if user == nil || meetName == nil {
		logger.Warn.Printf("Unauthorized access attempt to %s. Redirecting to /choose-meet", c.Request.URL.Path)
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort()
		return
	}

	c.Next()
}

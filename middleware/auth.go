// Package middleware middleware/auth.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// AuthRequired middleware to ensure user is logged in
// ✅ AuthRequired middleware to ensure user is logged in
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	// ✅ Debugging log
	if user == nil {
		logger.Warn.Printf("AuthRequired: No user found in session. Raw session data: %+v", session.Get("user"))
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort()
		return
	}

	c.Next()
}

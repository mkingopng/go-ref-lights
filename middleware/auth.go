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
// Usage:
//
//	router.Use(AuthRequired)
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	// block request if user session is missing
	if user == nil {
		logger.Warn.Printf("AuthRequired: No user found in session. Raw session data: %+v", session.Get("user"))
		c.Redirect(http.StatusFound, "/choose-meet")
		c.Abort() // 🔴 prevents further execution
		return
	}

	logger.Debug.Println("[AuthRequired] User authenticated - proceeding with request")
	c.Next()
}

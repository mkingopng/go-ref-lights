// Package middleware provides request filters and security checks for the application.
// File: middleware/admin_required.go

package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

// AdminRequired is a middleware that restricts access to admin-only routes.
// Usage:
//
//	router.Use(AdminRequired())
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isAdmin, ok := session.Get("isAdmin").(bool)

		logger.Debug.Printf("[AdminRequired] isAdmin=%v, ok=%v", isAdmin, ok)

		// Block request if user is not an admin
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

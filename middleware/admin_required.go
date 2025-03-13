// Package middleware provides request filters and security checks for the application.
// File: middleware/admin_required.go
package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"net/http"
)

// ------------------ admin authorisation middleware -------------------

// AdminRequired is a middleware that restricts access to admin-only routes.
// How it works:
// - Retrieves the session from the request context.
// - Checks if the session contains "isAdmin" and if the value is `true`.
// - If the user is not an admin, returns HTTP 401 Unauthorized and stops request execution.
// - Otherwise, the request proceeds.
//
// Usage:
//
//	router.Use(AdminRequired())
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isAdmin, ok := session.Get("isAdmin").(bool)

		logger.Debug.Printf("AdminRequired Middleware - isAdmin=%v, ok=%v", isAdmin, ok)

		// Block request if user is not an admin
		if !ok || !isAdmin {
			logger.Warn.Println("AdminRequired Middleware - Unauthorized attempt blocked")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort() // 🔴 Prevents further execution
			return
		}

		logger.Debug.Println("AdminRequired Middleware - Passed, continuing request")
		c.Next()
	}
}

// Package middleware description is Middleware that checks if the user is an admin.
// file: middleware/admin_required.go
package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"net/http"
)

// AdminRequired is a middleware that checks if the user is an admin.
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		isAdmin, ok := session.Get("isAdmin").(bool)

		logger.Debug.Printf("AdminRequired Middleware - isAdmin=%v, ok=%v", isAdmin, ok)

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

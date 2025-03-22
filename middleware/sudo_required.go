// File: middleware/sudo_required.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
)

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

		// Pass through if superuser
		c.Next()
	}
}

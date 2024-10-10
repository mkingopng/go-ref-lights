// middleware/auth.go
package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired is a middleware that checks if the user is authenticated
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		// User not authenticated
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
	} else {
		// User is authenticated
		c.Next()
	}
}

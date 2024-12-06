// middleware/auth.go
package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
)

// AuthRequired is a middleware that checks if the user is authenticated
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()
		return
	}
	c.Next()
}

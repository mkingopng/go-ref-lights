// Package middleware file: middleware/role.go
package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/http"
)

func PositionRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		refPos := session.Get("refPosition")

		// Map the request path to the expected position
		path := c.Request.URL.Path
		var requiredPos string
		switch path {
		case "/left":
			requiredPos = "left"
		case "/centre":
			requiredPos = "centre"
		case "/right":
			requiredPos = "right"
		}

		// If mismatch, redirect them or show an error
		if requiredPos != "" && refPos != requiredPos {
			c.Redirect(http.StatusFound, "/positions")
			c.Abort()
			return
		}

		c.Next()
	}
}

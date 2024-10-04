// controllers/page_controller.go
package controllers

import (
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// ShowLoginPage renders the login page
func ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// PerformLogin handles the login form submission
func PerformLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Simple in-memory authentication
	if username == "user" && password == "zerow" {
		session := sessions.Default(c)
		session.Set("user", username)
		session.Save()
		c.Redirect(http.StatusFound, "/")
	} else {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{
			"Error": "Invalid credentials",
		})
	}
}

// Index renders the index page
func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

// Left renders the left page
func Left(c *gin.Context) {
	data := gin.H{
		"WebsocketURL": "ws://localhost:8080/referee-updates",
	}
	c.HTML(http.StatusOK, "left.html", data)
}

// Centre renders the centre page
func Centre(c *gin.Context) {
	data := gin.H{
		"WebsocketURL": "ws://localhost:8080/referee-updates",
	}
	c.HTML(http.StatusOK, "centre.html", data)
}

// Right renders the right page
func Right(c *gin.Context) {
	data := gin.H{
		"WebsocketURL": "ws://localhost:8080/referee-updates",
	}
	c.HTML(http.StatusOK, "right.html", data)
}

// Lights renders the lights page
func Lights(c *gin.Context) {
	c.HTML(http.StatusOK, "lights.html", nil)
}

// GetQRCode generates and serves the QR code
func GetQRCode(c *gin.Context) {
	png, err := services.GenerateQRCode(250, 250)
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not generate QR code")
		return
	}
	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	c.Writer.Write(png)
}

// RefereeUpdates handles WebSocket connections
func RefereeUpdates(c *gin.Context) {
	websocket.ServeWs(c.Writer, c.Request)
}

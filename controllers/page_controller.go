// Package controllers file: controllers/page_controller.go
package controllers

import (
	"github.com/skip2/go-qrcode"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

var (
	ApplicationURL string
	WebsocketURL   string
)

// ShowPositionsPage displays the referee position selection page
func ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		logger.Warn.Println("ShowPositionsPage: User not logged in; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	svc := services.OccupancyService{}
	occ := svc.GetOccupancy()
	logger.Debug.Printf("ShowPositionsPage: Occupancy state: %+v", occ)

	data := gin.H{
		"ApplicationURL": ApplicationURL,
		"WebsocketURL":   WebsocketURL,
		"Positions": map[string]interface{}{
			"LeftOccupied":   occ.LeftUser != "",
			"LeftUser":       occ.LeftUser,
			"CentreOccupied": occ.CentreUser != "",
			"CentreUser":     occ.CentreUser,
			"RightOccupied":  occ.RightUser != "",
			"RightUser":      occ.RightUser,
		},
	}

	logger.Info.Println("ShowPositionsPage: Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition handles position assignment
func ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		logger.Warn.Println("ClaimPosition: User not logged in; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position := c.PostForm("position")
	userEmail := user.(string)
	svc := &services.OccupancyService{} // Use a pointer to avoid method call issue

	logger.Info.Printf("ClaimPosition: User %s attempting to claim position %s", userEmail, position)
	err := svc.SetPosition(position, userEmail)
	if err != nil {
		logger.Error.Printf("ClaimPosition: Failed to set position %s for user %s: %v", position, userEmail, err)
		c.String(http.StatusForbidden, "Error: %s", err.Error())
		return
	}

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("ClaimPosition: Error saving session for user %s: %v", userEmail, err)
	}

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s", userEmail, position)
	// Redirect to assigned position view
	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "centre":
		c.Redirect(http.StatusFound, "/centre")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		c.Redirect(http.StatusFound, "/positions")
	}
}

// SetConfig updates global configuration
func SetConfig(appURL, wsURL string) {
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("SetConfig: Global config updated: ApplicationURL=%s, WebsocketURL=%s", appURL, wsURL)
}

// PerformLogin redirects to Google OAuth login
func PerformLogin(c *gin.Context) {
	logger.Info.Println("PerformLogin: Redirecting to Google OAuth login")
	c.Redirect(http.StatusFound, "/auth/google/login")
}

// Logout handles user logout
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	userEmail := session.Get("user")
	refPosition := session.Get("refPosition")

	if userEmail != nil && refPosition != nil {
		logger.Info.Printf("Logout: Logging out user %s from position %s", userEmail, refPosition)
		services.UnsetPosition(refPosition.(string), userEmail.(string))
	}

	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("Logout: Error saving session during logout: %v", err)
	} else {
		logger.Info.Println("Logout: Session cleared successfully")
	}

	c.Redirect(http.StatusFound, "/login")
}

// Index renders the index page
func Index(c *gin.Context) {
	logger.Info.Println("Index: Rendering index page")
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "index.html", data)
}

// Left renders the left referee view
func Left(c *gin.Context) {
	logger.Info.Println("Left: Rendering left referee view")
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "left.html", data)
}

// Centre renders the centre referee view
func Centre(c *gin.Context) {
	logger.Info.Println("Centre: Rendering centre referee view")
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "centre.html", data)
}

// Right renders the right referee view
func Right(c *gin.Context) {
	logger.Info.Println("Right: Rendering right referee view")
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "right.html", data)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	logger.Info.Println("Lights: Rendering lights page")
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "lights.html", data)
}

// GetQRCode generates and serves the QR code
func GetQRCode(c *gin.Context) {
	logger.Info.Println("GetQRCode: Generating QR code")
	png, err := services.GenerateQRCode(250, 250, qrcode.Encode) // Pass the actual encoder function
	if err != nil {
		logger.Error.Printf("GetQRCode: Could not generate QR code: %v", err)
		c.String(http.StatusInternalServerError, "Could not generate QR code")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	if _, err = c.Writer.Write(png); err != nil {
		logger.Error.Printf("GetQRCode: Error writing QR code: %v", err)
	}
}

// RefereeUpdates handles WebSocket connections for live referee updates
func RefereeUpdates(c *gin.Context) {
	logger.Info.Println("RefereeUpdates: Establishing WebSocket connection for referee updates")
	websocket.ServeWs(c.Writer, c.Request)
}

// Health returns OK for health checks
func Health(c *gin.Context) {
	logger.Info.Println("Health: Health check requested")
	c.String(http.StatusOK, "OK")
}

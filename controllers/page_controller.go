// Package controllers file: controllers/page_controller.go
package controllers

import (
	"github.com/skip2/go-qrcode"
	"log"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
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

	// Make sure user is logged in
	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Make sure a meet is chosen
	meetName := session.Get("selectedMeet")
	if meetName == nil {
		c.Redirect(http.StatusFound, "/select-meet")
		return
	}

	// Retrieve occupancy and render positions
	svc := services.OccupancyService{}
	occ := svc.GetOccupancy()
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
		// Optionally display which meet is selected in the UI
		"SelectedMeet": meetName,
	}

	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition handles position assignment
func ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position := c.PostForm("position")
	userEmail := user.(string)
	svc := &services.OccupancyService{} // Use a pointer to avoid method call issue

	err := svc.SetPosition(position, userEmail)
	if err != nil {
		c.String(http.StatusForbidden, "Error: %s", err.Error())
		return
	}

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		log.Printf("Error saving session: %v", err)
	}

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
}

// PerformLogin redirects to Google OAuth login
func PerformLogin(c *gin.Context) {
	c.Redirect(http.StatusFound, "/auth/google/login")
}

// Logout handles user logout
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	userEmail := session.Get("user")
	refPosition := session.Get("refPosition")

	// Free up the referee position
	if userEmail != nil && refPosition != nil {
		services.UnsetPosition(refPosition.(string), userEmail.(string))
	}

	// Clear the session
	session.Clear()
	if err := session.Save(); err != nil {
		log.Printf("Error saving session: %v", err)
	}

	c.Redirect(http.StatusFound, "/login")
}

// Index renders the index page
func Index(c *gin.Context) {
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "index.html", data)
}

// Left renders the left referee view
func Left(c *gin.Context) {
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "left.html", data)
}

// Centre renders the centre referee view
func Centre(c *gin.Context) {
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "centre.html", data)
}

// Right renders the right referee view
func Right(c *gin.Context) {
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "right.html", data)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	data := gin.H{"WebsocketURL": WebsocketURL}
	c.HTML(http.StatusOK, "lights.html", data)
}

// GetQRCode generates and serves the QR code
func GetQRCode(c *gin.Context) {
	png, err := services.GenerateQRCode(250, 250, qrcode.Encode) // ✅ Pass the actual encoder function
	if err != nil {
		c.String(http.StatusInternalServerError, "Could not generate QR code")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	if _, err = c.Writer.Write(png); err != nil {
		log.Printf("Error writing QR code: %v", err)
	}
}

// RefereeUpdates handles WebSocket connections
func RefereeUpdates(c *gin.Context) {
	session := sessions.Default(c)
	meetNameVal := session.Get("selectedMeet")
	if meetNameVal == nil {
		meetNameVal = "DEFAULT_MEET"
	}
	// Append ?meetName=whatever
	c.Request.URL.RawQuery = "meetName=" + meetNameVal.(string)
	websocket.ServeWs(c.Writer, c.Request)
}

// Health returns OK for health checks
func Health(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

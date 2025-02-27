// Package controllers file: controllers/page_controller.go
package controllers

import (
	"github.com/skip2/go-qrcode"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
)

var (
	ApplicationURL string
	WebsocketURL   string
)

// SetMeet sets the selected meetId in the session.
func SetMeet(c *gin.Context) {
	meetId := c.PostForm("meetId")
	if meetId == "" {
		// If no meet is selected, redirect back to the selection page.
		c.Redirect(http.StatusFound, "/")
		return
	}
	session := sessions.Default(c)
	session.Set("meetId", meetId)
	if err := session.Save(); err != nil {
		logger.Error.Printf("SetMeet: Failed to save meetId: %v", err)
	}
	logger.Info.Printf("SetMeet: Stored meetId %s in session", meetId)
	// Redirect to login (or the next step in your flow)
	c.Redirect(http.StatusFound, "/login")
}

// ChooseMeet renders the meet selection page.
// ChooseMeet renders the meet selection page.
func ChooseMeet(c *gin.Context) {
	data, err := LoadMeets()
	if err != nil {
		// Log the error and show a message to the user
		logger.Error.Printf("ChooseMeet: Failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}
	// Pass the available meets to the template.
	c.HTML(http.StatusOK, "choose_meet.html", gin.H{
		"availableMeets": data.Meets,
	})
}

// ShowPositionsPage displays the referee position selection page.
func ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetId, ok := session.Get("meetId").(string)
	if user == nil || !ok || meetId == "" {
		logger.Warn.Println("ShowPositionsPage: User not logged in or no meet selected; redirecting to /login or /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}

	svc := services.OccupancyService{}
	occ := svc.GetOccupancy(meetId)
	logger.Debug.Printf("ShowPositionsPage: Occupancy state for meet %s: %+v", meetId, occ)

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
		"meetId": meetId,
	}

	logger.Info.Println("ShowPositionsPage: Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition handles position assignment.
func ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetId, ok := session.Get("meetId").(string)
	if user == nil || !ok || meetId == "" {
		logger.Warn.Println("ClaimPosition: User not logged in or no meet selected; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Note: Fixed typo "postion" -> "position"
	position := c.PostForm("position")
	userEmail := user.(string)
	svc := &services.OccupancyService{}

	err := svc.SetPosition(meetId, position, userEmail)
	if err != nil {
		logger.Error.Printf("ClaimPosition: Failed to set position %s for user %s in meet %s: %v", position, userEmail, meetId, err)
		c.String(http.StatusForbidden, "Error: %s", err.Error())
		return
	}

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("ClaimPosition: Error saving session for user %s: %v", userEmail, err)
		// Optionally handle the error
	}

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s for meet %s", userEmail, position, meetId)
	// Redirect based on the claimed position.
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

// ShowLoginPage redirects users to Google OAuth login
func ShowLoginPage(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("meetId") == nil {
		c.Redirect(http.StatusFound, "/") // Redirect to choose_meet page
		return
	}
	// Capture meetId from the query string (e.g., /login?meetId=meet1)
	meetId := c.Query("meetId")
	if meetId != "" {
		session := sessions.Default(c)
		session.Set("meetId", meetId)
		if err := session.Save(); err != nil { // Ensure session is saved here
			logger.Error.Printf("❌ Failed to save session: %v", err)
		} else {
			logger.Info.Printf("Stored meetId %s in session", meetId)
		}
	}
	logger.Info.Println("Redirecting to Google OAuth login page (ShowLoginPage)")
	c.Redirect(http.StatusFound, "/auth/google/login")
}

// Index renders the index page.
func Index(c *gin.Context) {
	session := sessions.Default(c)
	meetId, ok := session.Get("meetId").(string)
	if !ok || meetId == "" {
		logger.Warn.Println("Index: No meet selected; redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Printf("Rendering index page for meet %s", meetId)
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetId":       meetId,
	}
	c.HTML(http.StatusOK, "index.html", data)
}

// Left renders the left referee view
func Left(c *gin.Context) {
	session := sessions.Default(c)
	meetId, ok := session.Get("meetId").(string)
	if !ok || meetId == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Left: Rendering left referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetId":       meetId,
	}
	c.HTML(http.StatusOK, "left.html", data)
}

// Centre renders the centre referee view
func Centre(c *gin.Context) {
	session := sessions.Default(c)
	meetId, ok := session.Get("meetId").(string)
	if !ok || meetId == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Centre: Rendering centre referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetId":       meetId,
	}
	c.HTML(http.StatusOK, "centre.html", data)
}

// Right renders the right referee view
func Right(c *gin.Context) {
	session := sessions.Default(c)
	meetId, ok := session.Get("meetId").(string)
	if !ok || meetId == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Right: Rendering right referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetId":       meetId,
	}
	c.HTML(http.StatusOK, "right.html", data)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	session := sessions.Default(c)
	meetId, ok := session.Get("meetId").(string)
	if !ok || meetId == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Lights: Rendering lights page")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetId":       meetId,
	}
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
// todo: need to add feature to allow referees to change position
//func RefereeUpdates(c *gin.Context) {
//	logger.Info.Println("RefereeUpdates: Establishing WebSocket connection for referee updates")
//	websocket.ServeWs(c.Writer, c.Request)
//}

// Health returns OK for health checks
func Health(c *gin.Context) {
	logger.Info.Println("Health: Health check requested")
	c.String(http.StatusOK, "OK")
}

// SetConfig and Logout remain unchanged.
func SetConfig(appURL, wsURL string) {
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("SetConfig: Global config updated: ApplicationURL=%s, WebsocketURL=%s", appURL, wsURL)
}

func PerformLogin(c *gin.Context) {
	logger.Info.Println("PerformLogin: Redirecting to Google OAuth login")
	c.Redirect(http.StatusFound, "/auth/google/login")
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	userEmail := session.Get("user")
	refPosition := session.Get("refPosition")
	meetId, ok := session.Get("meetId").(string)

	if userEmail != nil && refPosition != nil && ok && meetId != "" {
		logger.Info.Printf("Logout: Logging out user %s from position %s", userEmail, refPosition)
		services.UnsetPosition(meetId, refPosition.(string), userEmail.(string))
	}

	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("Logout: Error saving session during logout: %v", err)
	} else {
		logger.Info.Println("Logout: Session cleared successfully")
	}

	c.Redirect(http.StatusFound, "/login")
}

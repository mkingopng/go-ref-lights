// Package controllers handles various page rendering and session management functions.
// File: controllers/page_controller.go
package controllers

import (
	"fmt"
	"go-ref-lights/models"
	"net/http"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
	"go-ref-lights/services"
)

// -------------------- global configuration --------------------

var anonOccupantCounter int
var anonCounterMu sync.Mutex

var (
	// ApplicationURL is the base URL of the application
	ApplicationURL string

	// WebsocketURL is the URL for the WebSocket server
	WebsocketURL string
)

// -------------------- active users --------------------

// getNextAnonymousName increments and returns a new occupant name,
// e.g. "AnonRef001", "AnonRef002", etc.
func getNextAnonymousName() string {
	anonCounterMu.Lock()
	defer anonCounterMu.Unlock()

	anonOccupantCounter++
	return fmt.Sprintf("AnonRef%03d", anonOccupantCounter)
}

// -------------------- health check endpoint --------------------

// Health provides a simple endpoint to check server health.
func Health(c *gin.Context) {
	logger.Info.Println("Health: Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// -------------------- user navigation and logout --------------------

// Home redirects the user to the dashboard and vacates their referee position.
func Home(c *gin.Context, occupancyService *services.OccupancyService) {
	session := sessions.Default(c)

	userEmail, ok1 := session.Get("user").(string)
	position, ok2 := session.Get("refPosition").(string)
	meetName, ok3 := session.Get("meetName").(string)

	if ok1 && ok2 && ok3 {
		if err := occupancyService.UnsetPosition(meetName, position, userEmail); err != nil {
			logger.Error.Printf("Home: error vacating position: %v", err)
		} else {
			logger.Info.Printf("Home: position '%s' vacated for user '%s' in meet '%s'", position, userEmail, meetName)
			session.Delete("refPosition")
			err := session.Save()
			if err != nil {
				return
			}
		}
	} else {
		logger.Warn.Println("Home: Missing user, refPosition or meetName in session.")
	}
	c.Redirect(http.StatusFound, "/choose-meet")
}

// Logout logs the user out, removes them from activeUsers, vacates their
// position, and redirects to login.
func Logout(c *gin.Context, occupancyService services.OccupancyServiceInterface) {
	session := sessions.Default(c)

	userEmail, hasUser := session.Get("user").(string)
	position, hasPosition := session.Get("refPosition").(string)
	meetName, hasMeet := session.Get("meetName").(string)

	isAdmin, _ := session.Get("isAdmin").(bool)
	if isAdmin && hasMeet {
		logger.Info.Printf("Logout: Admin user is logging out; resetting meet: %s", meetName)
		occupancyService.ResetOccupancyForMeet(meetName)
	}

	if hasUser && hasPosition && hasMeet {
		err := occupancyService.UnsetPosition(meetName, position, userEmail)
		if err != nil {
			logger.Error.Printf("Logout: error vacating position: %v", err)
		} else {
			logger.Info.Printf("Logout: position '%s' vacated for user '%s' in meet '%s'",
				position, userEmail, meetName)
		}

		activeUsersMu.Lock()
		delete(activeUsers, userEmail)
		activeUsersMu.Unlock()

		logger.Info.Printf("Logout: User %s removed from active users list", userEmail)
	} else {
		logger.Warn.Println("Logout: Missing user, refPosition, or meetName from session.")
	}

	session.Clear()
	logger.Info.Println("Logout: Session cleared (will be saved by middleware at end of request)")
	c.Redirect(http.StatusFound, "/index")
}

// -------------------- page rendering --------------------

// Index renders the main application page
func Index(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/set-meet")
		return
	}

	// load all meets from memory or your loaded creds
	creds, err := loadMeetCredsFunc()
	if err != nil {
		logger.Error.Printf("Index: failed to load meet creds: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meet credentials")
		return
	}

	// find the current meet
	var currentMeet *models.Meet
	for _, m := range creds.Meets {
		if m.Name == meetName {
			currentMeet = &m
			break
		}
	}

	if currentMeet == nil {
		logger.Warn.Printf("Meet not found: %s", meetName)
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	// pass the meet’s Logo to the template
	data := gin.H{
		"meetName":     meetName,
		"WebsocketURL": WebsocketURL,     // if you have that
		"Logo":         currentMeet.Logo, // <--- fix_me: potential nil pointer dereference
	}
	c.HTML(http.StatusOK, "index.html", data)
}

// ShowPositionsPage renders the positions selection page.
func ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName, ok := session.Get("meetName").(string)
	if user == nil || !ok || meetName == "" {
		logger.Warn.Println("ShowPositionsPage: User not logged in or no meet selected; redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}

	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
		"Positions": map[string]interface{}{
			"LeftOccupied":   false, // todo: Example data, replace with actual occupancy logic
			"LeftUser":       "",
			"centerOccupied": false,
			"centerUser":     "",
			"RightOccupied":  false,
			"RightUser":      "",
		},
	}
	logger.Info.Println("ShowPositionsPage: Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// GetQRCode generates and returns a QR code for the application URL.
func GetQRCode(c *gin.Context) {
	logger.Info.Println("GetQRCode: Generating QR code")

	meetName := c.Query("meetName")
	position := c.Query("position")
	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing meetName or position query param")
		return
	}

	qrURL := fmt.Sprintf("%s/referee/%s/%s", ApplicationURL, meetName, position)

	qrBytes, err := services.GenerateQRCode(qrURL, 300, qrcode.Medium)
	if err != nil {
		logger.Error.Printf("GetQRCode: Error generating QR code: %v", err)
		c.String(http.StatusInternalServerError, "QR generation failed")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	if _, err := c.Writer.Write(qrBytes); err != nil {
		logger.Error.Printf("GetQRCode: Error writing QR code bytes: %v", err)
	}
}

// SetConfig updates the global application and WebSocket URLs.
func SetConfig(appURL, wsURL string) {
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("SetConfig: Global config updated: ApplicationURL=%s, WebsocketURL=%s", appURL, wsURL)
}

// -------------------- referee view rendering --------------------

// Left renders the left referee view
func Left(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	refPosition := session.Get("refPosition")
	logger.Debug.Printf("Left handler: Session meetName='%s', refPosition='%v'", meetName, refPosition)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Left: Rendering left referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL, // WebsocketURL is now declared globally
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "left.html", data)
}

// Center renders the center referee view
func Center(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	refPosition := session.Get("refPosition")
	logger.Debug.Printf("Center handler: Session meetName='%s', refPosition='%v'", meetName, refPosition)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("center: Rendering center referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "center.html", data)
}

// Right renders the right referee view
func Right(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	refPosition := session.Get("refPosition")
	logger.Debug.Printf("Right handler: Session meetName='%s', refPosition='%v'", meetName, refPosition)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Right: Rendering right referee view")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "right.html", data)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("Lights: Rendering lights page")
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "lights.html", data)
}

// RefereeHandler renders the referee view based on the position parameter.
func RefereeHandler(c *gin.Context, occupancyService services.OccupancyServiceInterface) {
	meetName := c.Param("meetName")
	position := c.Param("position")

	// 1) Get or create a unique occupant for this session
	session := sessions.Default(c)
	occupant, ok := session.Get("anonymousOccupant").(string)
	if !ok || occupant == "" {
		occupant = getNextAnonymousName()
		session.Set("anonymousOccupant", occupant)
		if err := session.Save(); err != nil {
			logger.Error.Printf("RefereeHandler: session save error: %v", err)
		}
	}

	// 2) Attempt to claim seat under occupant name
	err := occupancyService.SetPosition(meetName, position, occupant)
	if err != nil {
		logger.Warn.Printf("RefereeHandler: Attempt to claim taken seat %s for meet %s by occupant=%s",
			position, meetName, occupant)
		// Return 409 Conflict or some suitable error
		c.String(http.StatusConflict, "This referee seat (%s) is already taken.", position)
		return
	}

	logger.Info.Printf("RefereeHandler: meetName=%s, position=%s claimed successfully by occupant=%s",
		meetName, position, occupant)

	// 3) Render the appropriate referee view
	switch position {
	case "left", "Left":
		renderLeft(c, meetName)
	case "center", "Center":
		renderCenter(c, meetName)
	case "right", "Right":
		renderRight(c, meetName)
	default:
		c.String(http.StatusBadRequest, "Unknown position: %s", position)
	}
}

// renderCenter renders the center referee page
func renderCenter(c *gin.Context, meetName string) {
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "center.html", data)
}

// renderRight renders the right referee page
func renderRight(c *gin.Context, meetName string) {
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "right.html", data)
}

// renderLeft renders the left referee page
func renderLeft(c *gin.Context, meetName string) {
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "left.html", data)
}

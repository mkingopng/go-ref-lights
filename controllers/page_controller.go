// Package controllers handles various page rendering and session management functions.
// File: controllers/page_controller.go
package controllers

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"

	"go-ref-lights/logger"
	"go-ref-lights/models"
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
func getNextAnonymousName() string {
	anonCounterMu.Lock()
	defer anonCounterMu.Unlock()

	anonOccupantCounter++
	return fmt.Sprintf("AnonRef%03d", anonOccupantCounter)
}

// -------------------- health check endpoint --------------------

// Health provides a simple endpoint to check server health.
func Health(c *gin.Context) {
	logger.Info.Println("[Health] Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// -------------------- user navigation and logout --------------------

// Logout logs the user out, removes them from ActiveUsers, vacates their
// position, and redirects to login.
func Logout(c *gin.Context, occupancyService services.OccupancyServiceInterface) {

	// session data
	session := sessions.Default(c)
	userEmail, _ := session.Get("user").(string)
	position, _ := session.Get("refPosition").(string)
	meetName, _ := session.Get("meetName").(string)
	isAdmin, _ := session.Get("isAdmin").(bool)

	if userEmail == "" {
		// no user => no occupant removal
		logger.Warn.Println("[Logout] No userEmail in session, skipping occupant removal.")
	} else if isAdmin {
		// Admin logic
		logger.Info.Printf("[Logout] Admin (%s) logging out of meet: %s", userEmail, meetName)
		if meetName != "" {
			occupancyService.ResetOccupancyForMeet(meetName)
		}
		ActiveUsersMu.Lock()
		delete(ActiveUsers, userEmail)
		ActiveUsersMu.Unlock()
		logger.Info.Printf("[Logout] Admin user %s removed from active users list", userEmail)
	} else {
		// Referee logic
		logger.Info.Printf("[Logout] Referee user=%s is logging out for meet=%s", userEmail, meetName)
		if position != "" && meetName != "" {
			if err := occupancyService.UnsetPosition(meetName, position, userEmail); err != nil {
				logger.Error.Printf("[Logout] Error vacating seat for user=%s: %v", userEmail, err)
			} else {
				logger.Info.Printf("[Logout] Freed seat=%s for user=%s in meet=%s", position, userEmail, meetName)
			}
		}
		ActiveUsersMu.Lock()
		delete(ActiveUsers, userEmail)
		ActiveUsersMu.Unlock()
		logger.Info.Printf("[Logout] User %s removed from active users list", userEmail)
	}

	// clear session
	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("[Logout] Error saving session after clearing: %v", err)
	}

	// final redirect
	if isAdmin {
		// meet director => redirect to /login
		logger.Info.Println("[Logout] Admin user. Redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
	} else {
		// referee => redirect to /logged-out (or your new page)
		logger.Info.Println("[Logout] Referee user. Redirecting to /logged-out")
		c.Redirect(http.StatusFound, "/logged-out")
	}
}

// -------------------- page rendering --------------------

// Index renders the main dashboard page screen after logging in
func Index(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	isSudo, _ := session.Get("sudo").(bool)

	// if the user didn't pick any meetName and is not superuser, redirect them
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/set-meet")
		return
	}

	// if they selected "Sudo" as their meetName, skip normal meet logic and go to /sudo
	if meetName == "Sudo" {
		c.Redirect(http.StatusFound, "/sudo")
		return
	}

	// normal meet logic:
	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Printf("[Index] Failed to load meet creds: %v", creds)
		c.String(http.StatusInternalServerError, "Failed to load meet credentials")
		return
	}

	// find the current meet
	var currentMeet *models.Meet
	for _, m := range creds.Meets {
		if m.Name == meetName {
			mCopy := m
			currentMeet = &mCopy
			break
		}
	}
	if currentMeet == nil {
		logger.Warn.Printf("[Index] Meet not found: %s", meetName)
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	data := gin.H{
		"meetName": meetName,
		"IsSudo":   isSudo,
		"Logo":     currentMeet.Logo,
	}

	c.HTML(http.StatusOK, "index.html", data)
}

// GetQRCode generates and returns a QR code for the application URL.
func GetQRCode(c *gin.Context) {
	logger.Info.Println("[GetQRCode] Generating QR code")

	meetName := c.Query("meetName")
	position := c.Query("position")
	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing meetName or position query param")
		return
	}

	qrURL := fmt.Sprintf("%s/referee/%s/%s", ApplicationURL, meetName, position)

	qrBytes, err := services.GenerateQRCode(qrURL, 300, qrcode.Medium)
	if err != nil {
		logger.Error.Printf("[GetQRCode] Error generating QR code: %v", err)
		c.String(http.StatusInternalServerError, "QR generation failed")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	if _, err := c.Writer.Write(qrBytes); err != nil {
		logger.Error.Printf("[GetQRCode] Error writing QR code bytes: %v", err)
	}
}

// SetConfig updates the global application and WebSocket URLs.
func SetConfig(appURL, wsURL string) {
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("[SetConfig] Global config updated: ApplicationURL=%s, WebsocketURL=%s", appURL, wsURL)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	if !ok || meetName == "" {
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	logger.Info.Println("[Lights] Rendering lights page")

	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Warn.Println("[LoginHandler] No global credentials available")
		c.HTML(http.StatusInternalServerError, "login.html", gin.H{
			"MeetName": meetName,
			"Error":    "Internal error, please try again later.",
		})
		return
	}

	// find the current meet
	var currentMeet *models.Meet

	for _, m := range creds.Meets {
		if m.Name == meetName {
			mCopy := m
			currentMeet = &mCopy
			break
		}
	}
	if currentMeet == nil {
		logger.Warn.Printf("[Lights] Meet not found: %s", meetName)
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
		"Logo":         currentMeet.Logo,
	}
	c.HTML(http.StatusOK, "lights.html", data)
}

// RefereeHandler renders the referee view based on the position parameter.
func RefereeHandler(c *gin.Context, occupancyService services.OccupancyServiceInterface) {
	meetNameParam := c.Param("meetName")
	position := c.Param("position")

	// retrieve session data
	session := sessions.Default(c)
	occupant, occupantExists := session.Get("user").(string)
	storedMeet, storedMeetExists := session.Get("meetName").(string)

	// if the session already has a meetName but it differs from the route param, handle it
	if storedMeetExists && storedMeet != "" && storedMeet != meetNameParam {
		logger.Warn.Printf("[RefereeHandler] Session meetName=%s but request meetName=%s. Aborting seat claim.",
			storedMeet, meetNameParam)

		c.String(http.StatusConflict,
			"You already have meet=%s in session, but tried to claim a seat in meet=%s. Please re-select a meet.",
			storedMeet, meetNameParam)
		return
	}

	// if occupant doesn’t exist, create a new occupant name
	if !occupantExists || occupant == "" {
		occupant = getNextAnonymousName()
	}

	// if the session had no meet set, or it was empty, store the new one
	if !storedMeetExists || storedMeet == "" {
		session.Set("meetName", meetNameParam)
	}

	// attempt to claim the seat
	if err := occupancyService.SetPosition(meetNameParam, position, occupant); err != nil {
		logger.Warn.Printf("[RefereeHandler] Attempt to claim seat=%s for occupant=%s in meet=%s failed: %v",
			position, occupant, meetNameParam, err)
		c.String(http.StatusConflict, "This referee seat (%s) is already taken.", position)
		return
	}

	// store occupant + position in session
	session.Set("user", occupant)
	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("[RefereeHandler] Failed to save session occupant=%s: %v", occupant, err)
	}

	// 7) Render the correct referee view
	logger.Info.Printf("[RefereeHandler] meetName=%s, position=%s claimed successfully by occupant=%s",
		meetNameParam, position, occupant)

	switch position {
	case "left", "Left":
		renderLeft(c, meetNameParam)
	case "center", "Center":
		renderCenter(c, meetNameParam)
	case "right", "Right":
		renderRight(c, meetNameParam)
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

// -------------- meet selection handling --------------

// ShowMeets renders the meet selection page.
func ShowMeets(c *gin.Context) {
	// retrieve meet data using a mockable function for easier testing
	meetsData, err := services.LoadBasicMeets()
	if err != nil {
		logger.Error.Printf("[ShowMeets] Failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	// render the meet selection page with available meets
	c.HTML(http.StatusOK, "choose-meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

func ChooseMeetHandler(c *gin.Context) {
	// This is a handler for GET /choose-meet. It loads basic meets from disk,
	// then renders the template
	meetsData, err := services.LoadBasicMeets()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	c.HTML(http.StatusOK, "choose-meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

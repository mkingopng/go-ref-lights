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
// useful for assigning anonymous referees when no email is present.
func getNextAnonymousName() string {
	anonCounterMu.Lock()
	defer anonCounterMu.Unlock()
	anonOccupantCounter++
	name := fmt.Sprintf("AnonRef%03d", anonOccupantCounter)
	logger.Debug.Printf("[getNextAnonymousName] Generated anonymous name: %s", name) // [Added Log]
	return name
}

// -------------------- health check endpoint --------------------

// Health provides a simple endpoint to check server health.
func Health(c *gin.Context) {
	clientIP := c.ClientIP() // [Added: capture client IP]
	// Log health check at DEBUG level (development only) - these are frequent and noisy
	httpContext := logger.NewHTTPContext("GET", "/health", c.Request.UserAgent(), clientIP, http.StatusOK)
	logger.LogDebugWithContext(httpContext, "Health check requested")
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// -------------------- user navigation and logout --------------------

// Logout logs the user out, removes them from ActiveUsers, vacates their
// position, and redirects to login.
func Logout(c *gin.Context, occupancyService services.OccupancyServiceInterface) {
	clientIP := c.ClientIP() // 🆕 Capture client IP for all log lines
	session := sessions.Default(c)

	userEmail, _ := session.Get("user").(string)
	position, _ := session.Get("refPosition").(string)
	meetName, _ := session.Get("meetName").(string)
	isAdmin, _ := session.Get("isAdmin").(bool)

	if userEmail == "" {
		logger.Warn.Printf("[Logout] No userEmail in session, skipping occupant removal. IP=%s", clientIP)
	} else if isAdmin {
		logger.Info.Printf("[Logout] Admin logout: user=%s, meet=%s, IP=%s", userEmail, meetName, clientIP)

		if meetName != "" {
			logger.Debug.Printf("[Logout] Calling ResetOccupancyForMeet for meet=%s", meetName)
			occupancyService.ResetOccupancyForMeet(meetName)
		}

		ActiveUsersMu.Lock()
		delete(ActiveUsers, userEmail)
		ActiveUsersMu.Unlock()
		logger.Info.Printf("[Logout] Admin user %s removed from active users list. IP=%s", userEmail, clientIP)

	} else {
		logger.Info.Printf("[Logout] Referee logout: user=%s, position=%s, meet=%s, IP=%s", userEmail, position, meetName, clientIP)

		if position != "" && meetName != "" {
			if err := occupancyService.UnsetPosition(meetName, position, userEmail); err != nil {
				logger.Error.Printf("[Logout] Failed to vacate seat=%s for user=%s, meet=%s: %v. IP=%s", position, userEmail, meetName, err, clientIP)
			} else {
				logger.Info.Printf("[Logout] Freed seat=%s for user=%s in meet=%s. IP=%s", position, userEmail, meetName, clientIP)
			}
		}

		ActiveUsersMu.Lock()
		delete(ActiveUsers, userEmail)
		ActiveUsersMu.Unlock()
		logger.Info.Printf("[Logout] User %s removed from active users list. IP=%s", userEmail, clientIP)
	}

	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("[Logout] Failed to save cleared session for user=%s. IP=%s: %v", userEmail, clientIP, err)
	}

	if isAdmin {
		logger.Info.Printf("[Logout] Redirecting admin user=%s to /set-meet. IP=%s", userEmail, clientIP)
		c.Redirect(http.StatusFound, "/set-meet")
	} else {
		logger.Info.Printf("[Logout] Redirecting referee user=%s to /logged-out. IP=%s", userEmail, clientIP)
		c.Redirect(http.StatusFound, "/logged-out")
	}
}

// -------------------- page rendering --------------------

// Index renders the main dashboard page screen after logging in
func Index(c *gin.Context) {
	clientIP := c.ClientIP()
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	isSudo, _ := session.Get("sudo").(bool)

	if !ok || meetName == "" {
		logger.Warn.Printf("[Index] No meetName in session. Redirecting to /set-meet. IP=%s", clientIP)
		c.Redirect(http.StatusFound, "/set-meet")
		return
	}

	if meetName == "Sudo" {
		logger.Info.Printf("[Index] Sudo mode detected. Redirecting to /sudo. IP=%s", clientIP)
		c.Redirect(http.StatusFound, "/sudo")
		return
	}
	logger.Debug.Printf("[Index] Loading dashboard for meet=%s. IP=%s", meetName, clientIP)

	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Printf("[Index] Failed to load meet credentials (nil returned). IP=%s", clientIP)
		c.String(http.StatusInternalServerError, "Failed to load meet credentials")
		return
	}

	var currentMeet *models.Meet
	for _, m := range creds.Meets {
		if m.Name == meetName {
			mCopy := m
			currentMeet = &mCopy
			break
		}
	}

	if currentMeet == nil {
		logger.Warn.Printf("[Index] Meet not found in credentials: meet=%s. IP=%s", meetName, clientIP)
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	// Log dashboard rendering at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/index", c.Request.UserAgent(), clientIP, http.StatusOK)
	httpContext["meetName"] = meetName
	httpContext["isSudo"] = isSudo
	logger.LogDebugWithContext(httpContext, "Rendering dashboard page")
	data := gin.H{
		"meetName": meetName,
		"IsSudo":   isSudo,
		"Logo":     currentMeet.Logo,
	}
	c.HTML(http.StatusOK, "index.html", data)
}

// GetQRCode generates and returns a QR code for the application URL.
func GetQRCode(c *gin.Context) {
	clientIP := c.ClientIP()
	meetName := c.Query("meetName")
	position := c.Query("position")

	// Log QR code request at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/qrcode", c.Request.UserAgent(), clientIP, http.StatusOK)
	httpContext["meetName"] = meetName
	httpContext["position"] = position
	logger.LogDebugWithContext(httpContext, "QR code generation requested")

	if meetName == "" || position == "" {
		logger.Warn.Printf("[GetQRCode] Missing query parameters. meetName='%s', position='%s', IP=%s", meetName, position, clientIP)
		c.String(http.StatusBadRequest, "Missing meetName or position query param")
		return
	}

	qrURL := fmt.Sprintf("%s/referee/%s/%s", ApplicationURL, meetName, position)
	logger.Debug.Printf("[GetQRCode] Constructed QR URL: %s", qrURL)

	qrBytes, err := services.GenerateQRCode(qrURL, 300, qrcode.Medium)
	if err != nil {
		logger.Error.Printf("[GetQRCode] QR generation failed. URL=%s, IP=%s, err=%v", qrURL, clientIP, err)
		c.String(http.StatusInternalServerError, "QR generation failed")
		return
	}

	// Log successful QR code generation at DEBUG level (development only)
	httpContext["statusCode"] = http.StatusOK
	logger.LogDebugWithContext(httpContext, "QR code generated successfully")
	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	if _, err := c.Writer.Write(qrBytes); err != nil {
		logger.Error.Printf("[GetQRCode] Failed to write QR code bytes to response. IP=%s, err=%v", clientIP, err)
	}
}

// SetConfig updates the global application and WebSocket URLs.
func SetConfig(appURL, wsURL string) {
	prevAppURL := ApplicationURL
	prevWSURL := WebsocketURL
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("[SetConfig] Global config updated. ApplicationURL: '%s' → '%s', WebsocketURL: '%s' → '%s'", prevAppURL, appURL, prevWSURL, wsURL)
}

// Lights renders the light control panel
func Lights(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	if !ok || meetName == "" {
		logger.Warn.Println("[Lights] Missing or empty meetName in session — redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	// Log lights page rendering at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/lights", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
	httpContext["meetName"] = meetName
	logger.LogDebugWithContext(httpContext, "Rendering lights page")

	creds := services.GetGlobalMeetCredentials()
	if creds == nil {
		logger.Error.Println("[Lights] Failed to load global meet credentials")
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

	logger.Debug.Printf("[Lights] Found meet: %s (Logo: %s)", currentMeet.Name, currentMeet.Logo)

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

	logger.Debug.Printf("[RefereeHandler] BEGIN. QR code requested: meetName=%s, position=%s, remoteAddr=%s",
		meetNameParam, position, c.Request.RemoteAddr)

	// retrieve session data
	session := sessions.Default(c)
	occupant, occupantExists := session.Get("user").(string)
	storedMeet, storedMeetExists := session.Get("meetName").(string)

	logger.Debug.Printf("[RefereeHandler] occupantExists=%v occupant=%q storedMeetExists=%v storedMeet=%q",
		occupantExists, occupant, storedMeetExists, storedMeet)

	// check if session meet mismatches route param
	if storedMeetExists && storedMeet != "" && storedMeet != meetNameParam {
		logger.Warn.Printf("[RefereeHandler] Conflict: session meetName=%s ≠ request meetName=%s — refusing seat claim.",
			storedMeet, meetNameParam)

		c.String(http.StatusConflict,
			"You already have meet=%s in session, but tried to claim a seat in meet=%s. Please re-select a meet.",
			storedMeet, meetNameParam)
		return
	}

	// generate anonymous occupant if needed
	if !occupantExists || occupant == "" {
		occupant = getNextAnonymousName()
		logger.Debug.Printf("[RefereeHandler] Generated new anonymous occupant: %s", occupant)
	}

	// set meetName in session if missing
	if !storedMeetExists || storedMeet == "" {
		logger.Debug.Printf("[RefereeHandler] Storing meetName=%s in session", meetNameParam)
		session.Set("meetName", meetNameParam)
	}

	// attempt seat claim
	if err := occupancyService.SetPosition(meetNameParam, position, occupant); err != nil {
		logger.Warn.Printf("[RefereeHandler] Seat claim failed — position=%s occupant=%s meet=%s: %v",
			position, occupant, meetNameParam, err)
		c.String(http.StatusConflict, "This referee seat (%s) is already taken.", position)
		return
	}

	// update session
	session.Set("user", occupant)
	session.Set("refPosition", position)
	logger.Debug.Printf("[RefereeHandler] Updated session: user=%s, refPosition=%s — attempting save", occupant, position)

	if err := session.Save(); err != nil {
		logger.Error.Printf("[RefereeHandler] session.Save failed for user=%s: %v", occupant, err)
	} else {
		logger.Debug.Printf("[RefereeHandler] session.Save succeeded for user=%s", occupant)
	}

	logger.Info.Printf("[RefereeHandler] SUCCESS: meet=%s, seat=%s claimed by occupant=%s",
		meetNameParam, position, occupant)

	// render appropriate referee view
	switch position {
	case "left", "Left":
		renderLeft(c, meetNameParam)
	case "center", "Center":
		renderCenter(c, meetNameParam)
	case "right", "Right":
		renderRight(c, meetNameParam)
	default:
		logger.Warn.Printf("[RefereeHandler] Invalid position requested: %s", position)
		c.String(http.StatusBadRequest, "Unknown position: %s", position)
	}
}

// renderCenter renders the center referee page
func renderCenter(c *gin.Context, meetName string) {
	// Log referee view rendering at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/referee/*/center", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
	httpContext["meetName"] = meetName
	httpContext["position"] = "center"
	logger.LogDebugWithContext(httpContext, "Rendering center referee view")

	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "center.html", data)
}

// renderRight renders the right referee page
func renderRight(c *gin.Context, meetName string) {
	// Log referee view rendering at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/referee/*/right", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
	httpContext["meetName"] = meetName
	httpContext["position"] = "right"
	logger.LogDebugWithContext(httpContext, "Rendering right referee view")

	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "right.html", data)
}

// renderLeft renders the left referee page
func renderLeft(c *gin.Context, meetName string) {
	// Log referee view rendering at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/referee/*/left", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
	httpContext["meetName"] = meetName
	httpContext["position"] = "left"
	logger.LogDebugWithContext(httpContext, "Rendering left referee view")

	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
	}
	c.HTML(http.StatusOK, "left.html", data)
}

// -------------- meet selection handling --------------

// ShowMeets renders the meet selection page.
func ShowMeets(c *gin.Context) {
	logger.Info.Println("[ShowMeets] Loading meet list for selection")

	// retrieve meet data using a mockable function for easier testing
	meetsData, err := services.LoadBasicMeets()
	if err != nil {
		logger.Error.Printf("[ShowMeets] Failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	logger.Info.Printf("[ShowMeets] Loaded %d meets", len(meetsData.Meets))

	// render the meet selection page with available meets
	c.HTML(http.StatusOK, "choose-meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

// ChooseMeetHandler handles GET /choose-meet and renders available meets.
func ChooseMeetHandler(c *gin.Context) {
	logger.Info.Println("[ChooseMeetHandler] Handling GET /choose-meet request")

	meetsData, err := services.LoadBasicMeets()
	if err != nil {
		logger.Error.Printf("[ChooseMeetHandler] Failed to load meets: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	logger.Info.Printf("[ChooseMeetHandler] Loaded %d meets", len(meetsData.Meets))

	c.HTML(http.StatusOK, "choose-meet.html", gin.H{
		"availableMeets": meetsData.Meets,
	})
}

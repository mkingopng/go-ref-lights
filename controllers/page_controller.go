// Package controllers file: controllers/page_controller.go
package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"net/http"
)

var (
	// ApplicationURL is the base URL of the application
	ApplicationURL string
	// WebsocketURL is the URL for the WebSocket server
	WebsocketURL string
)

// Health is a simple health check endpoint
func Health(c *gin.Context) {
	logger.Info.Println("Health: Health check requested")
	c.String(http.StatusOK, "OK")
}

// Home redirects the user to dashboard and vacates their position.
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
	c.Redirect(http.StatusFound, "/dashboard")
}

// Logout logs the user out, removes them from activeUsers, vacates their position, and redirects to login page.
// controllers/page_controller.go
func Logout(c *gin.Context, occupancyService services.OccupancyServiceInterface) {
	session := sessions.Default(c)

	userEmail, hasUser := session.Get("user").(string)
	position, hasPosition := session.Get("refPosition").(string)
	meetName, hasMeet := session.Get("meetName").(string)

	if hasUser && hasPosition && hasMeet {
		err := occupancyService.UnsetPosition(meetName, position, userEmail)
		if err != nil {
			logger.Error.Printf("Logout: error vacating position: %v", err)
		} else {
			logger.Info.Printf("Logout: position '%s' vacated for user '%s' in meet '%s'", position, userEmail, meetName)
		}
		delete(activeUsers, userEmail)
		logger.Info.Printf("Logout: User %s removed from active users list", userEmail)
	} else {
		logger.Warn.Println("Logout: Missing user, refPosition, or meetName from session.")
	}
	session.Clear()
	if err := session.Save(); err != nil {
		logger.Error.Printf("Logout: Error saving session: %v", err)
	} else {
		logger.Info.Println("Logout: Session cleared successfully")
	}
	c.Redirect(http.StatusFound, "/choose-meet")
}

// Index renders the main application page
func Index(c *gin.Context) {
	session := sessions.Default(c)
	meetName, ok := session.Get("meetName").(string)
	if !ok || meetName == "" {
		logger.Warn.Println("Index: No meet selected; redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}
	// Check if the session has the admin flag.
	isAdmin, _ := session.Get("isAdmin").(bool)

	logger.Info.Printf("Rendering index page for meet %s", meetName)
	data := gin.H{
		"WebsocketURL": WebsocketURL,
		"meetName":     meetName,
		"isAdmin":      isAdmin,
	}
	c.HTML(http.StatusOK, "index.html", data)
}

// ShowPositionsPage renders the positions page
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
			"LeftOccupied":   false, // Example data, replace with actual occupancy logic
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

// GetQRCode displays a QR code for the application URL
func GetQRCode(c *gin.Context) {
	logger.Info.Println("GetQRCode: Generating QR code")

	// Actually generate real PNG data:
	qrBytes, err := services.GenerateQRCode(300, 300, qrcode.Encode)
	if err != nil {
		logger.Error.Printf("GetQRCode: Error generating QR code: %v", err)
		c.String(http.StatusInternalServerError, "QR generation failed")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Content-Disposition", "inline; filename=\"qrcode.png\"")
	// Write the binary PNG bytes to the response
	if _, err := c.Writer.Write(qrBytes); err != nil {
		logger.Error.Printf("GetQRCode: Error writing QR code bytes: %v", err)
	}
}

// SetConfig sets global application and WebSocket URLs
func SetConfig(appURL, wsURL string) {
	ApplicationURL = appURL
	WebsocketURL = wsURL
	logger.Info.Printf("SetConfig: Global config updated: ApplicationURL=%s, WebsocketURL=%s", appURL, wsURL)
}

// PerformLogin processes user authentication
func PerformLogin(c *gin.Context) {
	session := sessions.Default(c)
	if session.Get("meetName") == nil {
		c.Redirect(http.StatusFound, "/") // Redirect to choose_meet page
		return
	}
	c.HTML(http.StatusOK, "login.html", gin.H{"MeetName": session.Get("meetName")})
}

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
		"WebsocketURL": WebsocketURL, // ✅ FIXED: WebsocketURL is now declared globally
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

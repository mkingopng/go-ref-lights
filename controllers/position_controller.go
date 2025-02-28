// Package controllers file: controllers/position_controller.go
package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

// PositionController struct with service dependency injection
type PositionController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewPositionController creates an instance of PositionController
func NewPositionController(service services.OccupancyServiceInterface) *PositionController {
	logger.Debug.Println("NewPositionController: Initializing PositionController")
	return &PositionController{OccupancyService: service}
}

// ShowPositionsPage displays the "choose your position" page
func (pc *PositionController) ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName, ok := session.Get("meetName").(string)
	if user == nil || !ok || meetName == "" {
		logger.Warn.Println("ShowPositionsPage: User not logged in or no meet selected; redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}

	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("ShowPositionsPage: Retrieved occupancy state: %+v", occ)

	// Get real occupancy from the service:
	occ = pc.OccupancyService.GetOccupancy(meetName)

	// Build the data structure to pass to positions.html
	data := gin.H{
		"Positions": map[string]interface{}{
			"LeftOccupied":   occ.LeftUser != "",
			"LeftUser":       occ.LeftUser,
			"centerOccupied": occ.CenterUser != "",
			"centerUser":     occ.CenterUser,
			"RightOccupied":  occ.RightUser != "",
			"RightUser":      occ.RightUser,
		},
		"meetName": meetName,
	}

	logger.Info.Println("ShowPositionsPage: Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition processes the form submission
func (pc *PositionController) ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName, ok := session.Get("meetName").(string)
	if user == nil || !ok || meetName == "" {
		logger.Warn.Println("ClaimPosition: User not logged in or no meet selected; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Grab the position from the form
	position := c.PostForm("position")
	userEmail := user.(string)
	logger.Info.Printf("ClaimPosition: User %s attempting to claim position %s in meet %s", userEmail, position, meetName)

	// Attempt to set position in the OccupancyService
	err := pc.OccupancyService.SetPosition(meetName, position, userEmail)
	if err != nil {
		logger.Error.Printf("ClaimPosition: Position is taken or invalid. %v", err)
		occ := pc.OccupancyService.GetOccupancy(meetName)
		c.HTML(http.StatusForbidden, "positions.html", gin.H{
			"Error":    "Sorry, that referee position is already occupied. Please choose a different one.",
			"meetName": meetName,
			"Positions": map[string]interface{}{
				"LeftOccupied":   occ.LeftUser != "",
				"LeftUser":       occ.LeftUser,
				"centerOccupied": occ.CenterUser != "",
				"centerUser":     occ.CenterUser,
				"RightOccupied":  occ.RightUser != "",
				"RightUser":      occ.RightUser,
			},
		})
		return
	}

	// Otherwise, the position was successfully claimed. Update the session:
	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("ClaimPosition: Error saving session for user %s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s for meet %s", userEmail, position, meetName)

	// Redirect to the correct referee view based on the claimed position
	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "center":
		c.Redirect(http.StatusFound, "/center")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		// If we don’t recognize the position, go back to /positions
		logger.Warn.Printf("ClaimPosition: Unknown position %s; redirecting to /positions", position)
		c.Redirect(http.StatusFound, "/positions")
	}
	// Finally, notify other clients that occupancy changed
	go pc.broadcastOccupancy(meetName)
}

// broadcastOccupancy sends a JSON payload indicating which seats are occupied.
// Any clients listening for "occupancyChanged" can update their UI accordingly.
func (pc *PositionController) broadcastOccupancy(meetName string) {
	occ := pc.OccupancyService.GetOccupancy(meetName)
	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}
	jsonBytes, _ := json.Marshal(msg)
	websocket.SendBroadcastMessage(jsonBytes)
}

// GetOccupancyAPI provides a JSON endpoint to retrieve the current occupancy.
func (pc *PositionController) GetOccupancyAPI(c *gin.Context) {
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)
	if !ok || meetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No meet selected"})
		return
	}

	occ := pc.OccupancyService.GetOccupancy(meetName)
	c.JSON(http.StatusOK, gin.H{
		"leftUser":   occ.LeftUser,
		"centreUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
	})
}

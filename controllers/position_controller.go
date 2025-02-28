// Package controllers file: controllers/position_controller.go
package controllers

import (
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

	// pass tje meetName to GetOccupancy
	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("ShowPositionsPage: Retrieved occupancy state: %+v", occ)
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

	position := c.PostForm("position")
	userEmail := user.(string)
	logger.Info.Printf("ClaimPosition: User %s attempting to claim position %s in meet %s", userEmail, position, meetName)

	err := pc.OccupancyService.SetPosition(meetName, position, userEmail)
	if err != nil {
		logger.Error.Printf("ClaimPosition: Failed to set position %s for user %s: %v", position, userEmail, err)
		c.String(http.StatusForbidden, "Error: %s", err.Error())
		return
	}

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("ClaimPosition: Error saving session for user %s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s for meet %s", userEmail, position, meetName)

	// Redirect to the appropriate view based on the claimed position
	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "center":
		c.Redirect(http.StatusFound, "/center")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		logger.Warn.Printf("ClaimPosition: Unknown position %s; redirecting to /positions", position)
		c.Redirect(http.StatusFound, "/positions")
	}
	go pc.broadcastOccupancy(meetName)
}

func (pc *PositionController) broadcastOccupancy(meetName string) {
	occ := pc.OccupancyService.GetOccupancy(meetName)
	websocket.BroadcastMessage(meetName, map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centreUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
	})
}

// GetOccupancyAPI in position_controller.go (or a new file):
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

// Package controllers manages referee position allocation, vacancy, and real-time occupancy updates.
// File: controllers/position_controller.go
package controllers

import (
	"encoding/json"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"net/http"
)

// PositionController manages referee position assignments
type PositionController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewPositionController initializes a PositionController instance
func NewPositionController(service services.OccupancyServiceInterface) *PositionController {
	logger.Debug.Println("[NewPositionController] Initializing PositionController")
	return &PositionController{OccupancyService: service}
}

// ------------------- Position vacancy -------------------

// VacatePosition allows a referee to vacate their assigned position
func (pc *PositionController) VacatePosition(c *gin.Context) {
	c.Redirect(http.StatusFound, "/logout?reason=vacate")
}

// ------------------- Real-time occupancy updates -------------------

func (pc *PositionController) BroadcastOccupancy(meetName string) {
	// get the current occupancy state
	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("[BroadcastOccupancy] Fetched occupancy: %+v", occ)

	// create the message
	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}

	// marshal the message to JSON
	jsonBytes, _ := json.Marshal(msg)
	logger.Debug.Printf("[BroadcastOccupancy] Sending message: %s", string(jsonBytes))

	// send to all connected clients
	go websocket.SendBroadcastMessage(jsonBytes)
	logger.Debug.Printf("[BroadcastOccupancy] Finished for meet=%s", meetName)
}

// ------------------- API endpoints -------------------

// GetOccupancyAPI provides a JSON response with the current referee occupancy
func (pc *PositionController) GetOccupancyAPI(c *gin.Context) {
	// get the meet from the session
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)

	// check if user is logged in and a meet is selected
	if !ok || meetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No meet selected"})
		return
	}

	// get the current occupancy state
	occ := pc.OccupancyService.GetOccupancy(meetName)
	c.JSON(http.StatusOK, gin.H{
		"leftUser":   occ.LeftUser,
		"centreUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
	})
}

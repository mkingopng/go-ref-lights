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
	clientIP := c.ClientIP()
	session := sessions.Default(c)
	userEmail, _ := session.Get("user").(string)
	position, _ := session.Get("refPosition").(string)
	meetName, _ := session.Get("meetName").(string)
	logger.Info.Printf("[VacatePosition] Request to vacate from user=%s, position=%s, meet=%s, IP=%s", userEmail, position, meetName, clientIP)
	c.Redirect(http.StatusFound, "/logout?reason=vacate")
}

// ------------------- Real-time occupancy updates -------------------

func (pc *PositionController) BroadcastOccupancy(meetName string) {
	logger.Info.Printf("[BroadcastOccupancy] Starting broadcast for meet=%s", meetName)

	// get the current occupancy state
	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("[BroadcastOccupancy] Current occupancy for meet=%s: Left=%s, Center=%s, Right=%s",
		meetName, occ.LeftUser, occ.CenterUser, occ.RightUser)

	// create the message
	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}

	// marshal the message to JSON
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		logger.Error.Printf("[BroadcastOccupancy] Failed to marshal message for meet=%s: %v", meetName, err)
		return
	}
	logger.Debug.Printf("[BroadcastOccupancy] Marshaled message for meet=%s: %s", meetName, string(jsonBytes))

	// send to all connected clients
	go websocket.SendBroadcastMessage(jsonBytes)
	logger.Info.Printf("[BroadcastOccupancy] Broadcast message sent for meet=%s", meetName)
}

// ------------------- API endpoints -------------------

// GetOccupancyAPI provides a JSON response with the current referee occupancy
func (pc *PositionController) GetOccupancyAPI(c *gin.Context) {
	clientIP := c.ClientIP()
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)

	if !ok || meetName == "" {
		logger.Warn.Printf("[GetOccupancyAPI] No meet selected in session. IP=%s", clientIP)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No meet selected"})
		return
	}

	logger.Info.Printf("[GetOccupancyAPI] Fetching occupancy for meet=%s. IP=%s", meetName, clientIP)

	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("[GetOccupancyAPI] Occupancy state: meet=%s, Left=%s, Center=%s, Right=%s",
		meetName, occ.LeftUser, occ.CenterUser, occ.RightUser)

	c.JSON(http.StatusOK, gin.H{
		"leftUser":   occ.LeftUser,
		"centreUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
	})
}

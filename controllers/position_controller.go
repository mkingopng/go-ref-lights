// Package controllers manages referee position allocation, vacancy, and real-time occupancy updates.
// File: controllers/position_controller.go
package controllers

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-xray-sdk-go/xray"
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

// ------------------- Position assignment -------------------

// ClaimPosition allows a referee to claim a position
func (pc *PositionController) ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName, ok := session.Get("meetName").(string)

	if user == nil || !ok || meetName == "" {
		logger.Warn.Println("[ClaimPosition] User not logged in or no meet selected; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position := c.PostForm("position")
	userEmail := user.(string)
	logger.Info.Printf("[ClaimPosition] User=%s attempting to claim position=%s in meet=%s", userEmail, position, meetName)

	err := pc.OccupancyService.SetPosition(meetName, position, userEmail)
	if err != nil {
		logger.Error.Printf("[ClaimPosition] Position is taken or invalid: %v", err)
		c.String(http.StatusForbidden, "Seat is already taken or invalid. Please try another approach.")
		return
	}

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("[ClaimPosition] Error saving session for user=%s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "center":
		c.Redirect(http.StatusFound, "/center")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		logger.Warn.Printf("[ClaimPosition] Unknown position %s; redirecting to /index", position)
		c.Redirect(http.StatusFound, "/index")
	}
	go pc.BroadcastOccupancy(meetName)
}

// ------------------- Position vacancy -------------------

// VacatePosition allows a referee to vacate their assigned position
func (pc *PositionController) VacatePosition(c *gin.Context) {
	c.Redirect(http.StatusFound, "/logout?reason=vacate")
}

// ------------------- Real-time occupancy updates -------------------

func (pc *PositionController) BroadcastOccupancy(meetName string) {
	_, seg := xray.BeginSegment(context.Background(), "BroadcastOccupancy")
	if seg != nil {
		defer seg.Close(nil)
		_ = seg.AddAnnotation("meet", meetName)
	}

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

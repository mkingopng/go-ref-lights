// Package controllers manages referee position allocation, vacancy, and real-time occupancy updates.
// File: controllers/position_controller.go
package controllers

import (
	"encoding/json"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// PositionController manages referee position assignments
type PositionController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewPositionController initializes a PositionController instance
func NewPositionController(service services.OccupancyServiceInterface) *PositionController {
	initContext := logger.NewSystemContext("initialization", "position_controller")
	logger.LogDebugWithContext(initContext, "Initializing PositionController")
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
	// Log position vacation request at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("POST", "/position/vacate", c.Request.UserAgent(), clientIP, http.StatusFound)
	httpContext["meetName"] = meetName
	httpContext["position"] = position
	httpContext["refereeId"] = userEmail
	logger.LogDebugWithContext(httpContext, "Position vacation requested, redirecting to logout")
	c.Redirect(http.StatusFound, "/logout?reason=vacate")
}

// ------------------- Real-time occupancy updates -------------------

func (pc *PositionController) BroadcastOccupancy(meetName string) {
	broadcastContext := logger.NewPositionContext("broadcast_occupancy", meetName, "", "")
	logger.LogInfoWithContext(broadcastContext, "Starting occupancy broadcast for meet %s", meetName)

	// get the current occupancy state
	occ := pc.OccupancyService.GetOccupancy(meetName)
	occupancyContext := logger.NewPositionContext("get_occupancy", meetName, "", "")
	occupancyContext["leftUser"] = occ.LeftUser
	occupancyContext["centerUser"] = occ.CenterUser
	occupancyContext["rightUser"] = occ.RightUser
	logger.LogDebugWithContext(occupancyContext,
		"Current occupancy state - Left: %s, Center: %s, Right: %s",
		occ.LeftUser, occ.CenterUser, occ.RightUser)

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
		marshalContext := logger.NewPositionContext("marshal_error", meetName, "", "")
		marshalContext["error"] = err.Error()
		marshalContext["messageType"] = "occupancy_broadcast"
		logger.LogErrorWithContext(marshalContext, "Failed to marshal occupancy broadcast message: %v", err)
		return
	}

	marshalContext := logger.NewPositionContext("marshal_success", meetName, "", "")
	marshalContext["messageSize"] = len(jsonBytes)
	marshalContext["messageType"] = "occupancy_broadcast"
	logger.LogDebugWithContext(marshalContext, "Occupancy message marshaled successfully (%d bytes)", len(jsonBytes))

	// send to all connected clients
	go websocket.SendBroadcastMessage(jsonBytes)

	sendContext := logger.NewPositionContext("broadcast_sent", meetName, "", "")
	sendContext["messageSize"] = len(jsonBytes)
	logger.LogInfoWithContext(sendContext, "Occupancy broadcast message sent for meet %s", meetName)
}

// ------------------- API endpoints -------------------

// GetOccupancyAPI provides a JSON response with the current referee occupancy
func (pc *PositionController) GetOccupancyAPI(c *gin.Context) {
	clientIP := c.ClientIP()
	session := sessions.Default(c)
	meetNameRaw := session.Get("meetName")
	meetName, ok := meetNameRaw.(string)

	if !ok || meetName == "" {
		apiContext := logger.NewHTTPContext("GET", "/occupancy", c.Request.UserAgent(), clientIP, http.StatusBadRequest)
		apiContext["component"] = "position_controller"
		apiContext["action"] = "get_occupancy_api"
		apiContext["error"] = "no_meet_selected"
		logger.LogWarnWithContext(apiContext, "Occupancy API request without meet selection")
		c.JSON(http.StatusBadRequest, gin.H{"error": "No meet selected"})
		return
	}

	// Log occupancy API request at DEBUG level (development only)
	httpContext := logger.NewHTTPContext("GET", "/occupancy", c.Request.UserAgent(), clientIP, http.StatusOK)
	httpContext["meetName"] = meetName
	logger.LogDebugWithContext(httpContext, "Occupancy data requested")

	occ := pc.OccupancyService.GetOccupancy(meetName)
	apiContext := logger.NewHTTPContext("GET", "/occupancy", c.Request.UserAgent(), clientIP, http.StatusOK)
	apiContext["component"] = "position_controller"
	apiContext["action"] = "get_occupancy_api"
	apiContext["meetName"] = meetName
	apiContext["leftUser"] = occ.LeftUser
	apiContext["centerUser"] = occ.CenterUser
	apiContext["rightUser"] = occ.RightUser
	logger.LogDebugWithContext(apiContext,
		"Occupancy API response - Left: %s, Center: %s, Right: %s",
		occ.LeftUser, occ.CenterUser, occ.RightUser)

	c.JSON(http.StatusOK, gin.H{
		"leftUser":   occ.LeftUser,
		"centreUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
	})
}

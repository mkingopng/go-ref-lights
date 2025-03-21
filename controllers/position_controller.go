// Package controllers manages referee position allocation, vacancy, and real-time occupancy updates.
// File: controllers/position_controller.go
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

// PositionController manages referee position assignments.
type PositionController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewPositionController initializes a PositionController instance.
func NewPositionController(service services.OccupancyServiceInterface) *PositionController {
	logger.Debug.Println("[NewPositionController] Initializing PositionController")
	return &PositionController{OccupancyService: service}
}

// ------------------- Position selection -------------------

// ShowPositionsPage renders the referee position selection page.
func (pc *PositionController) ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	meetName, ok := session.Get("meetName").(string)
	if user == nil || !ok || meetName == "" {
		logger.Warn.Println("[ShowPositionsPage] User not logged in or no meet selected; redirecting to /meets")
		c.Redirect(http.StatusFound, "/meets")
		return
	}

	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("[ShowPositionsPage] Retrieved occupancy state: %+v", occ)

	// Possibly redundant second call?
	occ = pc.OccupancyService.GetOccupancy(meetName)

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

	logger.Info.Println("[ShowPositionsPage] Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// ------------------- Position assignment -------------------

// ClaimPosition allows a referee to claim a position.
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
		// Replacing old fmt.Println:
		logger.Debug.Printf("[ClaimPosition] Controller calling GetOccupancy with: %s", meetName)

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

	// store referee position in session.
	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("[ClaimPosition] Error saving session for user=%s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	logger.Info.Printf("[ClaimPosition] User=%s successfully claimed position=%s for meet=%s", userEmail, position, meetName)

	// redirect to the correct path
	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "center":
		c.Redirect(http.StatusFound, "/center")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		logger.Warn.Printf("[ClaimPosition] Unknown position %s; redirecting to /positions", position)
		c.Redirect(http.StatusFound, "/positions")
	}
	// broadcast occupancy changes asynchronously
	go pc.BroadcastOccupancy(meetName)
}

// ------------------- Position vacancy -------------------

// VacatePosition allows a referee to vacate their assigned position.
func (pc *PositionController) VacatePosition(c *gin.Context) {
	session := sessions.Default(c)
	userEmail, ok := session.Get("user").(string)
	meetName, ok2 := session.Get("meetName").(string)

	if !ok || !ok2 || userEmail == "" || meetName == "" {
		logger.Warn.Println("[VacatePosition] User not logged in or no meet selected; redirecting to /login")
		c.Redirect(http.StatusFound, "/index")
		return
	}

	position, ok3 := session.Get("refPosition").(string)
	if !ok3 || position == "" {
		logger.Warn.Printf("[VacatePosition] user=%s not in any seat for meet=%s; can't vacate", userEmail, meetName)
		c.Redirect(http.StatusFound, "/index")
		return
	}

	if err := pc.OccupancyService.UnsetPosition(meetName, position, userEmail); err != nil {
		logger.Error.Printf("[VacatePosition] Error unsetting position for user=%s: %v", userEmail, err)
		c.Redirect(http.StatusFound, "/index")
		return
	}

	session.Delete("refPosition")
	if err := session.Save(); err != nil {
		logger.Error.Printf("[VacatePosition] Error saving session for user=%s: %v", userEmail, err)
		c.Redirect(http.StatusFound, "/index")
		return
	}

	logger.Info.Printf("[VacatePosition] user=%s vacated seat=%s for meet=%s", userEmail, position, meetName)
	go pc.BroadcastOccupancy(meetName)
	c.Redirect(http.StatusFound, "/index")
}

// ------------------- Real-time occupancy updates -------------------

// BroadcastOccupancy sends a real-time update of occupied referee positions.
func (pc *PositionController) BroadcastOccupancy(meetName string) {
	logger.Debug.Printf("[BroadcastOccupancy] Entering for meet=%s", meetName)
	occ := pc.OccupancyService.GetOccupancy(meetName)

	logger.Debug.Printf("[BroadcastOccupancy] Fetched occupancy: %+v", occ)

	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}
	jsonBytes, _ := json.Marshal(msg)
	logger.Debug.Printf("[BroadcastOccupancy] Sending message: %s", string(jsonBytes))

	go websocket.SendBroadcastMessage(jsonBytes)
	logger.Debug.Printf("[BroadcastOccupancy] Finished for meet=%s", meetName)
}

// ------------------- API endpoints -------------------

// GetOccupancyAPI provides a JSON response with the current referee occupancy.
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
		"centreUser": occ.CenterUser, // spelled “centreUser” for the JSON response
		"rightUser":  occ.RightUser,
	})
}

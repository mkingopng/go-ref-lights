// Package controllers file: controllers/position_controller.go
package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"net/http"
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
		logger.Error.Printf("ClaimPosition: Position is taken or invalid. %v", err)

		fmt.Println("Controller Calling GetOccupancy with:", meetName) // 🛠 Debugging Output
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

	session.Set("refPosition", position)
	if err := session.Save(); err != nil {
		logger.Error.Printf("ClaimPosition: Error saving session for user %s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s for meet %s", userEmail, position, meetName)

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
	go pc.BroadcastOccupancy(meetName)
}

// VacatePosition function
func (pc *PositionController) VacatePosition(c *gin.Context) {
	session := sessions.Default(c)
	userEmail, ok := session.Get("user").(string)
	meetName, ok2 := session.Get("meetName").(string)
	if !ok || !ok2 || userEmail == "" || meetName == "" {
		logger.Warn.Println("VacatePosition: User not logged in or no meet selected; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position, ok3 := session.Get("refPosition").(string)
	if !ok3 || position == "" {
		logger.Warn.Printf("VacatePosition: user %s not in any seat for meet %s; can't vacate", userEmail, meetName)
		c.Redirect(http.StatusFound, "/positions")
		return
	}

	err := pc.OccupancyService.UnsetPosition(meetName, position, userEmail)
	if err != nil {
		logger.Error.Printf("VacatePosition: error unsetting position for user %s: %v", userEmail, err)
		c.HTML(http.StatusInternalServerError, "positions.html", gin.H{
			"Error":    "Unable to vacate your seat. " + err.Error(),
			"meetName": meetName,
		})
		return
	}

	session.Delete("refPosition")
	if err := session.Save(); err != nil {
		logger.Error.Printf("VacatePosition: Error saving session for user %s: %v", userEmail, err)
		c.String(http.StatusInternalServerError, "Error saving session")
		return
	}

	logger.Info.Printf("VacatePosition: user %s vacated seat %s for meet %s", userEmail, position, meetName)
	go pc.BroadcastOccupancy(meetName)
	c.Redirect(http.StatusFound, "/positions")
}

// BroadcastOccupancy sends a JSON payload indicating which seats are occupied.
// Any clients listening for "occupancyChanged" can update their UI accordingly.
func (pc *PositionController) BroadcastOccupancy(meetName string) {
	logger.Debug.Printf("DEBUG: Entering broadcastOccupancy for meet: %s", meetName)
	occ := pc.OccupancyService.GetOccupancy(meetName)
	logger.Debug.Printf("DEBUG: broadcastOccupancy fetched occupancy: %+v", occ)

	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}
	jsonBytes, _ := json.Marshal(msg)
	logger.Debug.Printf("DEBUG: broadcastOccupancy sending message: %s", string(jsonBytes))

	go func() {
		websocket.SendBroadcastMessage(jsonBytes)
	}()

	logger.Debug.Printf("DEBUG: Finished broadcastOccupancy for meet: %s", meetName)
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

// Package controllers file: controllers/position_controller.go
package controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
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
	if user == nil {
		logger.Warn.Println("ShowPositionsPage: User not logged in; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	occ := pc.OccupancyService.GetOccupancy()
	logger.Debug.Printf("ShowPositionsPage: Retrieved occupancy state: %+v", occ)
	data := gin.H{
		"Positions": map[string]interface{}{
			"LeftOccupied":   occ.LeftUser != "",
			"LeftUser":       occ.LeftUser,
			"CentreOccupied": occ.CentreUser != "",
			"CentreUser":     occ.CentreUser,
			"RightOccupied":  occ.RightUser != "",
			"RightUser":      occ.RightUser,
		},
	}

	logger.Info.Println("ShowPositionsPage: Rendering positions page")
	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition processes the form submission
func (pc *PositionController) ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil {
		logger.Warn.Println("ClaimPosition: User not logged in; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position := c.PostForm("position")
	userEmail := user.(string)
	logger.Info.Printf("ClaimPosition: User %s attempting to claim position %s", userEmail, position)

	err := pc.OccupancyService.SetPosition(position, userEmail)
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

	logger.Info.Printf("ClaimPosition: User %s successfully claimed position %s", userEmail, position)

	// Redirect to the appropriate view based on the claimed position
	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "centre":
		c.Redirect(http.StatusFound, "/centre")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		logger.Warn.Printf("ClaimPosition: Unknown position %s; redirecting to /positions", position)
		c.Redirect(http.StatusFound, "/positions")
	}
}

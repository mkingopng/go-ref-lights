// file: controllers/position_controller.go
package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/services"
	"net/http"
)

// PositionController struct with service dependency injection
type PositionController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewPositionController creates an instance of PositionController
func NewPositionController(service services.OccupancyServiceInterface) *PositionController {
	return &PositionController{OccupancyService: service}
}

// ShowPositionsPage displays the "choose your position" page
func (pc *PositionController) ShowPositionsPage(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	occ := pc.OccupancyService.GetOccupancy()
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

	c.HTML(http.StatusOK, "positions.html", data)
}

// ClaimPosition processes the form submission
func (pc *PositionController) ClaimPosition(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")

	if user == nil {
		c.Redirect(http.StatusFound, "/login")
		return
	}

	position := c.PostForm("position")
	userEmail := user.(string)

	err := pc.OccupancyService.SetPosition(position, userEmail)
	if err != nil {
		c.String(http.StatusForbidden, "Error: %s", err.Error())
		return
	}

	session.Set("refPosition", position)
	session.Save()

	switch position {
	case "left":
		c.Redirect(http.StatusFound, "/left")
	case "centre":
		c.Redirect(http.StatusFound, "/centre")
	case "right":
		c.Redirect(http.StatusFound, "/right")
	default:
		c.Redirect(http.StatusFound, "/positions")
	}
}

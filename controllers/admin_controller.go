// controllers/admin_controller.go
package controllers

import (
	"go-ref-lights/logger"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/services"
)

// AdminController provides admin operations.
type AdminController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewAdminController creates a new instance of AdminController.
func NewAdminController(service services.OccupancyServiceInterface) *AdminController {
	return &AdminController{OccupancyService: service}
}

// AdminPanel renders the admin panel page.
// It requires that the logged-in user is an admin (e.g. username "admin").
func (ac *AdminController) AdminPanel(c *gin.Context) {
	session := sessions.Default(c)
	adminVal := session.Get("isAdmin")
	logger.Debug.Printf("AdminPanel: isAdmin value in session: %v", adminVal)
	isAdmin, ok := adminVal.(bool)
	if !ok || !isAdmin {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Determine the meet name—either from query params or session.
	meetName := c.Query("meet")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}

	occ := ac.OccupancyService.GetOccupancy(meetName)
	data := gin.H{
		"meetName":  meetName,
		"occupancy": occ,
	}
	c.HTML(http.StatusOK, "admin.html", data)
}

// ForceVacate allows an admin to force a referee position to be vacated.
func (ac *AdminController) ForceVacate(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil || user != "admin" {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	position := c.PostForm("position")
	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing parameters")
		return
	}

	occ := ac.OccupancyService.GetOccupancy(meetName)
	var occupant string
	switch position {
	case "left":
		occupant = occ.LeftUser
	case "center":
		occupant = occ.CenterUser
	case "right":
		occupant = occ.RightUser
	default:
		c.String(http.StatusBadRequest, "Invalid position")
		return
	}

	if occupant == "" {
		c.String(http.StatusBadRequest, "Position already vacant")
		return
	}

	if err := ac.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// Remove the user from activeUsers to allow future logins.
	delete(activeUsers, occupant)

	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ResetInstance forces a full reset of the meet instance.
// This clears the active users map and resets occupancy.
func (ac *AdminController) ResetInstance(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get("user")
	if user == nil || user != "admin" {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	if meetName == "" {
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}

	// Clear all active users.
	activeUsers = make(map[string]bool)

	// Reset the occupancy for the meet.
	ac.OccupancyService.ResetOccupancyForMeet(meetName)

	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

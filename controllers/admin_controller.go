// Package controllers controllers/admin_controller.go
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
	OccupancyService   services.OccupancyServiceInterface
	PositionController *PositionController
}

// NewAdminController creates a new instance of AdminController.
func NewAdminController(service services.OccupancyServiceInterface, posController *PositionController) *AdminController {
	return &AdminController{
		OccupancyService:   service,
		PositionController: posController,
	}
}

// AdminPanel renders admin panel page and requires that the user is an admin
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

	// Check if user is an admin
	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Println("ForceVacate: Unauthorized attempt")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	position := c.PostForm("position")

	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing parameters")
		return
	}

	// Ensure `GetOccupancy` returns a valid object
	occ := ac.OccupancyService.GetOccupancy(meetName)
	if occ == (services.Occupancy{}) { // Empty struct check
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	var occupant string
	switch position {
	case "left":
		occupant = occ.LeftUser
		occ.LeftUser = ""
	case "center":
		occupant = occ.CenterUser
		occ.CenterUser = ""
	case "right":
		occupant = occ.RightUser
		occ.RightUser = ""
	default:
		c.String(http.StatusBadRequest, "Invalid position")
		return
	}

	// Prevent vacating an already empty position
	if occupant == "" {
		c.String(http.StatusBadRequest, "Position already vacant")
		return
	}

	// Remove user from active users
	delete(activeUsers, occupant)

	// Save the new occupancy state
	if err := ac.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// Ensure WebSocket Broadcast Function is Called Correctly
	ac.PositionController.BroadcastOccupancy(meetName) // all BroadcastOccupancy

	logger.Info.Printf("ForceVacate: Admin forcibly removed %s from %s position in %s", occupant, position, meetName)

	// Redirect back to admin panel
	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ResetInstance forces a full reset of the meet instance.
// This clears the active users map and resets occupancy.
func (ac *AdminController) ResetInstance(c *gin.Context) {
	session := sessions.Default(c)

	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Println("ResetInstance: Unauthorized attempt")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		logger.Warn.Println("ResetInstance: No meet specified")
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}

	logger.Info.Printf("ResetInstance: Resetting meet '%s'", meetName)

	activeUsers = make(map[string]bool)
	ac.OccupancyService.ResetOccupancyForMeet(meetName) // ✅ Ensure this is called

	ac.PositionController.BroadcastOccupancy(meetName)

	logger.Info.Printf("ResetInstance: Meet '%s' reset successfully", meetName)

	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ForceLogout logs out a user forcibly (admin action)
func (ac *AdminController) ForceLogout(c *gin.Context) {
	session := sessions.Default(c)
	isAdmin := session.Get("isAdmin")

	if isAdmin == nil || isAdmin != true {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	username := c.PostForm("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	if _, exists := activeUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	delete(activeUsers, username)
	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// Package controllers provides HTTP handlers for various admin operations.
// File: controllers/admin_controller.go
package controllers

import (
	"go-ref-lights/logger"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"go-ref-lights/services"
)

// ---------------- Admin Controller ----------------

// AdminController provides admin operations for managing meets, referees, and users
type AdminController struct {
	OccupancyService   services.OccupancyServiceInterface
	PositionController *PositionController
}

// NewAdminController initializes a new instance of AdminController
func NewAdminController(service services.OccupancyServiceInterface, posController *PositionController) *AdminController {
	return &AdminController{
		OccupancyService:   service,
		PositionController: posController,
	}
}

// ---------------- admin panel management ----------------

// AdminPanel renders the admin panel page, ensuring the user has admin privileges.
// If the user is not an admin, they receive an HTTP 401 Unauthorized response.
// Requires a meet name to be specified via query parameters or session.
func (ac *AdminController) AdminPanel(c *gin.Context) {
	session := sessions.Default(c)
	adminVal := session.Get("isAdmin")

	logger.Debug.Printf("AdminPanel: isAdmin value in session: %v", adminVal)

	isAdmin, ok := adminVal.(bool)

	if !ok || !isAdmin {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Retrieve meet name from query parameters or session
	meetName := c.Query("meet")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}

	occupancy := ac.OccupancyService.GetOccupancy(meetName)
	data := gin.H{
		"meetName":  meetName,
		"occupancy": occupancy,
	}

	c.HTML(http.StatusOK, "admin.html", data)
}

// ---------------- referee position management ----------------

// ForceVacate allows an admin to forcibly vacate a referee from their assigned position.
// Requires:
// - `meetName` and `position` from the POST request body.
// - The user to have admin privileges.
func (ac *AdminController) ForceVacate(c *gin.Context) {
	session := sessions.Default(c)

	// Ensure user is an admin
	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Println("ForceVacate: Unauthorized attempt")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	position := c.PostForm("position")

	// validate input parameters
	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing parameters")
		return
	}

	// Ensure `GetOccupancy` returns a valid object
	occupancy := ac.OccupancyService.GetOccupancy(meetName)
	if occupancy == (services.Occupancy{}) { // Check if meet exists
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	var occupant string
	switch position {
	case "left":
		occupant = occupancy.LeftUser
		occupancy.LeftUser = ""

	case "center":
		occupant = occupancy.CenterUser
		occupancy.CenterUser = ""

	case "right":
		occupant = occupancy.RightUser
		occupancy.RightUser = ""

	default:
		c.String(http.StatusBadRequest, "Invalid position")
		return
	}

	// ensure there is an occupant before vacating
	if occupant == "" {
		c.String(http.StatusBadRequest, "Position already vacant")
		return
	}

	// remove user from the active list
	delete(activeUsers, occupant)

	// update occupancy state
	if err := ac.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// ensure WebSocket Broadcast function is called Correctly
	ac.PositionController.BroadcastOccupancy(meetName) // broadcast updated occupancy

	logger.Info.Printf("ForceVacate: Admin forcibly removed %s from %s position in %s", occupant, position, meetName)

	// Redirect back to the admin panel
	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ---------------- meet management ----------------

// ResetInstance performs a full reset of the meet instance.
// This clears active users and resets all referee positions.
func (ac *AdminController) ResetInstance(c *gin.Context) {
	session := sessions.Default(c)

	// ensure user is an admin
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

	activeUsers = make(map[string]bool) // clear active users
	ac.OccupancyService.ResetOccupancyForMeet(meetName)

	// broadcast updated occupancy state
	ac.PositionController.BroadcastOccupancy(meetName)

	logger.Info.Printf("ResetInstance: Meet '%s' reset successfully", meetName)

	// redirect back to admin panel
	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ---------------- user management ----------------

// ForceLogout forcibly logs out a user (admin action).
// requires:
// - `username` from the POST request body.
// - the user to have admin privileges.
func (ac *AdminController) ForceLogout(c *gin.Context) {
	session := sessions.Default(c)

	// ensure user is an admin
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

	// check if user is logged in
	if _, exists := activeUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	// remove user from the active list
	delete(activeUsers, username)

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

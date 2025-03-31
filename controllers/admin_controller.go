// Package controllers provide HTTP handlers for various admin operations.
// File: controllers/admin_controller.go
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

type PositionControllerInterface interface {
	BroadcastOccupancy(meetName string)
}

// AdminController provides admin operations for managing meets, referees, and users.
type AdminController struct {
	OccupancyService   services.OccupancyServiceInterface
	PositionController PositionControllerInterface
}

// NewAdminController initializes a new instance of AdminController.
func NewAdminController(service services.OccupancyServiceInterface, posController PositionControllerInterface) *AdminController {
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

	logger.Debug.Printf("[AdminPanel] isAdmin from session: %v", adminVal)

	isAdmin, ok := adminVal.(bool)
	if !ok || !isAdmin {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// retrieve meet name from query parameters or session
	meetName := c.Query("meet")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}
	// load meets
	creds, err := services.LoadMeetCredentials()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load meets")
		return
	}

	// find the correct logo
	var logo string
	for _, m := range creds.Meets {
		if m.Name == meetName {
			logo = m.Logo
			break
		}
	}

	// get occupancy
	occupancy := ac.OccupancyService.GetOccupancy(meetName)

	// pass everything to the template
	data := gin.H{
		"meetName":  meetName,
		"occupancy": occupancy,
		"Logo":      logo,
	}

	c.HTML(http.StatusOK, "admin.html", data)
}

// ---------------- referee position management ----------------

// ForceVacate
/*
Forcibly unassigns a referee from the specified position in a meet.
Requires admin privileges. It removes the occupant from ActiveUsers, calls
UnsetPosition, and redirects back to the admin panel.
*/
func (ac *AdminController) ForceVacate(c *gin.Context) {
	session := sessions.Default(c)

	// ensure user is an admin
	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Println("[ForceVacate] Unauthorized attempt")
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

	// ensure `GetOccupancy` returns a valid object
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
	delete(ActiveUsers, occupant)

	// update occupancy state
	if err := ac.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// ensure WebSocket Broadcast function is called
	ac.PositionController.BroadcastOccupancy(meetName)

	logger.Info.Printf("[ForceVacate] Admin forcibly removed %s from %s position in %s",
		occupant, position, meetName)

	// redirect back to the admin panel
	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ---------------- meet management ----------------

// ResetInstance
/*
Performs a full reset of the specified meet, clearing all active users and
resetting referee positions. Requires admin privileges. It then broadcasts
updated occupancy data to connected clients and redirects back to the admin
panel
*/
func (ac *AdminController) ResetInstance(c *gin.Context) {
	session := sessions.Default(c)

	// ensure user is an admin
	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Println("[ResetInstance] Unauthorized attempt")
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}
	meetName := c.PostForm("meetName")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		logger.Warn.Println("[ResetInstance] No meet specified")
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}

	logger.Info.Printf("[ResetInstance] Resetting meet '%s'", meetName)

	// clear active users
	ActiveUsers = make(map[string]bool)

	// reset occupancy
	ac.OccupancyService.ResetOccupancyForMeet(meetName)
	ac.PositionController.BroadcastOccupancy(meetName)

	logger.Info.Printf("[ResetInstance] Meet '%s' reset successfully", meetName)

	// redirect back to admin panel
	c.Redirect(http.StatusFound, "/admin?meet="+meetName)
}

// ---------------- user management ----------------

// ForceLogout
/*
Forcibly logs out a specified user by removing them from ActiveUsers.
This action requires admin privileges. Returns a JSON status message upon
success.
*/
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
	if _, exists := ActiveUsers[username]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	// remove user from the active list
	delete(ActiveUsers, username)

	c.JSON(http.StatusOK, gin.H{"message": "User logged out successfully"})
}

// SudoController handles global "superuser" actions across meets.
type SudoController struct {
	OccupancyService services.OccupancyServiceInterface
}

// NewSudoController constructs the controller, injecting needed services.
func NewSudoController(svc services.OccupancyServiceInterface) *SudoController {
	return &SudoController{
		OccupancyService: svc,
	}
}

// SudoPanel is an example method in SudoController
// data should be a slice of map, or a custom struct slice
// that your template can iterate over
func (sc *SudoController) SudoPanel(c *gin.Context) {
	meetsData, _ := services.LoadMeetCredentials()

	// Gather occupancy across meets
	var allOccupancies []map[string]interface{}
	for _, meet := range meetsData.Meets {
		occ := sc.OccupancyService.GetOccupancy(meet.Name)
		allOccupancies = append(allOccupancies, map[string]interface{}{
			"meetName":   meet.Name,
			"leftUser":   occ.LeftUser,
			"centerUser": occ.CenterUser,
			"rightUser":  occ.RightUser,
		})
	}

	// Pull the superuser logo if it exists
	var sudoLogo string
	if meetsData.Superuser != nil {
		sudoLogo = meetsData.Superuser.Logo
	}

	c.HTML(http.StatusOK, "sudo.html", gin.H{
		"meetsOccupancy": allOccupancies,
		"SudoLogo":       sudoLogo,
	})
}

// ForceVacateRefForAnyMeet forcibly vacates a referee from some meet
func (sc *SudoController) ForceVacateRefForAnyMeet(c *gin.Context) {
	meetName := c.PostForm("meetName")
	position := c.PostForm("position")

	// do minimal validation
	if meetName == "" || position == "" {
		c.String(http.StatusBadRequest, "Missing meetName or position")
		return
	}

	occ := sc.OccupancyService.GetOccupancy(meetName)
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
		c.String(http.StatusBadRequest, "Position is already vacant")
		return
	}

	// remove occupant from occupancy
	if err := sc.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// optionally remove occupant from ActiveUsers
	ActiveUsersMu.Lock()
	delete(ActiveUsers, occupant)
	ActiveUsersMu.Unlock()

	// broadcast update
	logger.Info.Printf("[ForceVacateRefForAnyMeet] Superuser forcibly removed %s from meet=%s pos=%s",
		occupant, meetName, position)
	go sc.broadcastOccupancy(meetName)

	// redirect or return success
	c.Redirect(http.StatusFound, "/sudo")
}

// ForceLogoutMeetDirector forcibly logs out a meet director
func (sc *SudoController) ForceLogoutMeetDirector(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		c.String(http.StatusBadRequest, "username is required")
		return
	}

	// remove them from ActiveUsers
	ActiveUsersMu.Lock()
	if _, exists := ActiveUsers[username]; !exists {
		ActiveUsersMu.Unlock()
		c.String(http.StatusNotFound, "No such user is logged in")
		return
	}

	delete(ActiveUsers, username)
	ActiveUsersMu.Unlock()

	logger.Info.Printf("[ForceLogoutMeetDirector] Superuser forcibly logged out user=%s", username)
	c.Redirect(http.StatusFound, "/sudo")
}

// RestartAndClearMeet forcibly resets an unhealthy meet instance
func (sc *SudoController) RestartAndClearMeet(c *gin.Context) {
	meetName := c.PostForm("meetName")
	if meetName == "" {
		c.String(http.StatusBadRequest, "meetName is required")
		return
	}

	// clear the meet state from the unified state
	websocket.ClearMeetState(meetName)

	// reset occupancy
	sc.OccupancyService.ResetOccupancyForMeet(meetName)

	logger.Info.Printf("[RestartAndClearMeet] Superuser forcibly reset meet: %s", meetName)
	c.Redirect(http.StatusFound, "/sudo")
}

// broadcastOccupancy is just a re-use of your existing logic from PositionController
func (sc *SudoController) broadcastOccupancy(meetName string) {
	occ := sc.OccupancyService.GetOccupancy(meetName)
	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}
	websocket.SendBroadcastMessage(mustMarshal(msg))
}

// mustMarshal is a tiny helperSudoPanel
func mustMarshal(v interface{}) []byte {
	bytes, _ := json.Marshal(v)
	return bytes
}

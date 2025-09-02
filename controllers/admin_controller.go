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
/*
If the user is not an admin, they receive an HTTP 401 Unauthorized response.
Requires a meet name to be specified via query parameters or session */
func (ac *AdminController) AdminPanel(c *gin.Context) {
	session := sessions.Default(c)
	adminVal := session.Get("isAdmin")

	logger.Debug.Printf("[AdminPanel] isAdmin from session: %v", adminVal)

	isAdmin, ok := adminVal.(bool)
	if !ok || !isAdmin {
		logger.Warn.Printf("[AdminPanel] Unauthorized access attempt: isAdmin=%v (type=%T) from %s", adminVal, adminVal, c.ClientIP())
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	// retrieve meet name from query parameters or session
	meetName := c.Query("meet")
	if meetName == "" {
		meetName, _ = session.Get("meetName").(string)
	}
	if meetName == "" {
		logger.Warn.Printf("[AdminPanel] Missing meetName for admin session from %s", c.ClientIP())
		c.String(http.StatusBadRequest, "Meet not specified")
		return
	}
	// load meets
	creds, err := services.LoadMeetCredentials()
	if err != nil {
		logger.Error.Printf("[AdminPanel] Failed to load meet credentials for meet=%s: %v", meetName, err)
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
	if logo == "" {
		logger.Warn.Printf("[AdminPanel] No logo found for meet=%s (may indicate missing config)", meetName)
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

// ForceVacate forcibly unassigns a referee from a specified position in a meet.
// It requires admin privileges. If successful, the user is removed from ActiveUsers,
// the seat is vacated, and the change is broadcast to clients.
func (ac *AdminController) ForceVacate(c *gin.Context) {
	session := sessions.Default(c)

	// ensure user is an admin
	isAdmin, ok := session.Get("isAdmin").(bool)
	if !ok || !isAdmin {
		logger.Warn.Printf("[ForceVacate] Unauthorized access attempt from %s", c.ClientIP())
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	meetName := c.PostForm("meetName")
	position := c.PostForm("position")

	// validate input parameters
	if meetName == "" || position == "" {
		logger.Warn.Printf("[ForceVacate] Missing parameters: meetName=%q, position=%q from %s", meetName, position, c.ClientIP())
		c.String(http.StatusBadRequest, "Missing parameters")
		return
	}

	// ensure `GetOccupancy` returns a valid object
	occupancy := ac.OccupancyService.GetOccupancy(meetName)
	if occupancy == (services.Occupancy{}) { // Check if meet exists
		logger.Warn.Printf("[ForceVacate] No occupancy found for meet=%s", meetName)
		c.String(http.StatusNotFound, "Meet not found")
		return
	}

	var occupant string
	switch position {
	case "left":
		occupant = occupancy.LeftUser
	case "center":
		occupant = occupancy.CenterUser
	case "right":
		occupant = occupancy.RightUser
	default:
		logger.Warn.Printf("[ForceVacate] Invalid position: %q in meet=%s", position, meetName)
		c.String(http.StatusBadRequest, "Invalid position")
		return
	}

	// ensure there is an occupant before vacating
	if occupant == "" {
		logger.Warn.Printf("[ForceVacate] Attempt to vacate already-empty %s position in meet=%s", position, meetName)
		c.String(http.StatusBadRequest, "Position already vacant")
		return
	}

	// remove user from the active list
	delete(ActiveUsers, occupant)

	// update occupancy state
	if err := ac.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		logger.Error.Printf("[ForceVacate] Error unsetting position=%s for user=%s in meet=%s: %v", position, occupant, meetName, err)
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// ensure WebSocket Broadcast function is called
	ac.PositionController.BroadcastOccupancy(meetName)

	// Log admin force vacate action - keep at INFO level for security auditing
	adminContext := logger.NewHTTPContext("POST", "/force-vacate", c.Request.UserAgent(), c.ClientIP(), http.StatusFound)
	adminContext["meetName"] = meetName
	adminContext["position"] = position
	adminContext["occupant"] = occupant
	adminContext["action"] = "force_vacate"
	logger.LogInfoWithContext(adminContext, "Admin forcibly removed referee from position")

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

	// Log meet reset initiation - keep at INFO level for security auditing
	adminContext := logger.NewHTTPContext("POST", "/reset-instance", c.Request.UserAgent(), c.ClientIP(), http.StatusFound)
	adminContext["meetName"] = meetName
	adminContext["action"] = "reset_meet_start"
	logger.LogInfoWithContext(adminContext, "Admin initiated meet reset")

	// clear all active users so none are "already logged in" for the old meet
	ActiveUsersMu.Lock()
	ActiveUsers = make(map[string]bool)
	ActiveUsersMu.Unlock()

	// reset occupancy for the old meet
	ac.OccupancyService.ResetOccupancyForMeet(meetName)
	ac.PositionController.BroadcastOccupancy(meetName)

	// Log meet reset completion - keep at INFO level for security auditing
	adminContext["action"] = "reset_meet_complete"
	logger.LogInfoWithContext(adminContext, "Admin meet reset completed successfully")

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
	isAdmin := session.Get("isAdmin") // ensure user is an admin

	if isAdmin == nil || isAdmin != true {
		logger.Warn.Printf("[ForceLogout] Unauthorized attempt from %s", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin privileges required"})
		return
	}

	username := c.PostForm("username")
	if username == "" {
		logger.Warn.Printf("[ForceLogout] Missing username in POST from %s", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing username parameter"})
		return
	}

	// check if user is logged in
	if _, exists := ActiveUsers[username]; !exists {
		logger.Warn.Printf("[ForceLogout] Attempt to logout non-logged-in user=%s by admin from %s", username, c.ClientIP())
		c.JSON(http.StatusNotFound, gin.H{"error": "User not logged in"})
		return
	}

	// remove user from the active list
	delete(ActiveUsers, username)
	// Log admin force logout - keep at INFO level for security auditing
	adminContext := logger.NewHTTPContext("POST", "/force-logout", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
	adminContext["targetUser"] = username
	adminContext["action"] = "force_logout"
	logger.LogInfoWithContext(adminContext, "Admin forcibly logged out user")
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
/*
data should be a slice of map, or a custom struct slice that your template can
iterate over
*/
func (sc *SudoController) SudoPanel(c *gin.Context) {
	meetsData, err := services.LoadMeetCredentials()
	if err != nil {
		logger.Error.Printf("[SudoPanel] Failed to load meet credentials: %v", err)
		c.String(http.StatusInternalServerError, "Failed to load meet data")
		return
	}

	// gather occupancy across meets
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

	// pull the superuser logo if it exists
	var sudoLogo string
	if meetsData.Superuser != nil {
		sudoLogo = meetsData.Superuser.Logo
	}
	if sudoLogo == "" {
		logger.Warn.Println("[SudoPanel] No logo found for superuser – fallback may apply")
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
		logger.Warn.Printf("[ForceVacateRefForAnyMeet] Missing meetName or position from %s", c.ClientIP())
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
		logger.Warn.Printf("[ForceVacateRefForAnyMeet] Invalid position=%q for meet=%s from %s", position, meetName, c.ClientIP())
		c.String(http.StatusBadRequest, "Invalid position")
		return
	}

	if occupant == "" {
		logger.Warn.Printf("[ForceVacateRefForAnyMeet] Attempt to vacate already-empty %s seat in meet=%s from %s", position, meetName, c.ClientIP())
		c.String(http.StatusBadRequest, "Position is already vacant")
		return
	}

	// remove occupant from occupancy
	if err := sc.OccupancyService.UnsetPosition(meetName, position, occupant); err != nil {
		logger.Error.Printf("[ForceVacateRefForAnyMeet] Error vacating %s in %s for user=%s: %v", position, meetName, occupant, err)
		c.String(http.StatusInternalServerError, "Error vacating position: "+err.Error())
		return
	}

	// optionally remove occupant from ActiveUsers
	ActiveUsersMu.Lock()
	delete(ActiveUsers, occupant)
	ActiveUsersMu.Unlock()

	// broadcast update
	logger.Info.Printf("[ForceVacateRefForAnyMeet] Superuser forcibly removed %s from meet=%s pos=%s", occupant, meetName, position)
	go sc.broadcastOccupancy(meetName)

	// redirect or return success
	c.Redirect(http.StatusFound, "/sudo")
}

// ForceLogoutMeetDirector forcibly logs out a meet director
func (sc *SudoController) ForceLogoutMeetDirector(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		logger.Warn.Printf("[ForceLogoutMeetDirector] Missing username in POST from %s", c.ClientIP())
		c.String(http.StatusBadRequest, "username is required")
		return
	}

	// remove them from ActiveUsers
	ActiveUsersMu.Lock()
	if _, exists := ActiveUsers[username]; !exists {
		ActiveUsersMu.Unlock()
		logger.Warn.Printf("[ForceLogoutMeetDirector] Attempt to logout nonexistent user=%s from %s", username, c.ClientIP())
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
		logger.Warn.Printf("[RestartAndClearMeet] Missing meetName in POST from %s", c.ClientIP())
		c.String(http.StatusBadRequest, "meetName is required")
		return
	}

	// clear the meet state from the unified state
	websocket.ClearMeetState(meetName)

	// reset occupancy
	sc.OccupancyService.ResetOccupancyForMeet(meetName)

	logger.Info.Printf("[RestartAndClearMeet] Superuser forcibly reset meet=%s", meetName)
	c.Redirect(http.StatusFound, "/sudo")
}

// broadcastOccupancy is just a re-use of your existing logic from PositionController
func (sc *SudoController) broadcastOccupancy(meetName string) {
	occ := sc.OccupancyService.GetOccupancy(meetName)
	if occ == (services.Occupancy{}) {
		logger.Warn.Printf("[broadcastOccupancy] No occupancy found for meet=%s – skipping broadcast", meetName)
		return
	}

	msg := map[string]interface{}{
		"action":     "occupancyChanged",
		"leftUser":   occ.LeftUser,
		"centerUser": occ.CenterUser,
		"rightUser":  occ.RightUser,
		"meetName":   meetName,
	}

	bytes := mustMarshal(msg)
	if bytes == nil {
		logger.Error.Printf("[broadcastOccupancy] Failed to marshal occupancy message for meet=%s", meetName)
		return
	}

	websocket.SendBroadcastMessage(bytes)
	logger.Info.Printf("[broadcastOccupancy] Broadcasted occupancy for meet=%s", meetName)
}

// mustMarshal is a tiny helperSudoPanel
func mustMarshal(v interface{}) []byte {
	bytes, err := json.Marshal(v)
	if err != nil {
		logger.Error.Printf("[mustMarshal] JSON marshal failed: %v", err)
		return nil
	}
	return bytes
}

package controllers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"go-ref-lights/logger"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"net/http"
)

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

// SudoPanel is an example method in your SudoController
// data should be a slice of map, or a custom struct slice
// that your template can iterate over
func (sc *SudoController) SudoPanel(c *gin.Context) {
	// user := sessions.Default(c).Get("user") // your superuser's name if needed
	meetsData, _ := loadMeetCredsFunc()
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

	c.HTML(http.StatusOK, "sudo.html", gin.H{
		"meetsOccupancy": allOccupancies,
	})
}

// ForceVacateRefForAnyMeet forcibly vacates a referee from some meet.
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

	// optionally remove occupant from activeUsers if you want:
	activeUsersMu.Lock()
	delete(activeUsers, occupant)
	activeUsersMu.Unlock()

	// broadcast update
	logger.Info.Printf("[ForceVacateRefForAnyMeet] Superuser forcibly removed %s from meet=%s pos=%s",
		occupant, meetName, position)
	go sc.broadcastOccupancy(meetName)

	// redirect or return success
	c.Redirect(http.StatusFound, "/sudo")
}

// ForceLogoutMeetDirector forcibly logs out a meet director
// In your app, the meet director is the user who did "admin" login for a meet.
// So you can do a quick check in activeUsers, or you can define a more direct approach.
func (sc *SudoController) ForceLogoutMeetDirector(c *gin.Context) {
	username := c.PostForm("username")
	if username == "" {
		c.String(http.StatusBadRequest, "username is required")
		return
	}

	// remove them from activeUsers, etc.
	activeUsersMu.Lock()
	if _, exists := activeUsers[username]; !exists {
		activeUsersMu.Unlock()
		c.String(http.StatusNotFound, "No such user is logged in")
		return
	}
	delete(activeUsers, username)
	activeUsersMu.Unlock()

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

	// 1) Clear the meet state from the unified state
	websocket.ClearMeetState(meetName)

	// 2) Reset occupancy
	sc.OccupancyService.ResetOccupancyForMeet(meetName)

	// 3) Optionally remove all users from activeUsers who are in that meet
	//    This is optional. If your logic needs to track which user belongs to which meet,
	//    you might do that in a specialized structure. For a minimal approach:
	//      (You might not actually track each user’s meet, so do what fits your design)
	//
	//    for userName := range activeUsers {
	//      // if userName is a ref or admin of meetName => forcibly remove
	//    }

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

// mustMarshal is a tiny helper
func mustMarshal(v interface{}) []byte {
	bytes, _ := json.Marshal(v)
	return bytes
}

// Package controllers
// file: controllers/referee_controller.go
package controllers

//import (
//	"fmt"
//	"github.com/gin-contrib/sessions"
//	"github.com/gin-gonic/gin"
//	"net/http"
//)
//
//func JoinRefereePosition(c *gin.Context) {
//	meetName := c.Query("meet")
//	pos := c.Query("pos")
//
//	// 1) Validate meet
//	occupancy := OccupancyService.GetOccupancy(meetName)
//	if occupancy == nil {
//		c.String(http.StatusBadRequest, "Meet not found")
//		return
//	}
//
//	// 2) Check if position is free
//	occupant := occupancy.GetUserAtPosition(pos)
//	if occupant != "" {
//		// position is occupied
//		c.HTML(http.StatusForbidden, "error.html", gin.H{
//			"Error": fmt.Sprintf("Position %s is occupied!", pos),
//		})
//		return
//	}
//
//	// 3) Assign occupant
//	// The identity of the user could be ephemeral. Possibly we store a random user ID.
//	userID := generateRefID()
//	if err := occupancyService.SetPosition(meetName, pos, userID); err != nil {
//		...
//	}
//
//	// store in session or store in ephemeral cookie
//	session := sessions.Default(c)
//	session.Set("refPosition", pos)
//	session.Set("user", userID)
//	session.Set("meetName", meetName)
//	session.Save()
//
//	// redirect to "referee view" for that pos
//	switch pos {
//	case "left":
//		c.Redirect(http.StatusFound, "/left")
//	case "center":
//		c.Redirect(http.StatusFound, "/center")
//		...
//	}
//}

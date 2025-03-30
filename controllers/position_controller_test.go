//go:build unit
// +build unit

package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

// reset globals if needed (ActiveUsers, etc.)
func resetPositionControllerGlobals() {
	ActiveUsersMu = sync.RWMutex{}
	ActiveUsers = make(map[string]bool)
}

// TestNewPositionController ensures the constructor sets up OccupancyService
func TestNewPositionController(t *testing.T) {
	mockOcc := new(MockOccupancyService)
	pc := NewPositionController(mockOcc)
	assert.NotNil(t, pc)
	assert.Equal(t, mockOcc, pc.OccupancyService)
}

// ----------------------------------------------------------------------------
// Test ClaimPosition
// ----------------------------------------------------------------------------

func TestClaimPosition(t *testing.T) {
	resetPositionControllerGlobals()
	mockOcc := new(MockOccupancyService)
	pc := NewPositionController(mockOcc)

	// We'll define a router that posts to "/claim", calling pc.ClaimPosition
	router := setupTestRouter(t)
	router.POST("/claim", pc.ClaimPosition)

	t.Run("No user or no meet => redirect /login", func(t *testing.T) {
		// no user in session => immediate redirect
		req := createPostRequest("/claim", map[string]string{
			"position": "left",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login", w.Result().Header.Get("Location"))
	})

	t.Run("SetPosition fails => 403 forbidden", func(t *testing.T) {
		// Put user + meetName in session, but mock setPosition -> error
		ck := SetSession(router, "/sessFail", map[string]interface{}{
			"user":     "someone@example.com",
			"meetName": "someMeet",
		})

		// expect a failing call
		mockOcc.On("SetPosition", "someMeet", "left", "someone@example.com").
			Return(fmt.Errorf("seat taken")).
			Once()

		// no occupancy broadcast expectation => we won't get that far
		req := createPostRequest("/claim", map[string]string{"position": "left"})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Seat is already taken or invalid")

		mockOcc.AssertExpectations(t)
	})
}

// ----------------------------------------------------------------------------
// Test VacatePosition
// ----------------------------------------------------------------------------

func TestVacatePosition(t *testing.T) {
	resetPositionControllerGlobals()
	mockOcc := new(MockOccupancyService)
	pc := NewPositionController(mockOcc)

	router := setupTestRouter(t)
	// We'll do a GET or POST route to call pc.VacatePosition, your code uses c.Redirect
	router.GET("/vacate", pc.VacatePosition)

	t.Run("No user or no meet => redirect /index", func(t *testing.T) {
		// session is empty => immediate redirect
		req, _ := http.NewRequest("GET", "/vacate", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/logout?reason=vacate", w.Result().Header.Get("Location"))
	})
}

// ----------------------------------------------------------------------------
// Test BroadcastOccupancy (direct call)
// ----------------------------------------------------------------------------

func TestBroadcastOccupancy(t *testing.T) {
	resetPositionControllerGlobals()

	// We'll stub out the occupancy call
	mockOcc := new(MockOccupancyService)
	pc := NewPositionController(mockOcc)

	// You might want to mock out websocket.SendBroadcastMessage, but that's in a separate package.
	// We'll just ensure .GetOccupancy is called, and no panic occurs.

	mockOcc.On("GetOccupancy", "someMeet").Return(services.Occupancy{
		LeftUser:   "LeftRef",
		CenterUser: "CenterRef",
		RightUser:  "RightRef",
	}).Once()

	// call it
	pc.BroadcastOccupancy("someMeet")
	// The goroutine inside doesn't need separate mocks if we do not verify how many times it's called.
	mockOcc.AssertExpectations(t)
}

// ----------------------------------------------------------------------------
// Test GetOccupancyAPI
// ----------------------------------------------------------------------------

func TestGetOccupancyAPI(t *testing.T) {
	resetPositionControllerGlobals()
	mockOcc := new(MockOccupancyService)
	pc := NewPositionController(mockOcc)

	router := setupTestRouter(t)
	router.GET("/api/occupancy", pc.GetOccupancyAPI)

	t.Run("No meet => 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/occupancy", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "No meet selected")
	})

	t.Run("Success => returns JSON", func(t *testing.T) {
		// set meetName in session
		ck := SetSession(router, "/sessOccupancy", map[string]interface{}{
			"meetName": "testMeet",
		})

		mockOcc.On("GetOccupancy", "testMeet").Return(services.Occupancy{
			LeftUser:   "Lefty",
			CenterUser: "CenterMan",
			RightUser:  "RightGuy",
		}).Once()

		req, _ := http.NewRequest("GET", "/api/occupancy", nil)
		req.AddCookie(ck)
		w := performRequest(router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()
		// e.g. {"leftUser":"Lefty","centreUser":"CenterMan","rightUser":"RightGuy"}
		assert.Contains(t, body, `"leftUser":"Lefty"`)
		assert.Contains(t, body, `"centreUser":"CenterMan"`)
		assert.Contains(t, body, `"rightUser":"RightGuy"`)

		mockOcc.AssertExpectations(t)
	})
}

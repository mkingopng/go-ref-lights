//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"sync"
	"testing"

	"go-ref-lights/services"

	"github.com/stretchr/testify/assert"
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

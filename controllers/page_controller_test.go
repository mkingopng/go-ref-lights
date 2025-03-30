//go:build unit
// +build unit

package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// reset global occupant counter + locks
func resetPageControllerGlobals() {
	anonCounterMu = sync.Mutex{}
	anonOccupantCounter = 0

	// Also reset the ActiveUsers map
	ActiveUsersMu = sync.RWMutex{}
	ActiveUsers = make(map[string]bool)

	// Null out occupancyService if needed
	// occupancyService = nil
}

// --------------- Test Health ---------------

func TestHealth(t *testing.T) {
	resetPageControllerGlobals()
	router := setupTestRouter(t)
	router.GET("/health", Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := performRequest(router, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"healthy"`)
}

// --------------- Test Logout ---------------

func TestLogout(t *testing.T) {
	resetPageControllerGlobals()
	mockOcc := new(MockOccupancyService)

	router := setupTestRouter(t)
	// define a route that calls Logout with our mock
	router.GET("/logout", func(c *gin.Context) {
		Logout(c, mockOcc)
	})

	t.Run("Empty user => no removal", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/logout", nil)
		w := performRequest(router, req)
		// with no session user, it warns and redirects to /choose-meet
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/choose-meet", w.Result().Header.Get("Location"))
		// no occupancy calls expected
		mockOcc.AssertExpectations(t)
	})

	t.Run("Admin => calls ResetOccupancyForMeet, removes user from ActiveUsers", func(t *testing.T) {
		// Mark a user in session as "admin"
		// Also store meetName => "someMeet"
		// Then verify it calls mockOcc.ResetOccupancyForMeet

		mockOcc.On("ResetOccupancyForMeet", "someMeet").Return().Once()

		ck := SetSession(router, "/setAdminLogout", map[string]interface{}{
			"user":     "admin1",
			"isAdmin":  true,
			"meetName": "someMeet",
		})

		ActiveUsersMu.Lock()
		ActiveUsers["admin1"] = true
		ActiveUsersMu.Unlock()

		req, _ := http.NewRequest("GET", "/logout", nil)
		req.AddCookie(ck)
		w := performRequest(router, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/choose-meet", w.Result().Header.Get("Location"))

		// check user is removed
		ActiveUsersMu.RLock()
		_, exists := ActiveUsers["admin1"]
		ActiveUsersMu.RUnlock()
		assert.False(t, exists)

		mockOcc.AssertExpectations(t)
	})

	t.Run("Normal user => UnsetPosition, remove user", func(t *testing.T) {
		resetPageControllerGlobals()
		mockOcc.On("UnsetPosition", "someMeet", "left", "someUser").Return(nil).Once()

		router2 := setupTestRouter(t)
		router2.GET("/logout", func(c *gin.Context) {
			Logout(c, mockOcc)
		})

		ck := SetSession(router2, "/setNormalLogout", map[string]interface{}{
			"user":        "someUser",
			"refPosition": "left",
			"meetName":    "someMeet",
			"isAdmin":     false,
		})

		ActiveUsersMu.Lock()
		ActiveUsers["someUser"] = true
		ActiveUsersMu.Unlock()

		req, _ := http.NewRequest("GET", "/logout", nil)
		req.AddCookie(ck)
		w := performRequest(router2, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/choose-meet", w.Result().Header.Get("Location"))

		// user removed
		ActiveUsersMu.RLock()
		_, found := ActiveUsers["someUser"]
		ActiveUsersMu.RUnlock()
		assert.False(t, found)

		mockOcc.AssertExpectations(t)
	})
}

// --------------- Test Index (minimal) ---------------

func TestIndex(t *testing.T) {
	resetPageControllerGlobals()

	// We skip references to LoadMeetCredentials by not testing the success path
	// We'll just confirm "no meet => redirect" works, to avoid undefined references
	router := setupTestRouter(t)
	router.GET("/index", Index)

	t.Run("No meet => 302 => /set-meet", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/index", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/set-meet", w.Result().Header.Get("Location"))
	})

	// If you did have the needed references for a "success" path,
	// you'd set session "meetName" and mock LoadMeetCredentials, etc.
}

// --------------- Test SetConfig ---------------

func TestSetConfig(t *testing.T) {
	resetPageControllerGlobals()

	// call SetConfig with some values
	SetConfig("http://myapp.example.com", "ws://mysocket.example.com")

	// check the globals
	assert.Equal(t, "http://myapp.example.com", ApplicationURL)
	assert.Equal(t, "ws://mysocket.example.com", WebsocketURL)
}

// --------------- Test RefereeHandler ---------------

func TestRefereeHandler(t *testing.T) {
	resetPageControllerGlobals()
	mockOcc := new(MockOccupancyService)

	router := setupTestRouter(t)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOcc)
	})

	t.Run("Successful seat claim => calls SetPosition, updates session => shows left view", func(t *testing.T) {
		// occupant not in session => getNextAnonymousName => "AnonRef001"
		// SetPosition => success => renderLeft

		// Expect mockOcc.SetPosition call
		mockOcc.On("SetPosition", "TestMeet", "left", "AnonRef001").Return(nil).Once()

		req, _ := http.NewRequest("GET", "/referee/TestMeet/left", nil)
		w := performRequest(router, req)
		// 200 => we rendered left.html
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Left ref view for TestMeet")

		mockOcc.AssertExpectations(t)
	})

	t.Run("Seat claim conflict => 409 => seat is taken", func(t *testing.T) {
		resetPageControllerGlobals()

		// occupant => "AnonRef001" again, because counter resets
		mockOcc.On("SetPosition", "TestMeet", "center", "AnonRef001").
			Return(fmt.Errorf("seat taken")).
			Once()

		req, _ := http.NewRequest("GET", "/referee/TestMeet/center", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "already taken.")
	})

	t.Run("Unknown position => 400 => 'Unknown position'", func(t *testing.T) {
		resetPageControllerGlobals()
		// occupant => "AnonRef001"
		// mockOcc.On("SetPosition", ...) success => we skip or just do once
		mockOcc.On("SetPosition", "TestMeet", "foo", "AnonRef001").
			Return(nil).Once()

		req, _ := http.NewRequest("GET", "/referee/TestMeet/foo", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Unknown position: foo")

		mockOcc.AssertExpectations(t)
	})
}

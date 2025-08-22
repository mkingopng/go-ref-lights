//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"sync"
	"testing"

	"go-ref-lights/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ============ Global test-setup ============

// Reset global maps/mutex so each test is isolated
func setupGlobalsForTest() {
	// If your real code uses sync.RWMutex, do this:
	ActiveUsersMu = sync.RWMutex{}
	// If it's just sync.Mutex, do:
	// ActiveUsersMu = sync.Mutex{}

	ActiveUsers = make(map[string]bool)
}

// Create a fresh router each time, registering only the Admin routes we want to test
func newTestRouterAdmin(t *testing.T, occ services.OccupancyServiceInterface, pos PositionControllerInterface) *gin.Engine {
	r := setupTestRouter(t)
	ctrl := NewAdminController(occ, pos)

	r.GET("/admin", ctrl.AdminPanel)
	r.POST("/forceVacate", ctrl.ForceVacate)
	r.POST("/resetInstance", ctrl.ResetInstance)
	r.POST("/forceLogout", ctrl.ForceLogout)
	return r
}

// Similarly, for Sudo routes (excluding SudoPanel, which needs services.Meet/MeetCredentials)
func newTestRouterSudo(t *testing.T, occ services.OccupancyServiceInterface) *gin.Engine {
	r := setupTestRouter(t)
	sc := NewSudoController(occ)

	// We skip sc.SudoPanel because it references services.MeetCredentials
	r.POST("/forceVacateAny", sc.ForceVacateRefForAnyMeet)
	r.POST("/forceLogoutDirector", sc.ForceLogoutMeetDirector)
	r.POST("/restartMeet", sc.RestartAndClearMeet)
	return r
}

// ============ AdminController Tests ============

func TestAdminPanel(t *testing.T) {
	t.Run("Not admin => 401", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		req, _ := http.NewRequest("GET", "/admin", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Admin but no meet => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		// Set isAdmin in session, but no meetName
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req, _ := http.NewRequest("GET", "/admin", nil)
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Meet not specified")
	})

	// We do NOT test the "load meets" or "occupancy" success path here,
	// because that would require the undefined services.MeetCredentials struct.
}

func TestForceVacate(t *testing.T) {
	t.Run("Not admin => 401", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		req := createPostRequest("/forceVacate", map[string]string{
			"meetName": "someMeet",
			"position": "left",
		})
		// no session => not admin
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing parameters => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})

		req := createPostRequest("/forceVacate", map[string]string{})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing parameters")
	})

	t.Run("Meet not found => 404", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		// Return zero struct => "meet not found"
		mockOcc.On("GetOccupancy", "badMeet").Return(services.Occupancy{})

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceVacate", map[string]string{
			"meetName": "badMeet",
			"position": "left",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "Meet not found")

		mockOcc.AssertExpectations(t)
	})

	t.Run("Invalid position => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		// Return a non-zero struct to pass the "meet not found" check
		mockOcc.On("GetOccupancy", "someMeet").Return(services.Occupancy{
			RightUser: "someone",
		})

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceVacate", map[string]string{
			"meetName": "someMeet",
			"position": "badPos",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid position")

		mockOcc.AssertExpectations(t)
	})

	t.Run("Position vacant => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		// Occupancy is not all-zero, but LeftUser is ""
		mockOcc.On("GetOccupancy", "someMeet").Return(services.Occupancy{
			LeftUser:  "",
			RightUser: "someone",
		})

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceVacate", map[string]string{
			"meetName": "someMeet",
			"position": "left",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Position already vacant")

		mockOcc.AssertExpectations(t)
	})

	t.Run("Success => 302", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		// occupant is "LeftRef"
		mockOcc.On("GetOccupancy", "someMeet").
			Return(services.Occupancy{LeftUser: "LeftRef"}).
			Once()
		mockOcc.On("UnsetPosition", "someMeet", "left", "LeftRef").Return(nil).Once()
		mockPos.On("BroadcastOccupancy", "someMeet").Return().Once()

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceVacate", map[string]string{
			"meetName": "someMeet",
			"position": "left",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/admin?meet=someMeet", w.Result().Header.Get("Location"))
		assert.False(t, ActiveUsers["LeftRef"])

		mockOcc.AssertExpectations(t)
		mockPos.AssertExpectations(t)
	})
}

func TestResetInstance(t *testing.T) {
	t.Run("Not admin => 401", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		req := createPostRequest("/resetInstance", map[string]string{
			"meetName": "someMeet",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("No meet => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/resetInstance", map[string]string{})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Meet not specified")
	})

	t.Run("Success => 302", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		mockOcc.On("ResetOccupancyForMeet", "someMeet").Return().Once()
		mockPos.On("BroadcastOccupancy", "someMeet").Return().Once()

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin":  true,
			"meetName": "someMeet",
		})
		req := createPostRequest("/resetInstance", map[string]string{
			"meetName": "someMeet",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/admin?meet=someMeet", w.Result().Header.Get("Location"))
		assert.Empty(t, ActiveUsers)

		mockOcc.AssertExpectations(t)
		mockPos.AssertExpectations(t)
	})
}

func TestForceLogout(t *testing.T) {
	t.Run("Not admin => 401", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		req := createPostRequest("/forceLogout", map[string]string{
			"username": "john",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Missing username => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceLogout", map[string]string{})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing username parameter")
	})

	t.Run("User not logged => 404", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceLogout", map[string]string{
			"username": "nope",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not logged in")
	})

	t.Run("Success => 200", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		mockPos := new(MockPositionController)

		router := newTestRouterAdmin(t, mockOcc, mockPos)
		// Mark john active
		ActiveUsers["john"] = true

		sessionCookie := SetSession(router, "/setAdmin", map[string]interface{}{
			"isAdmin": true,
		})
		req := createPostRequest("/forceLogout", map[string]string{
			"username": "john",
		})
		req.AddCookie(sessionCookie)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "User logged out successfully")
		assert.False(t, ActiveUsers["john"])
	})
}

// ============ SudoController (except SudoPanel) ============

func TestForceLogoutMeetDirector(t *testing.T) {
	t.Run("Missing username => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		router := newTestRouterSudo(t, mockOcc)

		req := createPostRequest("/forceLogoutDirector", map[string]string{})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("User not logged => 404", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		router := newTestRouterSudo(t, mockOcc)

		req := createPostRequest("/forceLogoutDirector", map[string]string{
			"username": "nobody",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Success => 302", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		router := newTestRouterSudo(t, mockOcc)

		ActiveUsers["directorUser"] = true
		req := createPostRequest("/forceLogoutDirector", map[string]string{
			"username": "directorUser",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.False(t, ActiveUsers["directorUser"])
	})
}

func TestRestartAndClearMeet(t *testing.T) {
	t.Run("Missing meet => 400", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		router := newTestRouterSudo(t, mockOcc)

		req := createPostRequest("/restartMeet", map[string]string{})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Success => 302", func(t *testing.T) {
		setupGlobalsForTest()
		mockOcc := new(MockOccupancyService)
		// We'll expect exactly one ResetOccupancyForMeet
		mockOcc.On("ResetOccupancyForMeet", "someMeet").Return().Once()

		router := newTestRouterSudo(t, mockOcc)
		req := createPostRequest("/restartMeet", map[string]string{
			"meetName": "someMeet",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/sudo", w.Result().Header.Get("Location"))

		mockOcc.AssertExpectations(t)
	})
}

// controllers/admin_controller_test.go
//go:build unit
// +build unit

package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

// SetupTestRouter returns a Gin engine with session middleware for testing.
func SetupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))
	return router
}

func TestAdminPanel_Unauthorized(t *testing.T) {
	mockOccupancyService := NewMockOccupancyService()
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := SetupTestRouter(t)
	router.GET("/admin", adminController.AdminPanel)

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminPanel_MissingMeetName(t *testing.T) {
	mockOccupancyService := NewMockOccupancyService()
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := SetupTestRouter(t)
	router.GET("/admin", adminController.AdminPanel)

	// Set session with isAdmin true but an empty meetName.
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"isAdmin":  true,
		"meetName": "",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	req, _ := http.NewRequest("GET", "/admin", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expecting a 400 Bad Request because no meet is specified.
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 if meetName is missing")
}

func TestAdminPanel_Success(t *testing.T) {
	router := SetupTestRouter(t)
	occupancyService := NewMockOccupancyService()
	// Simulate occupancy data for the test meet.
	testOccupancy := services.Occupancy{
		LeftUser:   "referee1@example.com",
		CenterUser: "referee2@example.com",
		RightUser:  "referee3@example.com",
	}
	occupancyService.On("GetOccupancy", "TestMeet").Return(testOccupancy)
	posController := NewPositionController(occupancyService)
	adminCtrl := NewAdminController(occupancyService, posController)
	router.GET("/admin", adminCtrl.AdminPanel)

	// Create request with admin session and meet name in query parameter.
	req, _ := http.NewRequest("GET", "/admin?meet=TestMeet", nil)
	w := httptest.NewRecorder()

	// Create a test context and set session values.
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	session := sessions.Default(c)
	session.Set("isAdmin", true)
	session.Set("meetName", "TestMeet")
	session.Save()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Admin Panel for Meet: TestMeet")
}

func TestResetInstance_Success(t *testing.T) {
	mockOccupancyService := NewMockOccupancyService()
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := SetupTestRouter(t)
	router.POST("/reset-instance", adminController.ResetInstance)

	// Set expectations on the mock.
	mockOccupancyService.
		On("ResetOccupancyForMeet", "TestMeet").
		Return().
		Once()
	mockOccupancyService.
		On("GetOccupancy", "TestMeet").
		Return(services.Occupancy{}).
		Once()

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"isAdmin":  true,
		"meetName": "TestMeet",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	req, _ := http.NewRequest("POST", "/reset-instance", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after resetting instance")
	mockOccupancyService.AssertExpectations(t)
}

func TestResetInstanceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockOccupancyService := NewMockOccupancyService()
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := SetupTestRouter(t)
	router.POST("/admin/reset-instance", adminController.ResetInstance)
	// Set expectations.
	mockOccupancyService.
		On("ResetOccupancyForMeet", "TestMeet").
		Return().
		Once()
	mockOccupancyService.
		On("GetOccupancy", "TestMeet").
		Return(services.Occupancy{}).
		Once()

	t.Run("Admin can reset instance", func(t *testing.T) {
		sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
			"isAdmin":  true,
			"meetName": "TestMeet",
		})
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}

		req, _ := http.NewRequest("POST", "/admin/reset-instance", strings.NewReader("meetName=TestMeet"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code, "Should return 302 redirect to admin panel")
		mockOccupancyService.AssertExpectations(t)
	})

	t.Run("Non-admin cannot reset instance", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/reset-instance", strings.NewReader("meetName=TestMeet"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")
	})
}

func TestActiveUsersHandler_AdminCanSeeActiveUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := SetupTestRouter(t)

	router.GET("/active-users", ActiveUsersHandler)

	// Prepare activeUsers map for test.
	activeUsers["referee1"] = true
	activeUsers["referee2"] = true

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"isAdmin": true,
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	req, _ := http.NewRequest("GET", "/active-users", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string][]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["users"], "referee1")
	assert.Contains(t, response["users"], "referee2")
}

func TestForceVacate_MissingParameters(t *testing.T) {
	router := SetupTestRouter(t)
	occupancyService := NewMockOccupancyService()
	posController := NewPositionController(occupancyService)
	adminCtrl := NewAdminController(occupancyService, posController)
	router.POST("/admin/force-vacate", adminCtrl.ForceVacate)

	req, _ := http.NewRequest("POST", "/admin/force-vacate", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	session := sessions.Default(c)
	session.Set("isAdmin", true)
	session.Save()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing parameters")
}

// You would continue similarly for:
// - ForceVacate: testing invalid position, vacating when already vacant, and success scenario.
// - ResetInstance: testing unauthorized, missing meet, and success path.
// - ForceLogout: testing missing username, user not found, and successful logout.

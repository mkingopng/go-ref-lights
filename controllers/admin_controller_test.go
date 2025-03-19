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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

func TestAdminPanel_Unauthorized(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := setupTestRouter(t)
	router.GET("/admin", adminController.AdminPanel)

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminPanel_MissingMeetName(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := setupTestRouter(t)
	router.GET("/admin", adminController.AdminPanel)

	// set session with isAdmin true but an empty meetName.
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

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 if meetName is missing")
}

func TestResetInstance_Success(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := setupTestRouter(t)
	router.POST("/reset-instance", adminController.ResetInstance)

	// set expectations on the mock.
	mockOccupancyService.
		On("ResetOccupancyForMeet", "TestMeet").
		Return().
		Once()
	mockOccupancyService.
		On("GetOccupancy", "TestMeet").
		Return(services.Occupancy{}).
		Once()

	// set session for admin with meetName "TestMeet".
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

	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := &PositionController{OccupancyService: mockOccupancyService}
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	router := setupTestRouter(t)
	router.POST("/admin/reset-instance", adminController.ResetInstance)
	// set expectation for both ResetOccupancyForMeet and GetOccupancy.
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
	router := setupTestRouter(t)

	router.GET("/active-users", ActiveUsersHandler)

	// prepare activeUsers map for test.
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

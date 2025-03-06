// controllers/admin_controller_test.go

//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

// Setup test router for admin controller
func setupAdminTestRouter(ac *AdminController) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	router.GET("/admin", ac.AdminPanel)
	router.POST("/force-vacate", ac.ForceVacate)
	router.POST("/reset-instance", ac.ResetInstance)

	return router
}

// Test unauthorized AdminPanel access
func TestAdminPanel_Unauthorized(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := new(PositionController)

	adminController := NewAdminController(mockOccupancyService, mockPositionController)
	router := setupAdminTestRouter(adminController)

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminPanel_MissingMeetName(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)
	mockPositionController := new(PositionController)
	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	store := cookie.NewStore([]byte("test-secret"))
	router := gin.Default()
	router.Use(sessions.Sessions("testsession", store))

	router.GET("/admin", adminController.AdminPanel)

	req, _ := http.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", true)
	session.Save()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Should return 400 if meetName is missing")
}

// Test ResetInstance (Success Case)
func TestResetInstance_Success(t *testing.T) {
	mockOccupancyService := new(MockOccupancyService)

	mockPositionController := &PositionController{
		OccupancyService: mockOccupancyService,
	}

	adminController := NewAdminController(mockOccupancyService, mockPositionController)

	store := cookie.NewStore([]byte("test-secret"))
	router := gin.Default()
	router.Use(sessions.Sessions("testsession", store))

	router.POST("/reset-instance", adminController.ResetInstance)

	mockOccupancyService.On("ResetOccupancyForMeet", "TestMeet").Return().Once()
	mockOccupancyService.On("GetOccupancy", "TestMeet").Return(services.Occupancy{}).Once()

	req, _ := http.NewRequest("POST", "/reset-instance", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", true)
	session.Set("meetName", "TestMeet")
	session.Save()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after resetting instance")
	mockOccupancyService.AssertExpectations(t)
}

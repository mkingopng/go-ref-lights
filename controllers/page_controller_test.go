// controllers/page_controller_test.go
//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

type MockOccupancyService struct {
	mock.Mock
}

func (m *MockOccupancyService) UnsetPosition(meetName, position, userEmail string) error {
	return m.Called(meetName, position, userEmail).Error(0)
}

func (m *MockOccupancyService) SetPosition(meetName, position, userEmail string) error {
	return m.Called(meetName, position, userEmail).Error(0)
}

func (m *MockOccupancyService) GetOccupancy(meetName string) services.Occupancy {
	return m.Called(meetName).Get(0).(services.Occupancy)
}

func (m *MockOccupancyService) ResetOccupancyForMeet(meetName string) {
	m.Called(meetName)
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := sessions.NewCookieStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))
	return router
}

func TestHealth(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()
	router.GET("/health", Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestLogout(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()

	mockService := new(MockOccupancyService)
	mockService.On("UnsetPosition", "Test Meet", "center", "user@example.com").Return(nil)

	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user", "user@example.com")
		session.Set("refPosition", "center")
		session.Set("meetName", "Test Meet")
		session.Save()
		c.Next()
	})

	router.GET("/logout", func(c *gin.Context) {
		Logout(c, mockService)
	})

	req, _ := http.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))

	mockService.AssertExpectations(t)
}

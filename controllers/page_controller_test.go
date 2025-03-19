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
	"go-ref-lights/websocket"
)

func TestHealth(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter(t)
	router.GET("/health", Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Update expected response to match new JSON format
	expectedResponse := `{"status":"healthy"}`
	assert.JSONEq(t, expectedResponse, w.Body.String(), "Unexpected response from /health endpoint")
}

func TestLogout(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter(t)

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

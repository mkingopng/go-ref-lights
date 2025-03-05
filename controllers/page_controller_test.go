// file: controllers/page_controller_test.go

//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

// Test Health Check
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

// Test Logout
func TestLogout(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()
	router.GET("/logout", Logout)

	req, _ := http.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// Test Index Page (No Meet Selected)
func TestIndex_NoMeetSelected(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()
	router.GET("/", Index)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/meets", w.Header().Get("Location"))
}

// Test Get QR Code
func TestGetQRCode(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()
	router.GET("/qrcode", GetQRCode)

	req, _ := http.NewRequest("GET", "/qrcode", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
}

// Test PerformLogin (No Meet Selected)
func TestPerformLogin_NoMeetSelected(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter()
	router.GET("/login", PerformLogin)

	req, _ := http.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

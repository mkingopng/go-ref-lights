// file: controllers/page_controller_test.go

package controllers

import (
	"bytes"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ✅ Test Health Check
func TestHealth(t *testing.T) {
	router := setupTestRouter()
	router.GET("/health", Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

// ✅ Test Logout
func TestLogout(t *testing.T) {
	router := setupTestRouter()
	router.GET("/logout", Logout)

	req, _ := http.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// ✅ Test Index Page (No Meet Selected)
func TestIndex_NoMeetSelected(t *testing.T) {
	router := setupTestRouter()
	router.GET("/", Index)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/meets", w.Header().Get("Location"))
}

// ✅ Test Claim Position
func TestClaimPosition(t *testing.T) {
	router := setupTestRouter()
	router.POST("/claim-position", ClaimPosition)

	form := bytes.NewBufferString("position=Left")
	req, _ := http.NewRequest("POST", "/claim-position", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ✅ Attach session middleware properly
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req // ✅ Associate the request with the context

	// ✅ Set session values inside the same request context
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c) // ✅ Ensure session middleware is applied

	session := sessions.Default(c)
	session.Set("user", "testuser")     // ✅ Simulate logged-in user
	session.Set("meetName", "TestMeet") // ✅ Simulate selected meet
	_ = session.Save()

	// ✅ Perform request using the same context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after claiming position")
	assert.Equal(t, "/positions", w.Header().Get("Location"))
}

// ✅ Test Get QR Code
func TestGetQRCode(t *testing.T) {
	router := setupTestRouter()
	router.GET("/qrcode", GetQRCode)

	req, _ := http.NewRequest("GET", "/qrcode", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
}

// ✅ Test PerformLogin (No Meet Selected)
func TestPerformLogin_NoMeetSelected(t *testing.T) {
	router := setupTestRouter()
	router.GET("/login", PerformLogin)

	req, _ := http.NewRequest("GET", "/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

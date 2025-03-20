// controllers/page_controller_test.go
//go:build unit
// +build unit

package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

var mockOccService = new(MockOccupancyService)

// TestHealth tests the Health function
func TestHealth(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter(t)
	router.GET("/health", Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	expectedResponse := `{"status":"healthy"}`
	assert.JSONEq(t, expectedResponse, w.Body.String(), "Unexpected response from /health endpoint")
}

// TestLogout tests the Logout function under various conditions
func TestLogout_NoSession(t *testing.T) {
	websocket.InitTest()
	router := setupTestRouter(t)

	mockService := new(MockOccupancyService)
	router.GET("/logout", func(c *gin.Context) {
		Logout(c, mockService)
	})

	req, _ := http.NewRequest("GET", "/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/set-meet", w.Header().Get("Location"))
	mockService.AssertExpectations(t)
}

// TestLogout tests the Logout function under various conditions
//func TestLogout(t *testing.T) {
//	gin.SetMode(gin.TestMode)
//	r := gin.Default()
//
//	store := cookie.NewStore([]byte("test-secret"))
//	r.Use(sessions.Sessions("testsession", store))
//
//	mockService := new(MockOccupancyService)
//	mockService.On("UnsetPosition", "Test Meet", "center", "user@example.com").Return(nil)
//
//	r.GET("/set-session-logout", func(c *gin.Context) {
//		session := sessions.Default(c)
//		session.Set("user", "user@example.com")
//		session.Set("refPosition", "center")
//		session.Set("meetName", "Test Meet")
//		_ = session.Save()
//		c.String(http.StatusOK, "session set for logout test")
//	})
//
//	r.GET("/logout", func(c *gin.Context) {
//		Logout(c, mockService)
//	})
//
//	req1, _ := http.NewRequest("GET", "/set-session-logout", nil)
//	w1 := httptest.NewRecorder()
//	r.ServeHTTP(w1, req1)
//
//	var logoutCookie *http.Cookie
//	for _, c := range w1.Result().Cookies() {
//		if c.Name == "testsession" {
//			logoutCookie = c
//			break
//		}
//	}
//	if logoutCookie == nil {
//		t.Fatal("Session cookie not found for logout test")
//	}
//
//	req2, _ := http.NewRequest("GET", "/logout", nil)
//	req2.AddCookie(logoutCookie)
//	w2 := httptest.NewRecorder()
//	r.ServeHTTP(w2, req2)
//
//	assert.Equal(t, http.StatusFound, w2.Code)
//	assert.Equal(t, "/set-meet", w2.Header().Get("Location"))
//	mockService.AssertExpectations(t)
//}

// TestIndex_NoMeetSelected tests the Index handler when no meet is selected
func TestIndex_NoMeetSelected(t *testing.T) {
	router := setupTestRouter(t)
	router.GET("/index", Index) // tie /index to the Index handler

	// no session set -> should redirect
	req, _ := http.NewRequest("GET", "/index", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/set-meet", w.Header().Get("Location"))
}

// TestIndex_WithMeetName tests the Index handler when a meet is selected
func Test_WithMeetName(t *testing.T) {
	router := setupTestRouter(t)
	router.GET("/index", Index)

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	req, _ := http.NewRequest("GET", "/index", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TestMeet",
		"Response should contain the meetName 'TestMeet' somewhere (eg in HTML).")
}

// TestLights_NoMeetSelected tests the Lights handler when no meet is selected
func TestLights_NoMeetSelected(t *testing.T) {
	router := setupTestRouter(t)
	router.GET("/lights", Lights)

	req, _ := http.NewRequest("GET", "/lights", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/meets", w.Header().Get("Location"))
}

// TestIndex_WithMeetName tests the Index handler when a meet is selected
func TestIndex_WithMeetName(t *testing.T) {
	router := setupTestRouter(t)
	router.GET("/index", Index)

	// set session with meetName
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	req, _ := http.NewRequest("GET", "/index", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TestMeet",
		"Response should contain the meetName 'TestMeet' somewhere (eg in HTML).")
}

// TestRefereeHandler_Success tests the RefereeHandler function
func TestRefereeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService)
	})
	mockOccService.On("SetPosition", "DemoMeet", "left", "AnonymousReferee").Return(nil).Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "DemoMeet")
	mockOccService.AssertExpectations(t)
}

// TestRefereeHandler_Conflict tests the RefereeHandler function when a conflict occurs
func TestRefereeHandler_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService) // pass directly
	})

	mockOccService.
		On("SetPosition", "DemoMeet", "left", "AnonymousReferee").
		Return(fmt.Errorf("left seat is already occupied")).
		Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already taken")
	mockOccService.AssertExpectations(t)
}

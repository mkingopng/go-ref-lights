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
	"github.com/stretchr/testify/mock"
	"go-ref-lights/models"
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
	assert.Equal(t, "/index", w.Header().Get("Location"))
	mockService.AssertExpectations(t)
}

// fix_me
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
	originalFunc := loadMeetCredsFunc

	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &models.MeetCreds{
			Meets: []models.Meet{
				{Name: "TestMeet", Logo: "test_logo.png"},
			},
		}, nil
	}

	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

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

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK when a valid meet is in session")
	assert.Contains(
		t,
		w.Body.String(),
		"TestMeet",
		"Response should contain the meetName 'TestMeet' in the HTML output",
	)
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

	// Save and override loadMeetCredsFunc
	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &models.MeetCreds{
			Meets: []models.Meet{
				// Provide a meet name that matches our test session
				{Name: "TestMeet", Logo: "test_logo.png"},
			},
		}, nil
	}
	// Restore after test
	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

	// Put "meetName" in the session so the /index route sees we selected "TestMeet"
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	// Now make a GET /index request, simulating a user visiting the main page
	req, _ := http.NewRequest("GET", "/index", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The /index handler should succeed and contain "TestMeet" in the HTML
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK if meetName is valid and loadMeetCredsFunc returns it.")
	assert.Contains(
		t,
		w.Body.String(),
		"TestMeet",
		"Response should contain 'TestMeet' in the HTML output",
	)
}

// TestRefereeHandler_Success tests the RefereeHandler function when it should succeed.
func TestRefereeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	// For this route, the code calls RefereeHandler(..., mockOccService)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService)
	})

	// The occupant tries to claim seat => success => Return nil (no error)
	mockOccService.
		On("SetPosition", "DemoMeet", "left", mock.AnythingOfType("string")).
		Return(nil).
		Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// We expect a 200 response from a successful seat claim
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "DemoMeet")

	mockOccService.AssertExpectations(t)
}

// TestRefereeHandler_Conflict tests the RefereeHandler function when SetPosition should fail.
func TestRefereeHandler_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService)
	})

	// This time, for the first (and only) call, we simulate an already-occupied seat => return error
	mockOccService.
		On("SetPosition", "DemoMeet", "left", mock.AnythingOfType("string")).
		Return(fmt.Errorf("left seat is already taken")).
		Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Because the seat is "already taken", the code returns 409 Conflict
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already taken")

	mockOccService.AssertExpectations(t)
}

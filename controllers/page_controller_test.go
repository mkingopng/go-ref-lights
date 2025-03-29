//go:build unit
// +build unit

// controllers/page_controller_test.go
package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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

	// We still expect a 302 redirect (StatusFound):
	assert.Equal(t, http.StatusFound, w.Code)

	// Instead of expecting "/index", we now expect "/choose-meet":
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))

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

	// save and override loadMeetCredsFunc
	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &models.MeetCreds{
			Meets: []models.Meet{
				// provide a meet name that matches our test session
				{Name: "TestMeet", Logo: "test_logo.png"},
			},
		}, nil
	}
	// restore after test
	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

	// put "meetName" in the session so the /index route sees we selected "TestMeet"
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	if sessionCookie == nil {
		t.Fatal("Session cookie not found")
	}

	// now make a GET /index request, simulating a user visiting the main page
	req, _ := http.NewRequest("GET", "/index", nil)
	req.AddCookie(sessionCookie)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// the /index handler should succeed and contain "TestMeet" in the HTML
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK if meetName is valid and loadMeetCredsFunc returns it.")
	assert.Contains(
		t,
		w.Body.String(),
		"TestMeet",
		"Response should contain 'TestMeet' in the HTML output",
	)
}

// TestRefereeHandler_Success tests the RefereeHandler function when it should succeed
func TestRefereeHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	// for this route, the code calls RefereeHandler(..., mockOccService)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService)
	})

	// the occupant tries to claim seat => success => Return nil (no error)
	mockOccService.
		On("SetPosition", "DemoMeet", "left", mock.AnythingOfType("string")).
		Return(nil).
		Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// we expect a 200 response from a successful seat claim
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "DemoMeet")

	mockOccService.AssertExpectations(t)
}

// TestRefereeHandler_Conflict tests the RefereeHandler function when SetPosition should fail
func TestRefereeHandler_Conflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		RefereeHandler(c, mockOccService)
	})

	// this time, for the first (and only) call, we simulate an already-occupied seat => return error
	mockOccService.
		On("SetPosition", "DemoMeet", "left", mock.AnythingOfType("string")).
		Return(fmt.Errorf("left seat is already taken")).
		Once()

	req, _ := http.NewRequest("GET", "/referee/DemoMeet/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// because the seat is "already taken", the code returns 409 Conflict
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already taken")

	mockOccService.AssertExpectations(t)
}

// mock meet data for testing
var testMeets = models.MeetCreds{
	Meets: []models.Meet{
		{Name: "TestMeet1", Date: "2025-03-10"},
		{Name: "TestMeet2", Date: "2025-04-15"},
	},
}

// Test LoadMeets Success
func TestLoadMeets_Success(t *testing.T) {
	websocket.InitTest()
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return &testMeets, nil }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	meets, err := loadMeetsFunc()
	assert.NoError(t, err, "LoadMeets should not return an error")
	assert.NotNil(t, meets, "Meets should not be nil")
	assert.Len(t, meets.Meets, 2, "There should be two meets")
	assert.Equal(t, "TestMeet1", meets.Meets[0].Name, "First meet should match")
}

// Test LoadMeets Failure (File Not Found)
func TestLoadMeets_FileNotFound(t *testing.T) {
	websocket.InitTest()
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return nil, os.ErrNotExist }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	meets, err := loadMeetsFunc()
	assert.Error(t, err, "LoadMeets should return an error if the file is missing")
	assert.Nil(t, meets, "Meets should be nil on failure")
}

// Test ShowMeets Handler
func TestShowMeets(t *testing.T) {
	websocket.InitTest()
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)

	// attach ShowMeets route
	router.GET("/meets", ShowMeets)

	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return &testMeets, nil }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // restore after test

	req, _ := http.NewRequest("GET", "/meets", nil)
	w := httptest.NewRecorder()

	// perform request
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "ShowMeets should return 200 OK")
	assert.Contains(t, w.Body.String(), "TestMeet1", "Response should contain TestMeet1")
	assert.Contains(t, w.Body.String(), "TestMeet2", "Response should contain TestMeet2")
}

// Test ShowMeets_Failure
func TestShowMeets_Failure(t *testing.T) {
	websocket.InitTest()
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/meets", ShowMeets)

	// simulate error loading meets
	originalLoadMeetsFunc := loadMeetsFunc
	loadMeetsFunc = func() (*models.MeetCreds, error) { return nil, os.ErrNotExist }
	defer func() { loadMeetsFunc = originalLoadMeetsFunc }() // Restore after test

	req, _ := http.NewRequest("GET", "/meets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code, "ShowMeets should return 500 on failure")
	assert.Contains(t, w.Body.String(), "Failed to load meets", "Error message should be returned")
}

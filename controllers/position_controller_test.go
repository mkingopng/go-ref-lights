// file: controllers/position_controller_test.go
package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
)

// mock Occupancy Service
var mockOccupancyService = &services.MockOccupancyService{}
var positionController = NewPositionController(mockOccupancyService)

// setup router for PositionController tests
func setupPositionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// attach handlers
	router.GET("/positions", positionController.ShowPositionsPage)
	router.POST("/claim-position", positionController.ClaimPosition)
	router.POST("/vacate-position", positionController.VacatePosition)
	router.GET("/occupancy", positionController.GetOccupancyAPI)

	return router
}

// test ShowPositionsPage (Redirect when not logged in)
func TestShowPositionsPage_NoUser(t *testing.T) {
	router := setupPositionTestRouter()
	req, _ := http.NewRequest("GET", "/positions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/meets", w.Header().Get("Location"))
}

// ✅ Test ClaimPosition (Successful Claim)
func TestClaimPosition_Success(t *testing.T) {
	router := setupPositionTestRouter()

	form := bytes.NewBufferString("position=left")
	req, _ := http.NewRequest("POST", "/claim-position", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ✅ Attach session middleware properly
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req // ✅ Associate the request with the context

	// ✅ Ensure session middleware is applied
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c)

	// ✅ Set session values inside the request context
	session := sessions.Default(c)
	session.Set("user", "testuser")     // ✅ Simulate logged-in user
	session.Set("meetName", "TestMeet") // ✅ Simulate selected meet
	_ = session.Save()

	// ✅ Perform request using the same context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after claiming position")
	assert.Equal(t, "/left", w.Header().Get("Location"))
}

// ✅ Test VacatePosition (Successful Vacate)
func TestVacatePosition_Success(t *testing.T) {
	router := setupPositionTestRouter()

	req, _ := http.NewRequest("POST", "/vacate-position", nil)

	// ✅ Attach session middleware properly
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req // ✅ Associate the request with the context

	// ✅ Ensure session middleware is applied
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c)

	// ✅ Set session values inside the request context
	session := sessions.Default(c)
	session.Set("user", "testuser")     // Simulate logged-in user
	session.Set("meetName", "TestMeet") // Simulate selected meet
	session.Set("refPosition", "left")  // Simulate a referee position
	_ = session.Save()

	// Perform request using the same context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after vacating position")
	assert.Equal(t, "/positions", w.Header().Get("Location"))
}

// ✅ Test GetOccupancyAPI
func TestGetOccupancyAPI_Success(t *testing.T) {
	router := setupPositionTestRouter()

	req, _ := http.NewRequest("GET", "/occupancy", nil)

	// ✅ Attach session middleware properly
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req // ✅ Associate the request with the context

	// ✅ Ensure session middleware is applied
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c)

	// ✅ Set session values inside the request context
	session := sessions.Default(c)
	session.Set("meetName", "TestMeet") // ✅ Simulate selected meet
	_ = session.Save()

	// ✅ Perform request using the same context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "leftUser")
	assert.Contains(t, response, "centreUser")
	assert.Contains(t, response, "rightUser")
}

//go:build unit
// +build unit

// file: controllers/position_controller_test.go
package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

var mockOccupancyService = new(services.MockOccupancyService)
var positionController = NewPositionController(mockOccupancyService)

// setup router for PositionController tests
func setupPositionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// attach handlers
	router.POST("/claim-position", positionController.ClaimPosition)
	router.POST("/vacate-position", positionController.VacatePosition)
	router.GET("/occupancy", positionController.GetOccupancyAPI)

	return router
}

// test VacatePosition_Success
func TestVacatePosition_Success(t *testing.T) {
	websocket.InitTest()
	mockOccupancyService = new(services.MockOccupancyService)
	positionController = NewPositionController(mockOccupancyService)

	router := setupPositionTestRouter()
	req, _ := http.NewRequest("POST", "/vacate-position", nil)
	w := httptest.NewRecorder()

	// we expect these two calls from the code:
	mockOccupancyService.On("UnsetPosition", "TestMeet", "left", "testuser").Return(nil).Once()
	mockOccupancyService.On("GetOccupancy", "TestMeet").Return(services.Occupancy{}).Once()

	// setup session
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c)
	session := sessions.Default(c)
	session.Set("user", "testuser")
	session.Set("meetName", "TestMeet")
	session.Set("refPosition", "left")
	_ = session.Save()

	t.Logf("Session refPosition (before request): %v", session.Get("refPosition"))
	// serve the POST /vacate-position request
	router.ServeHTTP(w, req)

	// wait a bit for async broadcast
	time.Sleep(200 * time.Millisecond)
	t.Log("Assertions after VacatePosition execution")

	// we expect a redirect
	assert.Equal(t, http.StatusFound, w.Code, "Should redirect after vacating position")
	assert.Equal(t, "/index", w.Header().Get("Location"))

	// confirm that the broadcast calls mockOccupancyService.GetOccupancy
	mockOccupancyService.AssertCalled(t, "GetOccupancy", "TestMeet")
	mockOccupancyService.AssertExpectations(t)
}

// Test GetOccupancyAPI (Successful GetOccupancy)
func TestGetOccupancyAPI_Success(t *testing.T) {
	websocket.InitTest()
	mockOccupancyService = new(services.MockOccupancyService)
	positionController = NewPositionController(mockOccupancyService)

	router := setupPositionTestRouter()
	req, _ := http.NewRequest("GET", "/occupancy", nil)
	w := httptest.NewRecorder()

	mockOccupancyService.On("GetOccupancy", "TestMeet").Return(services.Occupancy{
		LeftUser:   "user1",
		CenterUser: "",
		RightUser:  "user2",
	}).Once()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	store := cookie.NewStore([]byte("test-secret"))
	sessions.Sessions("testsession", store)(c)

	session := sessions.Default(c)
	session.Set("meetName", "TestMeet")
	_ = session.Save()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "leftUser")
	assert.Contains(t, response, "centreUser")
	assert.Contains(t, response, "rightUser")

	time.Sleep(150 * time.Millisecond)
	mockOccupancyService.AssertExpectations(t)
}

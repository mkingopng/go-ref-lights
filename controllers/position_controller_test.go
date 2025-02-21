// file: controllers/position_controller_test.go

package controllers

import (
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/services"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
)

// MockOccupancyService implements OccupancyServiceInterface for testing
type MockOccupancyService struct{}

func (m *MockOccupancyService) GetOccupancy() services.Occupancy {
	return services.Occupancy{
		LeftUser:   "left@example.com",
		CentreUser: "",
		RightUser:  "right@example.com",
	}
}

// Ensure SetPosition() matches OccupancyServiceInterface
func (m *MockOccupancyService) SetPosition(position, userEmail string) error {
	if position == "left" {
		return errors.New("Left position is already taken")
	}
	return nil
}

// Add ResetOccupancy() method to satisfy the interface
func (m *MockOccupancyService) ResetOccupancy() {
	// Simulate resetting all referee positions
}

// Create a new router with mock dependencies
func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Inject mock service
	mockService := &MockOccupancyService{}
	positionController := NewPositionController(mockService)

	// Load templates
	_, filename, _, _ := runtime.Caller(0)
	basepath := filepath.Join(filepath.Dir(filename), "../templates")
	router.LoadHTMLGlob(filepath.Join(basepath, "*.html"))

	// Routes
	router.GET("/positions", positionController.ShowPositionsPage)
	router.POST("/position/claim", positionController.ClaimPosition)

	return router
}

// Test ShowPositionsPage - Redirects unauthenticated users to /login
func TestShowPositionsPage_Unauthenticated(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/positions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// Test ShowPositionsPage - Displays positions for authenticated users
func TestShowPositionsPage_Authenticated(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("GET", "/positions", nil)
	w := httptest.NewRecorder()

	// Create Gin test context & manually set session
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	store := cookie.NewStore([]byte("secret"))
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c) // Apply middleware before calling `sessions.Default(c)`

	session := sessions.Default(c)
	session.Set("user", "test@example.com") // Ensure the user is stored in session
	err := session.Save()
	if err != nil {
		return
	} // Persist session across test requests

	// Run the request through the router
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Left (Occupied by left@example.com)")
	assert.Contains(t, w.Body.String(), "Centre (Available)")
	assert.Contains(t, w.Body.String(), "Right (Occupied by right@example.com)")
}

// Test ClaimPosition - Redirects unauthenticated users to /login
func TestClaimPosition_Unauthenticated(t *testing.T) {
	router := setupRouter()

	req, _ := http.NewRequest("POST", "/position/claim", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// Test ClaimPosition - Prevents double assignment
func TestClaimPosition_AlreadyTaken(t *testing.T) {
	router := setupRouter()

	// Ensure session middleware is applied to the router
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Step 1: First user claims "left"
	svc := &services.OccupancyService{}
	err := svc.SetPosition("left", "existing-user@example.com")
	assert.NoError(t, err, "First user should be able to claim the position")

	// Step 2: Verify the position is correctly assigned
	occ := svc.GetOccupancy()
	assert.Equal(t, "existing-user@example.com", occ.LeftUser, "Position should be assigned to first user")

	// Step 3: Create test request & apply session middleware
	req, _ := http.NewRequest("POST", "/position/claim", nil)
	req.PostForm = map[string][]string{"position": {"left"}}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Apply session store in test context
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c)

	// Set session for authenticated user
	session := sessions.Default(c)
	session.Set("user", "another-user@example.com")
	session.Save()

	// Step 4: Process request
	router.ServeHTTP(w, req)

	// Step 5: Validate response (should return 403 Forbidden)
	assert.Equal(t, http.StatusForbidden, w.Code, "Expected 403 Forbidden for already taken position")
	assert.Contains(t, w.Body.String(), "Left position is already taken", "Expected error message for taken position")
}

// main_test.go

package main

//
//import (
//	"net/http"
//	"net/http/httptest"
//	"strings"
//	"testing"
//
//	"github.com/gin-contrib/sessions"
//	"github.com/gin-contrib/sessions/cookie"
//	"github.com/gin-gonic/gin"
//	"github.com/stretchr/testify/assert"
//	"go-ref-lights/controllers"
//	"go-ref-lights/middleware"
//)
//
//// setupTestRouter initializes a Gin test router
//func setupTestRouter() *gin.Engine {
//	gin.SetMode(gin.TestMode)
//	router := gin.Default()
//
//	// Mock session store
//	store := cookie.NewStore([]byte("secret"))
//	router.Use(sessions.Sessions("testsession", store))
//
//	// Load application configuration
//	controllers.SetConfig("http://localhost:8080", "ws://localhost:8080/referee-updates")
//
//	// Define public routes
//	router.GET("/health", controllers.Health)
//	router.GET("/", controllers.ShowMeets)
//	router.POST("/set-meet", controllers.SetMeetHandler)
//	router.GET("/login", controllers.PerformLogin)
//	router.POST("/login", controllers.LoginHandler)
//	router.GET("/logout", controllers.Logout)
//
//	// Protected routes
//	protected := router.Group("/")
//	protected.Use(middleware.AuthRequired)
//	protected.Use(middleware.PositionRequired())
//
//	protected.GET("/dashboard", controllers.Index)
//	protected.GET("/positions", controllers.ShowPositionsPage)
//
//	return router
//}
//
//// TestHealthCheck verifies that the health check endpoint is available
//func TestHealthCheck(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/health", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Equal(t, "OK", w.Body.String())
//}
//
//// TestShowMeets ensures the meets selection page loads
//func TestShowMeets(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Contains(t, w.Body.String(), "Select a Meet")
//}
//
//// TestLoginEndpoint ensures login page loads correctly
//func TestLoginPage(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/login", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Contains(t, w.Body.String(), "Login")
//}
//
//// TestSetMeetHandler simulates selecting a meet and storing it in a session
//func TestSetMeetHandler(t *testing.T) {
//	router := setupTestRouter()
//
//	formData := "meetName=TestMeet"
//	req, _ := http.NewRequest("POST", "/set-meet", strings.NewReader(formData)) // FIXED!
//	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusFound, w.Code)
//	assert.Equal(t, "/login", w.Header().Get("Location"))
//}
//
//// TestLoginHandler_Success verifies successful login
//func TestLoginHandler_Success(t *testing.T) {
//	router := setupTestRouter()
//
//	formData := `{"username": "testuser", "password": "testpass"}`
//	req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData)) // FIXED!
//	req.Header.Set("Content-Type", "application/json")
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusFound, w.Code)
//	assert.Equal(t, "/dashboard", w.Header().Get("Location"))
//}
//
//// TestLoginHandler_Failure checks invalid login attempts
//func TestLoginHandler_Failure(t *testing.T) {
//	router := setupTestRouter()
//
//	formData := `{"username": "wronguser", "password": "wrongpass"}`
//	req, _ := http.NewRequest("POST", "/login", strings.NewReader(formData)) // FIXED!
//	req.Header.Set("Content-Type", "application/json")
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusUnauthorized, w.Code)
//	assert.Contains(t, w.Body.String(), "Invalid username or password")
//}
//
//// TestProtectedRoutes ensures unauthenticated users cannot access protected pages
//func TestProtectedRoutes_Unauthenticated(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/dashboard", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusFound, w.Code)
//	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
//}
//
//// TestLogout verifies that the logout process clears the session
//func TestLogout(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/logout", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusFound, w.Code)
//	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
//}
//
//// TestMiddleware_AuthRequired ensures unauthorized users are redirected
//func TestMiddleware_AuthRequired(t *testing.T) {
//	router := setupTestRouter()
//
//	req, _ := http.NewRequest("GET", "/positions", nil)
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusFound, w.Code)
//	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
//}
//
//// TestLoggingEndpoint ensures log messages are correctly received and processed
//func TestLoggingEndpoint(t *testing.T) {
//	router := setupTestRouter()
//
//	logEntry := `{"message": "Test log", "level": "info"}`
//	req, _ := http.NewRequest("POST", "/log", strings.NewReader(logEntry)) // FIXED!
//	req.Header.Set("Content-Type", "application/json")
//
//	w := httptest.NewRecorder()
//	router.ServeHTTP(w, req)
//
//	assert.Equal(t, http.StatusOK, w.Code)
//}

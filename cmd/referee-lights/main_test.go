//go:build unit
// +build unit

// cmd/referee-lights/main_test.go
package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

// testSetupTemplates creates a temporary templates directory with a fake template file
func testSetupTemplates(t *testing.T) string {
	t.Log("[testSetupTemplates] Creating temporary templates directory")
	tempDir, err := ioutil.TempDir("", "templates")

	if err != nil {
		t.Fatalf("Failed to create temp templates directory: %v", err)
	}

	dummyFile := filepath.Join(tempDir, "dummy.html")
	content := []byte("<html><body>Dummy Template</body></html>")

	if err := ioutil.WriteFile(dummyFile, content, 0644); err != nil {
		t.Fatalf("Failed to write dummy template: %v", err)
	}

	t.Cleanup(func() {
		t.Log("[testSetupTemplates] Cleaning up temporary templates directory")
		os.RemoveAll(tempDir)
	})

	t.Logf("[testSetupTemplates] Templates directory set to: %s\n", tempDir)
	return tempDir
}

// testSetupRouter creates a minimal Gin router for testing
func testSetupRouter(t *testing.T, templatesDir, env string) *gin.Engine {
	t.Log("[testSetupRouter] Setting up test router")
	router := gin.Default()

	// middleware to log incoming requests and paths
	router.Use(func(c *gin.Context) {
		t.Logf("[Middleware] Incoming request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		t.Logf("[Middleware] Completed request: %s %s", c.Request.Method, c.Request.URL.Path)
	})

	// public paths that should not be protected
	publicPaths := map[string]bool{
		"/health":   true,
		"/log":      true,
		"/":         true,
		"/set-meet": true,
		"/login":    true,
	}

	// protection middleware
	router.Use(func(c *gin.Context) {
		if publicPaths[c.Request.URL.Path] {
			t.Logf("[Protection Middleware] Public path %s accessed", c.Request.URL.Path)
			c.Next()
			return
		}

		// for testing, check if "meetName" is set in the Gin context
		if meetName, exists := c.Get("meetName"); !exists || meetName == "" {
			t.Logf("[Protection Middleware] No meetName in context for path %s; redirecting", c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		t.Logf("[Protection Middleware] meetName present for path %s", c.Request.URL.Path)
		c.Next()
	})

	// load templates
	t.Logf("[testSetupRouter] Loading templates from: %s", filepath.Join(templatesDir, "*.html"))
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// define public routes
	router.GET("/health", func(c *gin.Context) {
		t.Log("[Route /health] Health check called")
		c.String(http.StatusOK, "OK")
	})
	router.POST("/log", func(c *gin.Context) {
		t.Log("[Route /log] Log endpoint called")
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}

		if err := c.ShouldBindJSON(&payload); err != nil {
			t.Logf("[Route /log] Error binding JSON: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		t.Logf("[Route /log] Received log: level=%s, message=%s", payload.Level, payload.Message)
		c.Status(http.StatusOK)
	})
	router.GET("/", func(c *gin.Context) {
		t.Log("[Route /] Meet selection page accessed")
		c.String(http.StatusOK, "Meet Selection Page")
	})
	router.POST("/set-meet", func(c *gin.Context) {
		t.Log("[Route /set-meet] set-meet called; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
	})
	router.GET("/login", func(c *gin.Context) {
		t.Log("[Route /login] Login page accessed")
		c.String(http.StatusOK, "Login Page")
	})
	router.POST("/login", func(c *gin.Context) {
		t.Log("[Route /login] Login POST received; redirecting to /index")
		c.Redirect(http.StatusFound, "/index")
	})
	router.GET("/logout", func(c *gin.Context) {
		t.Log("[Route /logout] Logout called; redirecting to /")
		c.Redirect(http.StatusFound, "/")
	})

	// protected route
	router.GET("/index", func(c *gin.Context) {
		t.Log("[Route /index] Protected dashboard accessed")
		c.String(http.StatusOK, "Dashboard")
	})

	t.Log("[testSetupRouter] Test router setup complete")
	return router
}

// TestMainSetup resets global state before each test
func TestMainSetup(t *testing.T) {
	t.Log("[TestMainSetup] Resetting global state")
	websocket.InitTest() // reset any global state in the websocket package
}

// TestMainEndpoint tests the main endpoint
func TestHealthEndpoint(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(t, templatesDir, "development")

	req, _ := http.NewRequest("GET", "/health", nil)
	t.Log("[TestHealthEndpoint] Sending GET /health request")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	t.Logf("[TestHealthEndpoint] Received status code: %d, body: %s", resp.Code, resp.Body.String())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "OK", resp.Body.String())
}

// TestLogEndpoint tests the log endpoint
func TestLogEndpoint(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(t, templatesDir, "development")

	jsonPayload := `{"message": "Test log", "level": "info"}`
	req, _ := http.NewRequest("POST", "/log", strings.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	t.Log("[TestLogEndpoint] Sending POST /log with payload:", jsonPayload)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	t.Logf("[TestLogEndpoint] Received status code: %d", resp.Code)
	// expect HTTP 200 since /log is public.
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestProtectedRouteRedirect(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(t, templatesDir, "development")

	// do not set "meetName" in the context so the protected middleware will trigger
	req, _ := http.NewRequest("GET", "/index", nil)
	t.Log("[TestProtectedRouteRedirect] Sending GET /index without meetName")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	t.Logf("[TestProtectedRouteRedirect] Received status code: %d, Location header: %s", resp.Code, resp.Header().Get("Location"))
	// our middleware should redirect to "/" if "meetName" is not set
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "/", resp.Header().Get("Location"))
}

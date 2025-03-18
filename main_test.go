// main_test.go
//go:build unit
// +build unit

package main

import (
	"io/ioutil"
	"log"
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

// testSetupTemplates creates a temporary templates directory with a dummy template file.
func testSetupTemplates(t *testing.T) string {
	log.Println("[testSetupTemplates] Creating temporary templates directory")
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
		log.Println("[testSetupTemplates] Cleaning up temporary templates directory")
		os.RemoveAll(tempDir)
	})
	log.Printf("[testSetupTemplates] Templates directory set to: %s\n", tempDir)
	return tempDir
}

// testSetupRouter creates a minimal Gin router for testing.
func testSetupRouter(templatesDir, env string) *gin.Engine {
	log.Println("[testSetupRouter] Setting up test router")
	router := gin.Default()

	// Middleware to log incoming requests and paths.
	router.Use(func(c *gin.Context) {
		log.Printf("[Middleware] Incoming request: %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
		log.Printf("[Middleware] Completed request: %s %s", c.Request.Method, c.Request.URL.Path)
	})

	// Public paths that should not be protected.
	publicPaths := map[string]bool{
		"/health":   true,
		"/log":      true,
		"/":         true,
		"/set-meet": true,
		"/login":    true,
	}
	// Protection middleware.
	router.Use(func(c *gin.Context) {
		if publicPaths[c.Request.URL.Path] {
			log.Printf("[Protection Middleware] Public path %s accessed", c.Request.URL.Path)
			c.Next()
			return
		}
		// For testing, check if "meetName" is set in the Gin context.
		if meetName, exists := c.Get("meetName"); !exists || meetName == "" {
			log.Printf("[Protection Middleware] No meetName in context for path %s; redirecting", c.Request.URL.Path)
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		log.Printf("[Protection Middleware] meetName present for path %s", c.Request.URL.Path)
		c.Next()
	})

	// Load templates.
	log.Printf("[testSetupRouter] Loading templates from: %s", filepath.Join(templatesDir, "*.html"))
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// Define public routes.
	router.GET("/health", func(c *gin.Context) {
		log.Println("[Route /health] Health check called")
		c.String(http.StatusOK, "OK")
	})
	router.POST("/log", func(c *gin.Context) {
		log.Println("[Route /log] Log endpoint called")
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			log.Printf("[Route /log] Error binding JSON: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}
		log.Printf("[Route /log] Received log: level=%s, message=%s", payload.Level, payload.Message)
		c.Status(http.StatusOK)
	})
	router.GET("/", func(c *gin.Context) {
		log.Println("[Route /] Meet selection page accessed")
		c.String(http.StatusOK, "Meet Selection Page")
	})
	router.POST("/set-meet", func(c *gin.Context) {
		log.Println("[Route /set-meet] set-meet called; redirecting to /login")
		c.Redirect(http.StatusFound, "/login")
	})
	router.GET("/login", func(c *gin.Context) {
		log.Println("[Route /login] Login page accessed")
		c.String(http.StatusOK, "Login Page")
	})
	router.POST("/login", func(c *gin.Context) {
		log.Println("[Route /login] Login POST received; redirecting to /index")
		c.Redirect(http.StatusFound, "/index")
	})
	router.GET("/logout", func(c *gin.Context) {
		log.Println("[Route /logout] Logout called; redirecting to /")
		c.Redirect(http.StatusFound, "/")
	})

	// Protected route.
	router.GET("/index", func(c *gin.Context) {
		log.Println("[Route /index] Protected dashboard accessed")
		c.String(http.StatusOK, "Dashboard")
	})

	log.Println("[testSetupRouter] Test router setup complete")
	return router
}

// TestMainSetup resets global state before each test.
func TestMainSetup(t *testing.T) {
	log.Println("[TestMainSetup] Resetting global state")
	websocket.InitTest() // Reset any global state in the websocket package.
}

func TestHealthEndpoint(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(templatesDir, "development")

	req, _ := http.NewRequest("GET", "/health", nil)
	log.Println("[TestHealthEndpoint] Sending GET /health request")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	log.Printf("[TestHealthEndpoint] Received status code: %d, body: %s", resp.Code, resp.Body.String())
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "OK", resp.Body.String())
}

func TestLogEndpoint(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(templatesDir, "development")

	jsonPayload := `{"message": "Test log", "level": "info"}`
	req, _ := http.NewRequest("POST", "/log", strings.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	log.Println("[TestLogEndpoint] Sending POST /log with payload:", jsonPayload)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	log.Printf("[TestLogEndpoint] Received status code: %d", resp.Code)
	// Expect HTTP 200 since /log is public.
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestProtectedRouteRedirect(t *testing.T) {
	TestMainSetup(t)
	gin.SetMode(gin.TestMode)
	templatesDir := testSetupTemplates(t)
	router := testSetupRouter(templatesDir, "development")

	// Do not set "meetName" in the context so the protected middleware will trigger.
	req, _ := http.NewRequest("GET", "/index", nil)
	log.Println("[TestProtectedRouteRedirect] Sending GET /index without meetName")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	log.Printf("[TestProtectedRouteRedirect] Received status code: %d, Location header: %s", resp.Code, resp.Header().Get("Location"))
	// Our middleware should redirect to "/" if "meetName" is not set.
	assert.Equal(t, http.StatusFound, resp.Code)
	assert.Equal(t, "/", resp.Header().Get("Location"))
}

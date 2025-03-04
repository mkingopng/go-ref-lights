// main_test.go
package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// setupTestTemplates creates a temporary templates directory with a dummy file.
// Given: a need for a valid templates directory for HTML template loading.
func setupTestTemplates(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "templates")
	if err != nil {
		t.Fatalf("Failed to create temp templates directory: %v", err)
	}

	// Create a dummy template file so that router.LoadHTMLGlob does not fail.
	dummyFile := filepath.Join(tempDir, "dummy.html")
	content := []byte("<html><body>Dummy Template</body></html>")
	if err := ioutil.WriteFile(dummyFile, content, 0644); err != nil {
		t.Fatalf("Failed to write dummy template: %v", err)
	}

	// Clean up the temporary directory after test finishes.
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

// TestHealthEndpoint tests the /health endpoint.
// Given: A router with the health endpoint registered.
// When: A GET request is made to /health.
// Then: It should return HTTP 200 and the expected content.
func TestHealthEndpoint(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	templatesDir := setupTestTemplates(t)
	router := gin.Default()

	// Load templates as done in main.go
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// For testing purposes, we simulate the health endpoint handler.
	// In your actual application, this would be controllers.Health.
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// When: A GET request is sent to /health.
	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Then: Verify the response.
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.Code)
	}
	if resp.Body.String() != "OK" {
		t.Errorf("Expected response body 'OK', got %q", resp.Body.String())
	}
}

// TestLogEndpoint tests the /log endpoint.
// Given: A router with the /log endpoint that processes JSON log payloads.
// When: A valid JSON payload is posted to /log.
// Then: It should return HTTP 200.
func TestLogEndpoint(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Simulate the /log route as in main.go.
	router.POST("/log", func(c *gin.Context) {
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}
		// When: Binding the JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		// Here you would normally log the message based on level.
		// For testing, simply return OK.
		c.Status(http.StatusOK)
	})

	// When: Sending a valid log payload.
	jsonPayload := `{"message": "Test log", "level": "info"}`
	req, _ := http.NewRequest("POST", "/log", strings.NewReader(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Then: Expect a 200 OK response.
	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200 but got %d", resp.Code)
	}
}

// TestProtectedRouteRedirect tests the middleware that requires a session variable.
// Given: A router with session middleware and a custom middleware checking for "meetName".
// When: A request is made to a protected route without "meetName" set.
// Then: The user should be redirected (HTTP 302) to the meet selection ("/").
func TestProtectedRouteRedirect(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Configure a dummy session store.
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mySession", store))

	// Custom middleware that simulates redirection if "meetName" is not in session.
	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	})

	// Protected route simulation.
	router.GET("/dashboard", func(c *gin.Context) {
		c.String(http.StatusOK, "Dashboard")
	})

	// When: A GET request is sent to /dashboard without a valid session.
	req, _ := http.NewRequest("GET", "/dashboard", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Then: Verify that the response is a redirection to "/".
	if resp.Code != http.StatusFound {
		t.Errorf("Expected HTTP status %d for redirection, got %d", http.StatusFound, resp.Code)
	}
	if location := resp.Header().Get("Location"); location != "/" {
		t.Errorf("Expected redirection to '/', got %s", location)
	}
}

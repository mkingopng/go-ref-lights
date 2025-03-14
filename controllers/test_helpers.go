//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"go-ref-lights/websocket"
)

// setupTestRouter creates a new Gin engine with session middleware and dummy HTML templates.
// It also initializes the websocket package for tests.
func setupTestRouter(t *testing.T) *gin.Engine {
	websocket.InitTest()
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set up sessions with cookie store.
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Create minimal templates to avoid panics during testing.
	tmpDir := t.TempDir()
	if err := createDummyTemplates(tmpDir); err != nil {
		t.Fatalf("Failed to create dummy templates: %v", err)
	}

	// Use filepath.Join for cross-platform compatibility.
	router.LoadHTMLGlob(filepath.Join(tmpDir, "*.html"))

	return router
}

// createDummyTemplates writes a set of minimal HTML templates to the provided directory.
func createDummyTemplates(dir string) error {
	templates := map[string]string{
		"choose_meet.html": `<html><body>{{.}}</body></html>`,
		"login.html":       `<html><body>{{.}}</body></html>`,
		"positions.html":   `<html><body>{{.}}</body></html>`,
		"index.html":       `<html><body>{{.}}</body></html>`,
	}

	for name, content := range templates {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

// SetSession sets the given key/value pairs in the session using a helper route
// and returns the session cookie that can be attached to subsequent test requests.
func SetSession(router *gin.Engine, route string, data map[string]interface{}) *http.Cookie {
	// Create a helper route for setting session values.
	router.GET(route, func(c *gin.Context) {
		session := sessions.Default(c)
		for key, value := range data {
			session.Set(key, value)
		}
		session.Save()
		c.String(http.StatusOK, "session set")
	})

	// Call the helper route.
	req, _ := http.NewRequest("GET", route, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Extract and return the session cookie.
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "testsession" {
			return cookie
		}
	}
	return nil
}

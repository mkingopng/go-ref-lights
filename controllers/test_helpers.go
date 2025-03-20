// file: controllers/test_helpers.go
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
	"golang.org/x/crypto/bcrypt"
)

// setupTestRouter creates a new Gin engine with session middleware and fake HTML templates.
// It also initialises the websocket package for tests.
func setupTestRouter(t *testing.T) *gin.Engine {
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
		"left.html":        `<html><body>Left ref view for {{.meetName}}</body></html>`,
		"center.html":      `<html><body>Center ref view for {{.meetName}}</body></html>`,
		"right.html":       `<html><body>Right ref view for {{.meetName}}</body></html>`,
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
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "session save failed")
			return
		}
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

// hashPassword hashes the given password using bcrypt.
// This helper function is used by tests to prepare expected hashed values.
func hashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic("failed to hash password: " + err.Error())
	}
	return string(hashed)
}

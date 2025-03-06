//go:build unit
// +build unit

package controllers

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"go-ref-lights/websocket"
	"os"
	"path/filepath"
	"testing"
)

// Shared router setup for all controller tests
func setupTestRouter(t *testing.T) *gin.Engine {
	websocket.InitTest()
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set up sessions with cookie store
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Create minimal templates to avoid panics during testing
	tmpDir := t.TempDir()
	err := createDummyTemplates(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create dummy templates: %v", err)
	}

	router.LoadHTMLGlob(tmpDir + "/*.html")

	return router
}

// Helper function to create dummy templates for testing
func createDummyTemplates(dir string) error {
	templates := map[string]string{
		"choose_meet.html": `<html><body>{{.}}</body></html>`,
		"login.html":       `<html><body>{{.}}</body></html>`,
		"positions.html":   `<html><body>{{.}}</body></html>`,
		"index.html":       `<html><body>{{.}}</body></html>`,
	}

	for name, content := range templates {
		err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}

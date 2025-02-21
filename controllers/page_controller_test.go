package controllers

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
)

func TestIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Get absolute path to templates directory
	_, filename, _, _ := runtime.Caller(0) // Get current file path
	basepath := filepath.Join(filepath.Dir(filename), "../templates")

	router.LoadHTMLGlob(filepath.Join(basepath, "*.html")) // Load templates

	router.GET("/", Index)

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Scan To Login") // Ensure expected HTML content exists
}

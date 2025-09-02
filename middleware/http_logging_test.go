package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"go-ref-lights/logger"
)

func TestMain(m *testing.M) {
	// Set test environment
	os.Setenv("ENV", "test")

	// Initialize logger for tests
	if err := logger.InitLogger(); err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	logger.CloseLogger()
	os.Exit(code)
}

func TestHTTPLoggingMiddleware_SkipStaticAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(HTTPLoggingMiddleware())

	router.GET("/static/test.css", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	req := httptest.NewRequest("GET", "/static/test.css", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHTTPLoggingMiddleware_LogErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(HTTPLoggingMiddleware())

	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "server error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHTTPLoggingMiddleware_SkipHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(HTTPLoggingMiddleware())

	router.GET("/heartbeat", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/heartbeat", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthenticationLoggingMiddleware_LoginEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthenticationLoggingMiddleware())

	router.POST("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "logged in")
	})

	body := bytes.NewBufferString("username=test&password=secret")
	req := httptest.NewRequest("POST", "/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShouldSkipLogging(t *testing.T) {
	tests := []struct {
		uri      string
		expected bool
	}{
		{"/static/css/style.css", true},
		{"/favicon.ico", true},
		{"/heartbeat", true},
		{"/login", false},
		{"/api/data", false},
		{"/", false},
	}

	for _, test := range tests {
		result := shouldSkipLogging(test.uri)
		assert.Equal(t, test.expected, result, "URI: %s", test.uri)
	}
}

func TestIsAuthenticationEndpoint(t *testing.T) {
	tests := []struct {
		uri      string
		expected bool
	}{
		{"/login", true},
		{"/logout", true},
		{"/set-meet", true},
		{"/force-my-login", true},
		{"/api/data", false},
		{"/static/css/style.css", false},
		{"/", false},
	}

	for _, test := range tests {
		result := isAuthenticationEndpoint(test.uri)
		assert.Equal(t, test.expected, result, "URI: %s", test.uri)
	}
}

func TestExtractUsernameFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test POST form data
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		username := extractUsernameFromRequest(c, nil)
		c.String(http.StatusOK, username)
	})

	body := bytes.NewBufferString("username=testuser&password=secret")
	req := httptest.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "testuser", w.Body.String())

	// Test query parameters
	req2 := httptest.NewRequest("GET", "/test?username=queryuser", nil)
	w2 := httptest.NewRecorder()

	router.GET("/test", func(c *gin.Context) {
		username := extractUsernameFromRequest(c, nil)
		c.String(http.StatusOK, username)
	})

	router.ServeHTTP(w2, req2)

	assert.Equal(t, "queryuser", w2.Body.String())
}

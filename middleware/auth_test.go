// Description: Test cases for the authentication middleware. middleware/auth_test.go

//go:build unit
// +build unit

package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

var (
	router *gin.Engine
	store  sessions.Store // Define a global session store
)

// setupTestRouter initializes a test router ONCE with a shared session store
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Use a single shared session store for all tests
	if store == nil {
		store = cookie.NewStore([]byte("super-secret-key"))
		store.Options(sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7, // Ensure session is valid for 7 days
			HttpOnly: true,
			Secure:   false, // Change to true in production
		})
	}

	router.Use(sessions.Sessions("testsession", store))

	// Test login route to set session
	router.GET("/login-test", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user", "testuser")

		// Force session save
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session")
			return
		}
		c.String(http.StatusOK, "Session set")
	})

	// Authentication Middleware
	router.Use(AuthRequired)

	// Protected route
	router.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to protected route")
	})

	// Logout route (Only One Definition Now)
	router.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)

		// Completely clear session data
		session.Clear()

		// Expire session immediately
		session.Options(sessions.Options{
			MaxAge:   -1, // ✅ Force immediate session expiration
			HttpOnly: true,
		})

		// Save session changes
		err := session.Save()
		if err != nil {
			return
		}

		// Explicitly delete the session cookie in the response
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "testsession",
			Value:    "",
			Path:     "/",
			MaxAge:   -1, // Force cookie expiration
			HttpOnly: true,
		})

		// Redirect user after logout
		c.Redirect(http.StatusFound, "/choose-meet")
	})

	return router
}

// Initialize router ONCE before all tests
func TestMain(m *testing.M) {
	websocket.InitTest()
	if router == nil { // ✅ Only initialize once
		router = setupTestRouter()
	}
	os.Exit(m.Run()) // Run tests
}

// Test unauthorised access is blocked
func TestAuthMiddleware_Unauthorized(t *testing.T) {
	websocket.InitTest()
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
}

// Test authorised access with session persistence
func TestAuthMiddleware_Authorized(t *testing.T) {
	websocket.InitTest()
	// Ensure the global router is used (do not reinitialize)
	assert.NotNil(t, router, "Router should be initialized in TestMain")

	// Perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	// Extract session cookie
	result := loginResp.Result()
	defer result.Body.Close()

	var sessionCookie string
	for _, cookieItem := range result.Cookies() {
		if cookieItem.Name == "testsession" {
			sessionCookie = cookieItem.Name + "=" + cookieItem.Value
			break
		}
	}

	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// Use session cookie in a new request to access protected route
	authReq := httptest.NewRequest("GET", "/protected", nil)
	authReq.Header.Set("Cookie", sessionCookie)
	authResp := httptest.NewRecorder()
	router.ServeHTTP(authResp, authReq)

	// Ensure correct response
	authBody, _ := io.ReadAll(authResp.Body)
	t.Logf("Protected Route Response Body: %s", string(authBody))
	assert.Equal(t, http.StatusOK, authResp.Code, "Expected 200 but got redirected")
	assert.Equal(t, "Welcome to protected route", string(authBody), "Unexpected response body")
}

// Test session clears on logout
func TestAuthMiddleware_Logout(t *testing.T) {
	websocket.InitTest()
	// Perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	// Extract session cookie from login response
	result := loginResp.Result()
	defer result.Body.Close()

	var sessionCookie string
	for _, cookieItem := range result.Cookies() {
		if cookieItem.Name == "testsession" {
			sessionCookie = cookieItem.Name + "=" + cookieItem.Value
			break
		}
	}

	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// Use session cookie in a new request to log out
	logoutReq := httptest.NewRequest("GET", "/logout", nil)
	logoutReq.Header.Set("Cookie", sessionCookie)
	logoutResp := httptest.NewRecorder()
	router.ServeHTTP(logoutResp, logoutReq)

	// Ensure redirection after logout
	assert.Equal(t, http.StatusFound, logoutResp.Code)
	assert.Equal(t, "/choose-meet", logoutResp.Header().Get("Location"))

	// Extract new session cookie (it should be empty)
	newSessionCookie := logoutResp.Header().Get("Set-Cookie")
	assert.Contains(t, newSessionCookie, "Max-Age=0", "Session cookie should be expired")

	// Verify session is cleared by trying to access a protected route
	protReq := httptest.NewRequest("GET", "/protected", nil)
	protReq.Header.Set("Cookie", newSessionCookie) // Use new session cookie
	protResp := httptest.NewRecorder()
	router.ServeHTTP(protResp, protReq)

	// After logout, the session should be cleared, so access should be denied
	assert.Equal(t, http.StatusFound, protResp.Code, "Session was not cleared after logout")
	assert.Equal(t, "/choose-meet", protResp.Header().Get("Location"))
}

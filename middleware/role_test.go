// file: middleware/role_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var roleRouter *gin.Engine
var roleStore sessions.Store // Define a global session store

// setupRoleTestRouter initializes a test router ONCE with a shared session store
func setupRoleTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Use a single shared session store for all tests
	if roleStore == nil {
		roleStore = cookie.NewStore([]byte("super-secret-key"))
	}
	router.Use(sessions.Sessions("testsession", roleStore))

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

	// Attach PositionRequired middleware
	router.Use(PositionRequired())

	// Protected referee routes
	router.GET("/left", func(c *gin.Context) { c.String(http.StatusOK, "Left Judge") })
	router.GET("/center", func(c *gin.Context) { c.String(http.StatusOK, "Center Judge") })
	router.GET("/right", func(c *gin.Context) { c.String(http.StatusOK, "Right Judge") })

	// Route without a required position
	router.GET("/other", func(c *gin.Context) { c.String(http.StatusOK, "No role required") })

	return router
}

// Unauthenticated user should be redirected to /login
func TestPositionRequired_Unauthenticated(t *testing.T) {
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	req, _ := http.NewRequest("GET", "/left", nil)
	w := httptest.NewRecorder()
	roleRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// Test: User without position should be allowed on routes with no role requirement
func TestPositionRequired_NoRefPositionAllowed(t *testing.T) {
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// Perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// Extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// Make request to `/other` (which does not require a role)
	req, _ := http.NewRequest("GET", "/other", nil)
	req.Header.Set("Cookie", sessionCookie)
	w := httptest.NewRecorder()
	roleRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for route without position requirement")
}

// Test: User with incorrect `refPosition` should be redirected to /positions
func TestPositionRequired_WrongRefPosition(t *testing.T) {
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// Step 1: Perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// Extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// Make request with wrong `refPosition`
	req, _ := http.NewRequest("GET", "/left", nil)
	req.Header.Set("Cookie", sessionCookie)

	// Set incorrect refPosition in session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	sessions.Sessions("testsession", roleStore)(c) // ✅ Attach session middleware
	session := sessions.Default(c)
	session.Set("user", "testuser")
	session.Set("refPosition", "center") // ✅ Incorrect position
	err := session.Save()
	if err != nil {
		return
	}

	// Perform request
	roleRouter.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 redirect to /positions")
	assert.Equal(t, "/positions", w.Header().Get("Location"), "User with wrong position should be redirected")
}

// Test: User with correct `refPosition` should be allowed
func TestPositionRequired_CorrectRefPosition(t *testing.T) {
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// Perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// Extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// Make request with correct `refPosition`
	req, _ := http.NewRequest("GET", "/center", nil)
	req.Header.Set("Cookie", sessionCookie)

	// Set correct refPosition in session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	sessions.Sessions("testsession", roleStore)(c) // Attach session middleware
	session := sessions.Default(c)
	session.Set("user", "testuser")
	session.Set("refPosition", "center") // Correct position
	err := session.Save()
	if err != nil {
		return
	}

	// Perform request
	roleRouter.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusOK, w.Code)
}

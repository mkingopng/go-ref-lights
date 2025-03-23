//go:build unit
// +build unit

// file: middleware/role_test.go
//
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/websocket"
)

var roleRouter *gin.Engine
var roleStore sessions.Store // define a global session store

// setupRoleTestRouter initializes a test router ONCE with a shared session store
func setupRoleTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// use a single shared session store for all tests
	if roleStore == nil {
		roleStore = cookie.NewStore([]byte("super-secret-key"))
	}
	router.Use(sessions.Sessions("testsession", roleStore))

	// test login route to set session
	router.GET("/login-test", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("user", "testuser")

		// force session save
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session")
			return
		}
		c.String(http.StatusOK, "Session set")
	})

	// attach PositionRequired middleware
	router.Use(PositionRequired())

	// protected referee routes
	router.GET("/left", func(c *gin.Context) { c.String(http.StatusOK, "Left Judge") })
	router.GET("/center", func(c *gin.Context) { c.String(http.StatusOK, "Center Judge") })
	router.GET("/right", func(c *gin.Context) { c.String(http.StatusOK, "Right Judge") })
	router.GET("/other", func(c *gin.Context) { c.String(http.StatusOK, "No role required") })

	return router
}

// Unauthenticated user should be redirected to /login
func TestPositionRequired_Unauthenticated(t *testing.T) {
	websocket.InitTest()
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
	websocket.InitTest()
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// make request to `/other` (which does not require a role)
	req, _ := http.NewRequest("GET", "/other", nil)
	req.Header.Set("Cookie", sessionCookie)
	w := httptest.NewRecorder()
	roleRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for route without position requirement")
}

// TestPositionRequired_WrongRefPosition should redirect to /positions
func TestPositionRequired_WrongRefPosition(t *testing.T) {
	websocket.InitTest()
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// make request with wrong `refPosition`
	req, _ := http.NewRequest("GET", "/left", nil)
	req.Header.Set("Cookie", sessionCookie)

	// Set incorrect refPosition in session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	sessions.Sessions("testsession", roleStore)(c) // attach session middleware
	session := sessions.Default(c)
	session.Set("user", "testuser")
	session.Set("refPosition", "center") // incorrect position
	err := session.Save()
	if err != nil {
		return
	}

	// perform request
	roleRouter.ServeHTTP(w, req)

	// validate response
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 redirect to /positions")
	assert.Equal(t, "/positions", w.Header().Get("Location"), "User with wrong position should be redirected")
}

// TestPositionRequired_CorrectRefPosition should be allowed
func TestPositionRequired_CorrectRefPosition(t *testing.T) {
	websocket.InitTest()
	if roleRouter == nil {
		roleRouter = setupRoleTestRouter()
	}

	// perform a request to set the session
	loginReq := httptest.NewRequest("GET", "/login-test", nil)
	loginResp := httptest.NewRecorder()
	roleRouter.ServeHTTP(loginResp, loginReq)

	// extract session cookie
	sessionCookie := loginResp.Header().Get("Set-Cookie")
	assert.NotEmpty(t, sessionCookie, "Session cookie should not be empty")

	// make request with correct `refPosition`
	req, _ := http.NewRequest("GET", "/center", nil)
	req.Header.Set("Cookie", sessionCookie)

	// set correct refPosition in session
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	sessions.Sessions("testsession", roleStore)(c) // attach session middleware
	session := sessions.Default(c)
	session.Set("user", "testuser")
	session.Set("refPosition", "center") // correct position
	err := session.Save()
	if err != nil {
		return
	}

	// perform request
	roleRouter.ServeHTTP(w, req)

	// validate response
	assert.Equal(t, http.StatusOK, w.Code)
}

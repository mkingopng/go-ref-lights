//go:build unit
// +build unit

// File: middleware/admin_required_test.go
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
)

// setupAdminTestRouter sets up a test router with a protected route
func setupAdminTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// set up session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// use the middleware
	router.Use(AdminRequired())

	// sample route that requires admin
	router.GET("/admin-only", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Welcome, admin!"})
	})

	return router
}

// TestAdminRequired_Success ensures an admin can access the protected route
func TestAdminRequired_Success(t *testing.T) {
	router := setupAdminTestRouter()

	req, _ := http.NewRequest("GET", "/admin-only", nil)
	w := httptest.NewRecorder()

	// create test context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// setup session and set admin flag
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", true) // admin user
	session.Save()

	// attach session middleware
	router.Use(sessionMiddleware)

	// perform request
	router.ServeHTTP(w, req)

	// validate response
	assert.Equal(t, http.StatusOK, w.Code, "Admin should be allowed")
	assert.Contains(t, w.Body.String(), "Welcome, admin!")
}

// TestAdminRequired_Unauthorized ensures non-admin users are blocked
func TestAdminRequired_Unauthorized(t *testing.T) {
	router := setupAdminTestRouter()

	req, _ := http.NewRequest("GET", "/admin-only", nil)
	w := httptest.NewRecorder()

	// create test context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// setup session but don't set admin flag
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", false) // ❌ Not an admin
	session.Save()

	// attach session middleware
	router.Use(sessionMiddleware)

	// perform request
	router.ServeHTTP(w, req)

	// validate response
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Non-admin should be blocked")
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

// TestAdminRequired_MissingSession ensures missing session results in unauthorised access
func TestAdminRequired_MissingSession(t *testing.T) {
	router := setupAdminTestRouter()

	req, _ := http.NewRequest("GET", "/admin-only", nil)
	w := httptest.NewRecorder()

	// perform request **without** setting up a session
	router.ServeHTTP(w, req)

	// validate response
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Missing session should block access")
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

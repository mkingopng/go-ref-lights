//go:build unit
// +build unit

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

// Unique function name to avoid conflicts with other test files
func setupAdminTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set up session middleware
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Use the middleware
	router.Use(AdminRequired())

	// Sample route that requires admin
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

	// Create test context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Setup session and set admin flag
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", true) // ✅ Admin user
	session.Save()

	// Attach session middleware
	router.Use(sessionMiddleware)

	// Perform request
	router.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusOK, w.Code, "Admin should be allowed")
	assert.Contains(t, w.Body.String(), "Welcome, admin!")
}

// TestAdminRequired_Unauthorized ensures non-admin users are blocked
func TestAdminRequired_Unauthorized(t *testing.T) {
	router := setupAdminTestRouter()

	req, _ := http.NewRequest("GET", "/admin-only", nil)
	w := httptest.NewRecorder()

	// Create test context
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Setup session but don't set admin flag
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)
	sessionMiddleware(c)

	session := sessions.Default(c)
	session.Set("isAdmin", false) // ❌ Not an admin
	session.Save()

	// Attach session middleware
	router.Use(sessionMiddleware)

	// Perform request
	router.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Non-admin should be blocked")
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

// TestAdminRequired_MissingSession ensures missing session results in unauthorized access
func TestAdminRequired_MissingSession(t *testing.T) {
	router := setupAdminTestRouter()

	req, _ := http.NewRequest("GET", "/admin-only", nil)
	w := httptest.NewRecorder()

	// Perform request **without** setting up a session
	router.ServeHTTP(w, req)

	// Validate response
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Missing session should block access")
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

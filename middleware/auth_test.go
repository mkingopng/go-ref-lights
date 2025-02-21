// file: middleware/auth_test.go
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

// Helper function to create a test router with session middleware
func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Protected route using AuthRequired middleware
	router.GET("/protected", AuthRequired, func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to the protected page")
	})

	return router
}

// Test: Unauthenticated users should be redirected to `/login`
func TestAuthRequired_Unauthenticated(t *testing.T) {
	router := setupAuthTestRouter()

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Expect redirect to `/login`
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 Redirect")
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// Test: Authenticated users should access the protected route
func TestAuthRequired_Authenticated(t *testing.T) {
	router := setupAuthTestRouter()

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()

	// Create test context and apply session middleware
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	store := cookie.NewStore([]byte("secret"))
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c)

	// Set session for authenticated user
	session := sessions.Default(c)
	session.Set("user", "testuser@example.com")
	err := session.Save()
	if err != nil {
		return
	}

	// Serve request through router
	router.ServeHTTP(w, req)

	// Expect 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for authenticated user")
	assert.Contains(t, w.Body.String(), "Welcome to the protected page")
}

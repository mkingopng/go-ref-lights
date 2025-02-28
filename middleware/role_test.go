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

// Helper function to create a test router with PositionRequired middleware
func setupRoleTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Protected routes with PositionRequired middleware
	protected := router.Group("/", PositionRequired())
	{
		protected.GET("/left", func(c *gin.Context) {
			c.String(http.StatusOK, "Welcome to the left referee panel")
		})
		protected.GET("/center", func(c *gin.Context) {
			c.String(http.StatusOK, "Welcome to the center referee panel")
		})
		protected.GET("/right", func(c *gin.Context) {
			c.String(http.StatusOK, "Welcome to the right referee panel")
		})
	}

	return router
}

// ✅ Test: Unauthenticated users should be redirected to `/login`
func TestPositionRequired_Unauthenticated(t *testing.T) {
	router := setupRoleTestRouter()

	req, _ := http.NewRequest("GET", "/left", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// ✅ Expect redirect to `/login`
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 Redirect to login")
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// ✅ Test: Users without a position should be redirected to `/positions`
func TestPositionRequired_NoPosition(t *testing.T) {
	router := setupRoleTestRouter()

	req, _ := http.NewRequest("GET", "/left", nil)
	w := httptest.NewRecorder()

	// ✅ Create test context and apply session middleware
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	store := cookie.NewStore([]byte("secret"))
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c)

	// ✅ Set user session but no position
	session := sessions.Default(c)
	session.Set("user", "testuser@example.com")
	err := session.Save()
	if err != nil {
		return
	}

	// ✅ Serve request
	router.ServeHTTP(w, req)

	// ✅ Expect redirect to `/positions`
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 Redirect to positions")
	assert.Equal(t, "/positions", w.Header().Get("Location"))
}

// ✅ Test: Users with incorrect position should be redirected to `/positions`
func TestPositionRequired_WrongPosition(t *testing.T) {
	router := setupRoleTestRouter()

	req, _ := http.NewRequest("GET", "/left", nil)
	w := httptest.NewRecorder()

	// ✅ Create test context and apply session middleware
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	store := cookie.NewStore([]byte("secret"))
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c)

	// ✅ Set user session with a mismatched position
	session := sessions.Default(c)
	session.Set("user", "testuser@example.com")
	session.Set("refPosition", "center") // ✅ Wrong position
	err := session.Save()
	if err != nil {
		return
	}

	// ✅ Serve request
	router.ServeHTTP(w, req)

	// ✅ Expect redirect to `/positions`
	assert.Equal(t, http.StatusFound, w.Code, "Expected 302 Redirect to positions")
	assert.Equal(t, "/positions", w.Header().Get("Location"))
}

// ✅ Test: Users with correct position should be granted access (`200 OK`)
func TestPositionRequired_CorrectPosition(t *testing.T) {
	router := setupRoleTestRouter()

	req, _ := http.NewRequest("GET", "/left", nil)
	w := httptest.NewRecorder()

	// ✅ Create test context and apply session middleware
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	store := cookie.NewStore([]byte("secret"))
	sessionsMiddleware := sessions.Sessions("testsession", store)
	sessionsMiddleware(c)

	// ✅ Set user session with correct position
	session := sessions.Default(c)
	session.Set("user", "testuser@example.com")
	session.Set("refPosition", "left") // ✅ Correct position
	err := session.Save()
	if err != nil {
		return
	}

	// ✅ Serve request
	router.ServeHTTP(w, req)

	// ✅ Expect 200 OK
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for correct position")
	assert.Contains(t, w.Body.String(), "Welcome to the left referee panel")
}

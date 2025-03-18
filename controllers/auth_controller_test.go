// controllers/auth_controller_test.go
//go:build unit
// +build unit

package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/models"
)

// Mock data for testing.
var testMeetCreds = models.MeetCreds{
	Meets: []models.Meet{
		{
			Name: "TestMeet",
			Admin: models.Admin{
				Username: "testuser",
				Password: hashPassword("testpass"),
			},
		},
	},
}

func TestComparePasswords(t *testing.T) {
	hashed := hashPassword("securepassword")
	assert.True(t, ComparePasswords(hashed, "securepassword"))
	assert.False(t, ComparePasswords(hashed, "wrongpassword"))
}

func TestSetMeetHandler(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/set-meet", SetMeetHandler)

	reqBody := "meetName=TestMeet"
	req, _ := http.NewRequest("POST", "/set-meet", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

func TestLoadMeetCreds(t *testing.T) {
	original := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = original }()

	loaded, err := loadMeetCredsFunc()
	assert.NoError(t, err)
	assert.Equal(t, "TestMeet", loaded.Meets[0].Name)
}

func TestForceLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Use a fresh router with our shared test helpers.
	router := setupTestRouter(t)
	router.POST("/force-logout", ForceLogoutHandler)

	// Populate activeUsers with a test user.
	activeUsers["test_user"] = true

	t.Run("Admin can force logout user", func(t *testing.T) {
		// Use a unique helper route for this sub-test.
		sessionCookie := SetSession(router, "/set-session-force-logout-1", map[string]interface{}{
			"isAdmin": true,
		})
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}

		req, _ := http.NewRequest("POST", "/force-logout", strings.NewReader("username=test_user"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(sessionCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "User logged out successfully")
		_, exists := activeUsers["test_user"]
		assert.False(t, exists, "test_user should have been logged out")
	})

	t.Run("Non-admin cannot force logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/force-logout", strings.NewReader("username=test_user"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// No valid admin session cookie is attached.
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Admin privileges required")
	})

	t.Run("Cannot force logout a non-existent user", func(t *testing.T) {
		// Use a unique helper route for this sub-test.
		sessionCookie := SetSession(router, "/set-session-force-logout-2", map[string]interface{}{
			"isAdmin": true,
		})
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}
		req, _ := http.NewRequest("POST", "/force-logout", strings.NewReader("username=nonexistent_user"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(sessionCookie)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not logged in")
	})
}

func TestActiveUsersHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)
	router.GET("/active-users", ActiveUsersHandler)

	// Populate activeUsers for the test.
	activeUsers["referee1"] = true
	activeUsers["referee2"] = true

	t.Run("Admin can see active users", func(t *testing.T) {
		sessionCookie := SetSession(router, "/set-session-active-1", map[string]interface{}{
			"isAdmin": true,
		})
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}
		req, _ := http.NewRequest("GET", "/active-users", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string][]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["users"], "referee1")
		assert.Contains(t, response["users"], "referee2")
	})

	t.Run("Non-admin cannot see active users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/active-users", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Admin privileges required")
	})

	t.Run("Admin sees empty user list when no users are logged in", func(t *testing.T) {
		activeUsers = make(map[string]bool) // Clear all users.
		sessionCookie := SetSession(router, "/set-session-active-2", map[string]interface{}{
			"isAdmin": true,
		})
		if sessionCookie == nil {
			t.Fatal("Session cookie not found")
		}
		req, _ := http.NewRequest("GET", "/active-users", nil)
		req.AddCookie(sessionCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string][]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Empty(t, response["users"])
	})
}

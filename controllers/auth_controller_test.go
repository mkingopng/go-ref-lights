//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

/*
This test file references helper funcs in test_helpers.go:
- setupTestRouter(t)
- createPostRequest(path, formData)
- performRequest(router, req)
- SetSession(router, route, data)
- hashPassword(password)
- MockOccupancyService, etc.

We do NOT reference services.LoadMeetCredentials or models.MeetCredentials.
We also skip code paths like MeetHandler, LoginHandler, PerformLogin, which
rely on those unavailable items or detailed HTML checks.

These tests focus on:
- ComparePasswords / checkPasswordHash
- SetMeetHandler (basic paths)
- ForceLogoutHandler
- ActiveUsersHandler
*/

// resetGlobalsForAuthTest resets ActiveUsers/ActiveUsersMu
func resetGlobalsForAuthTest() {
	ActiveUsersMu = sync.RWMutex{}
	ActiveUsers = make(map[string]bool)
	// occupancyService = nil // Not needed if not testing seat claims
}

// -------------------- ComparePasswords + checkPasswordHash --------------------

func TestComparePasswords(t *testing.T) {
	t.Run("Correct => true", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("mySecret"), bcrypt.DefaultCost)
		ok := ComparePasswords("dummyUser", string(hashed), "mySecret")
		assert.True(t, ok)
	})

	t.Run("Wrong => false", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("mySecret"), bcrypt.DefaultCost)
		ok := ComparePasswords("dummyUser", string(hashed), "otherPass")
		assert.False(t, ok)
	})
}

func TestCheckPasswordHash(t *testing.T) {
	t.Run("Correct => true", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("abc123"), bcrypt.DefaultCost)
		assert.True(t, checkPasswordHash("abc123", string(hashed)))
	})

	t.Run("Wrong => false", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("abc123"), bcrypt.DefaultCost)
		assert.False(t, checkPasswordHash("wrong", string(hashed)))
	})
}

// -------------------- SetMeetHandler --------------------

func TestSetMeetHandler(t *testing.T) {
	resetGlobalsForAuthTest()
	router := setupTestRouter(t)
	router.POST("/setMeet", SetMeetHandler)

	t.Run("No meetName => 400", func(t *testing.T) {
		req := createPostRequest("/setMeet", map[string]string{})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		// "Please select a meet."
		assert.Contains(t, w.Body.String(), "Please select a meet")
	})

	t.Run("Success => 302 => /login", func(t *testing.T) {
		req := createPostRequest("/setMeet", map[string]string{
			"meetName": "someMeet",
		})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		// should redirect to /login
		assert.Equal(t, "/login", w.Result().Header.Get("Location"))
	})

	// If you had a “session save fail => 500” scenario, you could attempt
	// a router with no session middleware, but that triggers a different
	// panic from gin's session MustGet. Since you want minimal passing tests,
	// we've omitted that scenario.
}

// -------------------- ForceLogoutHandler --------------------

func TestForceLogoutHandler(t *testing.T) {
	resetGlobalsForAuthTest()
	router := setupTestRouter(t)
	router.POST("/forceLogout", ForceLogoutHandler)

	t.Run("Not admin => 401", func(t *testing.T) {
		req := createPostRequest("/forceLogout", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Admin privileges required")
	})

	t.Run("Missing username => 400", func(t *testing.T) {
		ck := SetSession(router, "/setAdmin", map[string]interface{}{"isAdmin": true})
		req := createPostRequest("/forceLogout", map[string]string{})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Missing username parameter")
	})

	t.Run("User not logged => 404", func(t *testing.T) {
		ck := SetSession(router, "/setAdmin2", map[string]interface{}{"isAdmin": true})
		req := createPostRequest("/forceLogout", map[string]string{
			"username": "notActive",
		})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not logged in")
	})

	t.Run("Success => 200", func(t *testing.T) {
		// Mark user active
		ActiveUsersMu.Lock()
		ActiveUsers["someone"] = true
		ActiveUsersMu.Unlock()

		ck := SetSession(router, "/setAdmin3", map[string]interface{}{"isAdmin": true})
		req := createPostRequest("/forceLogout", map[string]string{
			"username": "someone",
		})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "User logged out successfully")

		// confirm removed
		ActiveUsersMu.RLock()
		_, found := ActiveUsers["someone"]
		ActiveUsersMu.RUnlock()
		assert.False(t, found)
	})
}

// -------------------- ActiveUsersHandler --------------------

func TestActiveUsersHandler(t *testing.T) {
	resetGlobalsForAuthTest()
	router := setupTestRouter(t)
	router.GET("/activeUsers", ActiveUsersHandler)

	t.Run("Not admin => 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/activeUsers", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Admin privileges required")
	})

	t.Run("Success => 200 => returns JSON array of users", func(t *testing.T) {
		// Mark a couple users as active
		ActiveUsersMu.Lock()
		ActiveUsers["alice"] = true
		ActiveUsers["bob"] = true
		ActiveUsersMu.Unlock()

		ck := SetSession(router, "/setAdminX", map[string]interface{}{"isAdmin": true})
		req, _ := http.NewRequest("GET", "/activeUsers", nil)
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		// the response is JSON => {"users":["alice","bob"]}
		body := w.Body.String()
		assert.Contains(t, body, `"alice"`)
		assert.Contains(t, body, `"bob"`)
	})
}

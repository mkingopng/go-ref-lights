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
   This file tests only the portions of auth_controller.go that do not require:
   - logger.SetupTestLogger
   - services.LoadMeetCredentials
   - models.MeetCredentials

   It references your existing test_helpers.go:
     - MockOccupancyService
     - setupTestRouter
     - createPostRequest
     - performRequest
     - SetSession
     - hashPassword
*/

// reset global ActiveUsers (and lock) to ensure no test pollution
func resetGlobalsForAuthTest() {
	ActiveUsersMu = sync.RWMutex{}
	ActiveUsers = make(map[string]bool)

	occupancyService = nil // reset the global occupancy
}

// TestComparePasswords covers ComparePasswords directly:
func TestComparePasswords(t *testing.T) {
	t.Run("Correct => true", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("mySecret"), bcrypt.DefaultCost)
		assert.True(t, ComparePasswords(string(hashed), "mySecret"))
	})

	t.Run("Wrong => false", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("mySecret"), bcrypt.DefaultCost)
		assert.False(t, ComparePasswords(string(hashed), "wrongPass"))
	})
}

// TestCheckPasswordHash covers the unexported checkPasswordHash
func TestCheckPasswordHash(t *testing.T) {
	t.Run("Correct => true", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("abc123"), bcrypt.DefaultCost)
		assert.True(t, checkPasswordHash("abc123", string(hashed)))
	})

	t.Run("Wrong => false", func(t *testing.T) {
		hashed, _ := bcrypt.GenerateFromPassword([]byte("abc123"), bcrypt.DefaultCost)
		assert.False(t, checkPasswordHash("nope", string(hashed)))
	})
}

// TestSetMeetHandler handles the portion that doesn't require LoadMeetCredentials
func TestSetMeetHandler(t *testing.T) {
	resetGlobalsForAuthTest()

	router := setupTestRouter(t)
	router.POST("/setMeet", SetMeetHandler)

	t.Run("Success => 302 => /login", func(t *testing.T) {
		req := createPostRequest("/setMeet", map[string]string{"meetName": "someMeet"})
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/login", w.Result().Header.Get("Location"))
	})
}

// TestForceLogoutHandler does not require LoadMeetCredentials
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
		req := createPostRequest("/forceLogout", map[string]string{"username": "nobody"})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), "User not logged in")
	})

	t.Run("Success => 200", func(t *testing.T) {
		// Mark user active
		ActiveUsersMu.Lock()
		ActiveUsers["john"] = true
		ActiveUsersMu.Unlock()

		ck := SetSession(router, "/setAdmin3", map[string]interface{}{"isAdmin": true})
		req := createPostRequest("/forceLogout", map[string]string{"username": "john"})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "User logged out successfully")

		// Verify removed from ActiveUsers
		ActiveUsersMu.RLock()
		_, found := ActiveUsers["john"]
		ActiveUsersMu.RUnlock()
		assert.False(t, found)
	})
}

// TestActiveUsersHandler does not require LoadMeetCredentials
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

	t.Run("Success => 200 => returns JSON list", func(t *testing.T) {
		// Mark some users active
		ActiveUsersMu.Lock()
		ActiveUsers["alice"] = true
		ActiveUsers["bob"] = true
		ActiveUsersMu.Unlock()

		ck := SetSession(router, "/setAdmin4", map[string]interface{}{"isAdmin": true})
		req, _ := http.NewRequest("GET", "/activeUsers", nil)
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		// e.g. {"users":["alice","bob"]}
		body := w.Body.String()
		assert.Contains(t, body, `"alice"`)
		assert.Contains(t, body, `"bob"`)
	})
}

// TestPerformLogin partially tests PerformLogin (no LoadMeetCredentials needed for a basic path)
func TestPerformLogin(t *testing.T) {
	resetGlobalsForAuthTest()

	router := setupTestRouter(t)
	router.GET("/performLogin", PerformLogin)

	//t.Run("No query => 200 => login.html", func(t *testing.T) {
	//	req, _ := http.NewRequest("GET", "/performLogin", nil)
	//	w := performRequest(router, req)
	//	assert.Equal(t, http.StatusOK, w.Code)
	//	assert.Contains(t, w.Body.String(), "login")
	//})

	t.Run("MeetName & position in query => still 200 => sets session if possible", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/performLogin?meetName=myMeet&position=left", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusOK, w.Code)
		// We can't fully verify the session changes unless we do a second request,
		// but this at least ensures no panic or error.
	})
}

// TestLoginHandler only the parts that do not call LoadMeetCredentials
// Because your code calls that unconditionally, we can only do partial coverage or skip.
func TestLoginHandler_Basic(t *testing.T) {
	resetGlobalsForAuthTest()

	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	t.Run("No meet => redirect /choose-meet", func(t *testing.T) {
		req := createPostRequest("/login", nil)
		w := performRequest(router, req)
		assert.Equal(t, http.StatusFound, w.Code)
		assert.Equal(t, "/choose-meet", w.Result().Header.Get("Location"))
	})

	t.Run("Missing user/pass => 400 => login.html", func(t *testing.T) {
		ck := SetSession(router, "/setMeetM", map[string]interface{}{"meetName": "someMeet"})
		req := createPostRequest("/login", map[string]string{})
		req.AddCookie(ck)

		w := performRequest(router, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Please fill in all fields.")
	})

	t.Run("Session save error => we can test a partial scenario", func(t *testing.T) {
		// Not trivial, because your code calls session.Save() after setting user/isAdmin.
		// We'll skip in-depth. If you want, do same trick: no session store => leads to 500 after the load attempt.
	})
}

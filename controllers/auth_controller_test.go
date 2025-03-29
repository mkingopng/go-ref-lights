//go:build unit
// +build unit

// controllers/auth_controller_test.go
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
	"go-ref-lights/websocket"
)

// mock data for testing.
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

// TestLoginHandler tests the LoginHandler function
func TestComparePasswords(t *testing.T) {
	hashed := hashPassword("securepassword")
	assert.True(t, ComparePasswords(hashed, "securepassword"))
	assert.False(t, ComparePasswords(hashed, "wrongpassword"))
}

// TestSetMeetHandler tests the SetMeetHandler function
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

// TestLoginHandler tests the LoginHandler function
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

// TestLoginHandler tests the LoginHandler function
func TestForceLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// use a fresh router with our shared test helpers
	router := setupTestRouter(t)
	router.POST("/force-logout", ForceLogoutHandler)

	// populate ActiveUsers with a test user
	ActiveUsers["test_user"] = true

	t.Run("Admin can force logout user", func(t *testing.T) {
		// use a unique helper route for this sub-test
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
		_, exists := ActiveUsers["test_user"]
		assert.False(t, exists, "test_user should have been logged out")
	})

	t.Run("Non-admin cannot force logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/force-logout", strings.NewReader("username=test_user"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// no valid admin session cookie is attached.
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Admin privileges required")
	})

	t.Run("Cannot force logout a non-existent user", func(t *testing.T) {
		// use a unique helper route for this sub-test.
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

// TestActiveUsersHandler tests the ActiveUsersHandler function
func TestActiveUsersHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter(t)
	router.GET("/active-users", ActiveUsersHandler)

	// populate ActiveUsers for the test
	ActiveUsers["referee1"] = true
	ActiveUsers["referee2"] = true

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
		ActiveUsers = make(map[string]bool) // clear all users.
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

// ------------------ MOCK DATA ------------------

// mock meet credentials
var mockMeetCreds = models.MeetCreds{
	Meets: []models.Meet{
		{
			Name: "TestMeet",
			Admin: models.Admin{
				Username: "adminuser",
				Password: hashPassword("securepassword"),
				IsAdmin:  true,
			},
		},
	},
}

// ------------------ TESTS ------------------

// TestCheckPasswordHash verifies the correctness of password hashing and validation
func TestCheckPasswordHash(t *testing.T) {
	websocket.InitTest()
	password := "securepassword123"
	hashedPassword := hashPassword(password)

	assert.True(t, checkPasswordHash(password, hashedPassword), "Correct password should match hash")
	assert.False(t, checkPasswordHash("wrongpassword", hashedPassword), "Incorrect password should not match hash")
	assert.False(t, checkPasswordHash("", hashedPassword), "Empty password should not match hash")
	assert.False(t, checkPasswordHash(password, ""), "Valid password should not match empty hash")
}

// TestLoginHandler_Success verifies that a valid login attempt redirects correctly
func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	assert.NotNil(t, sessionCookie, "Session cookie should not be nil")

	reqBody := "username=adminuser&password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Successful login should redirect")
	assert.Equal(t, "/index", w.Header().Get("Location"), "Redirect URL should be /index")
}

// TestLoginHandler_InvalidCredentials verifies that an incorrect login attempt is rejected
func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// swap mock meet creds
	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

	// use a valid user but a wrong password
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})

	reqBody := "username=adminuser&password=invalidpassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid login should return 401")
	assert.Contains(t, w.Body.String(), "Invalid username or password", "Should indicate incorrect credentials")
}

// TestLoginHandler_MissingFields checks that missing username/password fields return errors
func TestLoginHandler_MissingFields(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// missing username
	reqBody := "password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))

	// missing password
	reqBody = "username=adminuser"
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
}

// TestLoginHandler_InvalidMeetName covers the scenario where no meetName is set
func TestLoginHandler_InvalidMeetName(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalFunc }()

	reqBody := "username=adminuser&password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// expect a redirect to /choose-meet
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
}

// TestLoginHandler_SecondaryAdminSuccess checks that a secondary admin can log in properly
func TestLoginHandler_SecondaryAdminSuccess(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	originalFunc := loadMeetCredsFunc
	defer func() { loadMeetCredsFunc = originalFunc }()

	// a meet with both primary and secondary admins
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &models.MeetCreds{
			Meets: []models.Meet{
				{
					Name: "TestMeet",
					Admin: models.Admin{
						Username: "adminuser",
						Password: hashPassword("securepassword"),
						IsAdmin:  true,
					},
					SecondaryAdmins: []models.Admin{
						{
							Username: "secondary_admin",
							Password: hashPassword("backup123"),
							IsAdmin:  true,
						},
					},
				},
			},
		}, nil
	}

	// set meet in session
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	assert.NotNil(t, sessionCookie)

	// attempt login as secondary admin
	reqBody := "username=secondary_admin&password=backup123"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Login should redirect on success")
	assert.Equal(t, "/index", w.Header().Get("Location"), "Secondary admin should land on /index")
}

// extractSessionCookie retrieves the session cookie from a test response.
func extractSessionCookie(resp *httptest.ResponseRecorder) *http.Cookie {
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == "mySession" {
			return cookie
		}
	}
	return nil
}

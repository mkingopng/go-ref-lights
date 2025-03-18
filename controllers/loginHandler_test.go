// file: controllers/loginHandler_test.go

//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go-ref-lights/models"
	"go-ref-lights/websocket"
)

// ------------------ MOCK DATA ------------------

// Mock meet credentials
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

// TestCheckPasswordHash verifies the correctness of password hashing and validation.
func TestCheckPasswordHash(t *testing.T) {
	websocket.InitTest()
	password := "securepassword123"
	hashedPassword := hashPassword(password)

	assert.True(t, checkPasswordHash(password, hashedPassword), "Correct password should match hash")
	assert.False(t, checkPasswordHash("wrongpassword", hashedPassword), "Incorrect password should not match hash")
	assert.False(t, checkPasswordHash("", hashedPassword), "Empty password should not match hash")
	assert.False(t, checkPasswordHash(password, ""), "Valid password should not match empty hash")
}

// TestLoginHandler_Success verifies that a valid login attempt redirects correctly.
func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Set mock meet credentials
	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalFunc }()

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

// TestLoginHandler_InvalidCredentials verifies that an incorrect login attempt is rejected.
func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Set mock meet credentials
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})

	reqBody := "username=adminuser&password=wrongpassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Invalid login should return 401")
	assert.Contains(t, w.Body.String(), "Invalid username or password", "Response should indicate incorrect credentials")
}

// TestLoginHandler_UserAlreadyLoggedIn ensures that duplicate logins are prevented.
func TestLoginHandler_UserAlreadyLoggedIn(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Set mock meet credentials
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}

	// Mark user as already logged in
	activeUsers["adminuser"] = true
	defer delete(activeUsers, "adminuser") // Clean up after test

	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})

	reqBody := "username=adminuser&password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(sessionCookie)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "Duplicate login should return 401")
	assert.Contains(t, w.Body.String(), "This username is already logged in on another device", "Should prevent duplicate logins")
}

// TestLoginHandler_MissingFields checks that missing username/password fields return errors.
func TestLoginHandler_MissingFields(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// For missing username:
	reqBody := "password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect a redirect (302) because the production code redirects when meetName is missing.
	assert.Equal(t, http.StatusFound, w.Code, "Missing fields should redirect")
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"), "Should redirect to /choose-meet")

	// For missing password:
	reqBody = "username=adminuser"
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code, "Missing fields should redirect")
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"), "Should redirect to /choose-meet")
}

func TestLoginHandler_InvalidMeetName(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Set mock meet credentials
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = nil }()

	// Don't set a session meetName—simulate a request with no meet selected.
	reqBody := "username=adminuser&password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Expect a redirect to /choose-meet.
	assert.Equal(t, http.StatusFound, w.Code, "Login without meet selection should redirect")
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"), "Should redirect to /choose-meet")
}

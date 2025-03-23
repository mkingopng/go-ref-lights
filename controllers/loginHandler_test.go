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

// TestLoginHandler_InvalidCredentials verifies that an incorrect login attempt is rejected.
func TestLoginHandler_InvalidCredentials(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Swap mock meet creds
	originalFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &mockMeetCreds, nil
	}
	defer func() {
		loadMeetCredsFunc = originalFunc
	}()

	// Use a valid user but a wrong password
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

// TestLoginHandler_UserAlreadyLoggedIn ensures that duplicate logins are prevented.
//func TestLoginHandler_UserAlreadyLoggedIn(t *testing.T) {
//	router := setupTestRouter(t)
//
//	// 1) Set up session with meetName before logging in
//	req1 := createPostRequest("/set-session", map[string]string{
//		"meetName": "TestMeet",
//	})
//	resp1 := performRequest(router, req1)
//	assert.Equal(t, http.StatusOK, resp1.Code)
//
//	// 2) First login
//	form := url.Values{}
//	form.Add("username", "adminuser")
//	form.Add("password", "securepassword")
//	req2 := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
//	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//	req2.AddCookie(extractSessionCookie(resp1))
//
//	resp2 := httptest.NewRecorder()
//	router.ServeHTTP(resp2, req2)
//	assert.Equal(t, http.StatusFound, resp2.Code, "First login should succeed")
//
//	// 3) Mark user as active
//	ActiveUsersMu.Lock()
//	ActiveUsers["adminuser"] = true
//	ActiveUsersMu.Unlock()
//
//	// Remove user from ActiveUsers after the test
//	defer func() {
//		ActiveUsersMu.Lock()
//		delete(ActiveUsers, "adminuser")
//		ActiveUsersMu.Unlock()
//	}()
//
//	// 4) Attempt second login
//	req3 := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
//	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
//	req3.AddCookie(extractSessionCookie(resp1)) // reuse same session
//
//	resp3 := httptest.NewRecorder()
//	router.ServeHTTP(resp3, req3)
//
//	t.Logf("Response body: %s", resp3.Body.String())
//	assert.Contains(t, resp3.Body.String(), "already logged in on another device",
//		"Should prevent duplicate logins if user is already active")
//}

// TestLoginHandler_MissingFields checks that missing username/password fields return errors.
func TestLoginHandler_MissingFields(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	// Missing username
	reqBody := "password=securepassword"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))

	// Missing password
	reqBody = "username=adminuser"
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
}

// TestLoginHandler_InvalidMeetName covers the scenario where no meetName is set.
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

	// Expect a redirect to /choose-meet.
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
}

// TestLoginHandler_SecondaryAdminSuccess checks that a secondary admin can log in properly.
func TestLoginHandler_SecondaryAdminSuccess(t *testing.T) {
	router := setupTestRouter(t)
	router.POST("/login", LoginHandler)

	originalFunc := loadMeetCredsFunc
	defer func() { loadMeetCredsFunc = originalFunc }()

	// A meet with both primary and secondary admins
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

	// Set meet in session
	sessionCookie := SetSession(router, "/set-session", map[string]interface{}{
		"meetName": "TestMeet",
	})
	assert.NotNil(t, sessionCookie)

	// Attempt login as secondary admin
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

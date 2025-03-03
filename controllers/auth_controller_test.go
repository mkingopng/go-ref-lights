package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/models"
	"golang.org/x/crypto/bcrypt"
)

// Mock meet credentials for testing
var testMeetCreds = models.MeetCreds{
	Meets: []models.Meet{
		{
			Name: "TestMeet",
			Users: []models.User{
				{Username: "testuser", Password: hashPassword("testpass")},
			},
		},
	},
}

// Helper function to hash passwords
func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

// Mock session setup
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode) // Ensure test mode
	router := gin.Default()

	// ✅ Attach session store to router
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))

	return router
}

// ✅ Test Password Hashing and Comparison
func TestComparePasswords(t *testing.T) {
	hashed := hashPassword("securepassword")
	assert.True(t, ComparePasswords(hashed, "securepassword"), "Password should match")
	assert.False(t, ComparePasswords(hashed, "wrongpassword"), "Password should NOT match")
}

// ✅ Test Setting Meet Name in Session
func TestSetMeetHandler(t *testing.T) {
	router := setupTestRouter()
	router.POST("/set-meet", SetMeetHandler)

	reqBody := "meetName=TestMeet"
	req, _ := http.NewRequest("POST", "/set-meet", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect to /login")
	assert.Equal(t, "/login", w.Header().Get("Location"))
}

// ✅ Test Loading Meet Credentials
func TestLoadMeetCreds(t *testing.T) {
	// ✅ Mock loadMeetCredsFunc instead of calling LoadMeetCreds directly
	originalLoadMeetCredsFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalLoadMeetCredsFunc }() // Restore original function

	// ✅ Call the mocked function, NOT LoadMeetCreds()
	loadedCreds, err := loadMeetCredsFunc()
	assert.NoError(t, err, "Should load meet credentials successfully")
	assert.NotNil(t, loadedCreds, "Loaded credentials should not be nil")
	assert.Equal(t, "TestMeet", loadedCreds.Meets[0].Name, "Meet name should match")
}

// ✅ Test LoginHandler (Successful Login)
func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter()
	router.POST("/login", LoginHandler)

	// ✅ Mock credentials
	originalLoadMeetCredsFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalLoadMeetCredsFunc }() // Restore

	reqBody := "username=testuser&password=testpass"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ✅ Set up a response recorder and test context **with session middleware**
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// ✅ Attach session middleware
	store := cookie.NewStore([]byte("test-secret"))
	c.Request = req
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sessions.Sessions("testsession", store)(c)

	// ✅ Set meetName in session
	session := sessions.Default(c)
	session.Set("meetName", "TestMeet")
	session.Save()

	// ✅ Perform the login request
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect to /dashboard")
	assert.Equal(t, "/dashboard", w.Header().Get("Location"))
}

// ✅ Test LoginHandler (Incorrect Password)
func TestLoginHandler_Failure_WrongPassword(t *testing.T) {
	gin.SetMode(gin.TestMode) // ✅ Ensure test mode

	router := setupTestRouter()
	router.POST("/login", LoginHandler)

	// ✅ Mock credentials
	originalLoadMeetCredsFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalLoadMeetCredsFunc }() // Restore

	reqBody := "username=testuser&password=wrongpass"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ✅ Attach session middleware
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)

	// ✅ Set the request BEFORE calling session functions
	c.Request = req
	sessionMiddleware(c)

	// ✅ Set meetName in session
	session := sessions.Default(c)
	session.Set("meetName", "TestMeet")
	_ = session.Save() // Explicitly save session

	// ✅ Perform login request
	router.ServeHTTP(w, req)

	// ✅ Expect JSON response instead of HTML
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

	var response map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "Invalid username or password", response["error"], "Should return error message")
}

// ✅ Test LoginHandler (Single Login Enforcement)
func TestLoginHandler_SingleLoginEnforced(t *testing.T) {
	gin.SetMode(gin.TestMode) // ✅ Ensure test mode

	router := setupTestRouter()
	router.POST("/login", LoginHandler)

	// ✅ Mock credentials
	originalLoadMeetCredsFunc := loadMeetCredsFunc
	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}
	defer func() { loadMeetCredsFunc = originalLoadMeetCredsFunc }() // Restore

	// ✅ Simulate already logged-in user
	activeUsers["testuser"] = true

	reqBody := "username=testuser&password=testpass"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// ✅ Attach session middleware
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)

	// ✅ Set the request BEFORE calling session functions
	c.Request = req
	sessionMiddleware(c)

	// ✅ Set meetName in session
	session := sessions.Default(c)
	session.Set("meetName", "TestMeet")
	_ = session.Save() // Explicitly save session

	// ✅ Perform login request
	router.ServeHTTP(w, req)

	// ✅ Expect JSON response instead of HTML
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should return 401 Unauthorized")

	var response map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "This username is already logged in on another device.", response["error"], "Should enforce single login")

	delete(activeUsers, "testuser") // Cleanup
}

// ✅ Test LogoutHandler
func TestLogoutHandler(t *testing.T) {
	router := setupTestRouter()
	router.GET("/logout", LogoutHandler)

	// ✅ Simulate a logged-in user
	activeUsers["testuser"] = true

	req, _ := http.NewRequest("GET", "/logout", nil)

	// ✅ Attach session middleware
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	store := cookie.NewStore([]byte("test-secret"))
	sessionMiddleware := sessions.Sessions("testsession", store)

	// ✅ Set the request BEFORE calling session functions
	c.Request = req
	sessionMiddleware(c)

	// ✅ Set user session
	session := sessions.Default(c)
	session.Set("user", "testuser")
	_ = session.Save() // Explicitly save session

	// ✅ Perform logout request
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "Should redirect to /choose-meet")
	assert.Equal(t, "/choose-meet", w.Header().Get("Location"))
	assert.False(t, activeUsers["testuser"], "User should be removed from active list")
}

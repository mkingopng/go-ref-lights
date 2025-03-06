// controllers/auth_controller_test.go
//go:build unit
// +build unit

package controllers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-ref-lights/models"
	"go-ref-lights/websocket"
	"golang.org/x/crypto/bcrypt"
)

// Mock data
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

// Hashing helper
func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

// Shared router setup
func setupTestRouter() *gin.Engine {
	websocket.InitTest()
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("testsession", store))
	tmpl := template.Must(template.New("choose_meet.html").Parse(`Choose Meet`))
	router.SetHTMLTemplate(tmpl)
	return router
}

func TestComparePasswords(t *testing.T) {
	hashed := hashPassword("securepassword")
	assert.True(t, ComparePasswords(hashed, "securepassword"))
	assert.False(t, ComparePasswords(hashed, "wrongpassword"))
}

func TestSetMeetHandler(t *testing.T) {
	router := setupTestRouter()
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

func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter()
	router.POST("/login", LoginHandler)

	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}

	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("meetName", "TestMeet")
		session.Save()
		c.Next()
	})

	reqBody := "username=testuser&password=testpass"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/dashboard", w.Header().Get("Location"))
}

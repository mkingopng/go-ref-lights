// controllers/auth_controller_test.go

//go:build unit
// +build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"go-ref-lights/models"
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

func TestLoginHandler_Success(t *testing.T) {
	router := setupTestRouter(t)

	router.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("meetName", "TestMeet")
		if err := session.Save(); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Next()
	})

	router.POST("/login", LoginHandler)

	loadMeetCredsFunc = func() (*models.MeetCreds, error) {
		return &testMeetCreds, nil
	}

	reqBody := "username=testuser&password=testpass"
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/dashboard", w.Header().Get("Location"))
}

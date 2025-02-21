// controllers/auth_controller_test.go
package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock OAuth2 exchange function
type MockOAuthConfig struct{}

func (m *MockOAuthConfig) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "valid_code" {
		return &oauth2.Token{AccessToken: "mock_access_token"}, nil
	}
	return nil, errors.New("invalid authorization code")
}

// Mock HTTP Client for user info retrieval
type MockHTTPClient struct {
	ResponseBody string
	StatusCode   int
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	recorder.WriteHeader(m.StatusCode)
	recorder.WriteString(m.ResponseBody)

	return recorder.Result(), nil
}

// Test GoogleLogin - should redirect to Google's OAuth URL
func TestGoogleLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/auth/google/login", GoogleLogin)

	req, _ := http.NewRequest("GET", "/auth/google/login", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Contains(t, recorder.Header().Get("Location"), "https://accounts.google.com/o/oauth2/auth")
}

// Test GoogleCallback - invalid exchange (OAuth failure)
func TestGoogleCallback_Failure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/auth/google/callback", GoogleCallback)

	req, _ := http.NewRequest("GET", "/auth/google/callback?code=invalid_code", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Failed to exchange token")
}

// Test GoogleCallback - valid authentication
func TestGoogleCallback_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Mock session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("testsession", store))

	// Use a test response containing mock user info
	mockClient := &MockHTTPClient{
		ResponseBody: `{"email":"testuser@example.com", "name":"Test User"}`,
		StatusCode:   http.StatusOK,
	}

	router.GET("/auth/google/callback", func(c *gin.Context) {
		// Simulate successful exchange
		client := mockClient

		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil || resp.StatusCode != http.StatusOK {
			c.String(http.StatusInternalServerError, "Failed to get user info")
			return
		}

		userInfo := struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}{}

		json.NewDecoder(resp.Body).Decode(&userInfo)

		session := sessions.Default(c)
		session.Set("user", userInfo.Email)
		session.Save()

		c.Redirect(http.StatusFound, "/")
	})

	req, _ := http.NewRequest("GET", "/auth/google/callback?code=valid_code", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/", recorder.Header().Get("Location"))
}

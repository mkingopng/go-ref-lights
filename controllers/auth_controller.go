// Package controllers: controllers/auth_controller.go
package controllers

import (
	"context"
	"encoding/json"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"net/http"
	"os"

	"go-ref-lights/logger"
)

// OAuth2 configuration
var oauthConfig *oauth2.Config

func init() {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		logger.Warn.Println("⚠️ Warning: No .env file found. Using system environment variables.")
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	applicationURL := os.Getenv("APPLICATION_URL")

	if clientID == "" || clientSecret == "" || applicationURL == "" {
		logger.Error.Fatal("❌ Missing required Google OAuth environment variables (GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, APPLICATION_URL). Check your .env file.")
	}

	redirectURL := applicationURL + "/auth/google/callback"

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	logger.Info.Println("OAuth configuration initialized successfully.")
}

// GoogleLogin redirects users to Google's OAuth login page
func GoogleLogin(c *gin.Context) {
	state := "randomstate" // Ideally, generate a random secure state
	url := oauthConfig.AuthCodeURL(state)
	logger.Info.Printf("Redirecting user to Google OAuth login page: %s", url)
	c.Redirect(http.StatusFound, url)
}

// GoogleCallback handles the OAuth callback and stores user session
func GoogleCallback(c *gin.Context) {
	code := c.Query("code")

	// Exchange authorization code for access token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		logger.Error.Printf("❌ OAuth Exchange failed: %v", err)
		c.String(http.StatusInternalServerError, "Failed to exchange token")
		return
	}

	// Use the token to get user info
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		logger.Error.Printf("❌ Failed to retrieve user info: %v", err)
		c.String(http.StatusInternalServerError, "Failed to get user info")
		return
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			logger.Warn.Printf("Warning: Failed to close response body: %v", err)
		}
	}(resp.Body)

	// Parse user info (e.g., email and name)
	userInfo := struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		logger.Error.Printf("❌ Failed to parse user info: %v", err)
		c.String(http.StatusInternalServerError, "Failed to parse user info")
		return
	}

	// Store user info in session
	session := sessions.Default(c)
	session.Set("user", userInfo.Email)
	if err := session.Save(); err != nil {
		logger.Error.Printf("❌ Failed to save session: %v", err)
	} else {
		logger.Info.Printf("Session saved for user: %s", userInfo.Email)
	}

	logger.Info.Printf("User %s successfully authenticated. Redirecting to home page.", userInfo.Email)
	c.Redirect(http.StatusFound, "/") // Redirect to home page after login
}

// ShowLoginPage redirects users to Google OAuth login
func ShowLoginPage(c *gin.Context) {
	// capture meetId from the query string (eg /login?meetId=meet1)
	meetId := c.Query("meetId")
	if meetId != "" {
		session := sessions.Default(c)
		session.Set("meetId", meetId)
		if err := session.Save(); err != nil {
			logger.Error.Printf("❌ Failed to save session: %v", err)
		} else {
			logger.Info.Printf("Stored meetId %s in session", meetId)
		}
	}
	logger.Info.Println("Redirecting to Google OAuth login page (ShowLoginPage)")
	c.Redirect(http.StatusFound, "/auth/google/login")
}

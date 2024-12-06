// controllers/auth_controller.go
package controllers

import (
	"context"
	"encoding/json"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io"
	"log"
	"net/http"
	"os"
)

var oauthConfig *oauth2.Config

func init() {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURL := os.Getenv("APPLICATION_URL") + "/auth/google/callback"

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
}

func GoogleLogin(c *gin.Context) {
	// Generate a random state string for security
	state := "randomstate" // Here we just use a static string for demonstration
	url := oauthConfig.AuthCodeURL(state)
	c.Redirect(http.StatusFound, url)
}

func GoogleCallback(c *gin.Context) {
	//state := c.Query("state")
	code := c.Query("code")

	// Check state if needed, here we assume it's always correct
	// Exchange code for token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("oauthConfig.Exchange() failed: %v", err)
		c.String(http.StatusInternalServerError, "Failed to exchange token")
		return
	}
	// Use the token to get user info
	client := oauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed getting user info: %v", err)
		c.String(http.StatusInternalServerError, "Failed to get user info")
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}(resp.Body)
	// Parse user info (for example as a JSON with email and name)
	userInfo := struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("Failed to decode user info: %v", err)
		c.String(http.StatusInternalServerError, "Failed to parse user info")
		return
	}
	// Store user info in session
	session := sessions.Default(c)
	session.Set("user", userInfo.Email) // or store the entire user object as needed
	session.Save()

	c.Redirect(http.StatusFound, "/") // Redirect to the main page or dashboard
}

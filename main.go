// main.go
package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"go-ref-lights/controllers"
	"go-ref-lights/middleware"
	"go-ref-lights/websocket"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	// Set Gin to release mode for production (optional but recommended)
	gin.SetMode(gin.ReleaseMode)

	// Initialize the router
	router := gin.Default()

	// Set X-Frame-Options header to allow embedding
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set(
			"X-Frame-Options",
			"ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// Add this route for health checks
	router.GET("/health", controllers.Health)

	// Read configuration from environment variables
	applicationURL := os.Getenv("APPLICATION_URL")
	if applicationURL == "" {
		applicationURL = "http://localhost:8080" // Default to localhost for local testing
	}

	websocketURL := os.Getenv("WEBSOCKET_URL")
	if websocketURL == "" {
		websocketURL = "ws://localhost:8080/referee-updates" // Default to localhost for local testing
	}

	// Pass these values to controllers or wherever needed
	controllers.SetConfig(applicationURL, websocketURL)

	// Initialize session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))

	// Determine the absolute path to the templates directory
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	templatesDir := filepath.Join(basepath, "templates", "*.html") // Corrected path

	// Load HTML templates
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(templatesDir)

	// Serve static files under /static
	router.Static("/static", "./static")

	// Serve favicon.ico
	router.GET("/favicon.ico", func(c *gin.Context) {
		c.File("/static/images/favicon.ico") // Adjust the path if necessary
	})

	// Public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)
	router.GET("/logout", controllers.Logout) // Added Logout route

	// Google Auth routes
	router.GET("/auth/google/login", controllers.GoogleLogin)
	router.GET("/auth/google/callback", controllers.GoogleCallback)

	// Protected routes
	protected := router.Group("/", middleware.AuthRequired)
	{
		protected.GET("/", controllers.Index)
		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/referee-updates", controllers.RefereeUpdates)
	}

	// Start the WebSocket handler
	go websocket.HandleMessages()

	// Start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

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
	"os"
	"path/filepath"
	"runtime"
)

func main() {
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
		applicationURL = "https://referee-lights.michaelkingston.com.au"
	}

	websocketURL := os.Getenv("WEBSOCKET_URL")
	if websocketURL == "" {
		websocketURL = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	}

	// Pass these values to controllers or wherever needed
	controllers.SetConfig(applicationURL, websocketURL)

	// Initialize session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))

	// Determine the absolute path to the templates directory
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	templatesDir := filepath.Join(basepath, "static", "templates", "*.html") // Updated path

	// Load HTML templates
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(templatesDir)

	// Static files
	router.Static("/static", "./static")

	// Public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)

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

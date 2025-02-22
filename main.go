// main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go-ref-lights/controllers"
	"go-ref-lights/middleware"
	"go-ref-lights/websocket"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	}

	// Set Gin to release mode for production (optional but recommended)
	gin.SetMode(gin.ReleaseMode)

	// Initialize the router
	router := gin.Default()

	// Read environment variables
	applicationURL := os.Getenv("APPLICATION_URL")
	if applicationURL == "" {
		applicationURL = "http://localhost:8080"
	}

	websocketURL := os.Getenv("WEBSOCKET_URL")
	if websocketURL == "" {
		websocketURL = "ws://localhost:8080/referee-updates"
	}

	// Pass these values to controllers
	controllers.SetConfig(applicationURL, websocketURL)

	// Set security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set(
			"X-Frame-Options",
			"ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// Add health check route
	router.GET("/health", controllers.Health)

	// Initialize session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("GIN_MODE") == "release", // Secure in production
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))

	// Determine absolute path for templates
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	templatesDir := filepath.Join(basepath, "templates")

	// Validate that the templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("Templates directory does not exist: %s", templatesDir)
	}

	// Load HTML templates
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	// Serve static files
	router.Static("/static", "./static")

	// Serve favicon.ico
	router.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join(basepath, "static", "images", "favicon.ico")
		c.File(faviconPath)
	})

	// Public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)
	router.GET("/logout", controllers.Logout)
	router.GET("/positions", controllers.ShowPositionsPage)
	router.POST("/position/claim", controllers.ClaimPosition)

	// Google Auth routes
	router.GET("/auth/google/login", controllers.GoogleLogin)
	router.GET("/auth/google/callback", controllers.GoogleCallback)

	// Protected routes
	protected := router.Group("/", middleware.AuthRequired, middleware.PositionRequired())
	{
		protected.GET("/", controllers.Index)
		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)
	}

	// WebSocket Route for Live Updates
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// Start the WebSocket message handler in a separate goroutine
	go websocket.HandleMessages()

	// Start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// main.go
package main

import (
	"encoding/json"
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

// We'll store a simple struct for upcoming meets
var upcomingMeets []struct {
	Date     string `json:"date"`
	MeetName string `json:"meetName"`
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	}

	// Set Gin to release mode for production (optional)
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

	// Pass these values to controllers (used by some templates)
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
		// In production, you'd want Secure=true if using HTTPS
		Secure:   os.Getenv("GIN_MODE") == "release",
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))

	// Load upcoming_meets.json
	err = loadUpcomingMeetsJSON("upcoming_meets.json")
	if err != nil {
		log.Fatalf("Failed to load meets JSON: %v", err)
	}

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

	// 1) Root route: If no meet is chosen, redirect to /select-meet
	//    Otherwise, go to the normal index page
	router.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("selectedMeet") == nil {
			// Force them to pick a meet first
			c.Redirect(http.StatusFound, "/select-meet")
			return
		}
		// If a meet is chosen, just show the normal index page
		controllers.Index(c) // or redirect to /positions, your call
	})

	// Public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)
	router.GET("/logout", controllers.Logout)

	// Google Auth routes
	router.GET("/auth/google/login", controllers.GoogleLogin)
	router.GET("/auth/google/callback", controllers.GoogleCallback)

	// 2) Show the "Select Meet" page
	router.GET("/select-meet", func(c *gin.Context) {
		c.HTML(http.StatusOK, "select_meet.html", gin.H{
			"Meets": upcomingMeets,
		})
	})

	// 3) Handle form POST from /select-meet
	router.POST("/select-meet", func(c *gin.Context) {
		chosenMeet := c.PostForm("meetName")
		session := sessions.Default(c)
		session.Set("selectedMeet", chosenMeet)
		_ = session.Save()

		// Then redirect them to /positions or wherever
		c.Redirect(http.StatusFound, "/positions")
	})

	// Protected routes: Must be logged in, have a position, AND have selected a meet
	protected := router.Group("/",
		middleware.AuthRequired,
		middleware.PositionRequired(),
		ensureMeetSelected(),
	)
	{
		protected.GET("/positions", controllers.ShowPositionsPage)
		protected.POST("/position/claim", controllers.ClaimPosition)

		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/referee-updates", controllers.RefereeUpdates)
	}

	// Start the WebSocket message handler in a separate goroutine
	go websocket.HandleMessages()

	// Finally, run the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// loadUpcomingMeetsJSON loads the upcoming_meets.json file into our upcomingMeets slice
func loadUpcomingMeetsJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &upcomingMeets)
}

// ensureMeetSelected is middleware that forces the user to pick a meet
// before accessing the protected routes.
func ensureMeetSelected() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("selectedMeet") == nil {
			c.Redirect(http.StatusFound, "/select-meet")
			c.Abort()
			return
		}
		c.Next()
	}
}

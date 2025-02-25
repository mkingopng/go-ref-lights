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

var upcomingMeets []struct {
	Date     string `json:"date"`
	MeetName string `json:"meetName"`
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	} else {
		log.Println("Loaded .env file successfully.")
	}
	log.Printf("APPLICATION_URL: %s", os.Getenv("APPLICATION_URL"))
	log.Printf("WEBSOCKET_URL: %s", os.Getenv("WEBSOCKET_URL"))

	// Production config or dev
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	applicationURL := os.Getenv("APPLICATION_URL")
	if applicationURL == "" {
		applicationURL = "http://localhost:8080"
	}
	log.Printf("Using APPLICATION_URL: %s", applicationURL)

	websocketURL := os.Getenv("WEBSOCKET_URL")
	if websocketURL == "" {
		websocketURL = "ws://localhost:8080/referee-updates"
	}
	log.Printf("Using WEBSOCKET_URL: %s", websocketURL)

	// for your templates
	controllers.SetConfig(applicationURL, websocketURL)
	log.Println("Controller configuration set.")

	// security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set(
			"X-Frame-Options",
			"ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// health
	router.GET("/health", controllers.Health)

	// sessions
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   os.Getenv("GIN_MODE") == "release",
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mysession", store))
	log.Printf("Session store = %v", store)

	// load upcoming meets
	log.Println("Loading upcoming meets from JSON...")
	err = loadUpcomingMeetsJSON("upcoming_meets.json")
	if err != nil {
		log.Fatalf("Failed to load meets JSON: %v", err)
	}
	log.Printf("Loaded %d upcoming meets.", len(upcomingMeets))

	// templates
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	templatesDir := filepath.Join(basepath, "templates")

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("Templates directory does not exist: %s", templatesDir)
	}
	fmt.Println("Templates Path:", templatesDir)
	log.Println("Templates loaded from:", templatesDir)

	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))
	router.Static("/static", "./static")

	router.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join(basepath, "static", "images", "favicon.ico")
		c.File(faviconPath)
	})

	// 1) Force the user to see SELECT-MEET first
	router.GET("/", func(c *gin.Context) {
		// Always redirect to /select-meet
		log.Printf("Root request received, redirecting to /select-meet")
		c.Redirect(http.StatusFound, "/select-meet")
	})

	// 2) "Select Meet" page
	router.GET("/select-meet", func(c *gin.Context) {
		log.Printf("Select Meet page requested")
		c.HTML(http.StatusOK, "select_meet.html", gin.H{
			"Meets": upcomingMeets,
		})
	})

	router.POST("/select-meet", func(c *gin.Context) {
		chosenMeet := c.PostForm("meetName")
		log.Printf("Meet selected: %s", chosenMeet)
		session := sessions.Default(c)
		session.Set("selectedMeet", chosenMeet)
		_ = session.Save()
		c.Redirect(http.StatusFound, "/index")
	})

	// 3) /index route that displays the actual home/Index page
	router.GET("/index", func(c *gin.Context) {
		session := sessions.Default(c)
		if session.Get("selectedMeet") == nil {
			log.Println("No meet selected in session; redirecting to /select-meet")
			// if meet not chosen, back to select-meet
			c.Redirect(http.StatusFound, "/select-meet")
			return
		}
		// otherwise, show the normal index
		log.Printf("Displaying index for meet: %v", session.Get("selectedMeet"))
		controllers.Index(c)
	})

	// Public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)
	router.GET("/logout", controllers.Logout)

	// Google Auth
	router.GET("/auth/google/login", controllers.GoogleLogin)
	router.GET("/auth/google/callback", controllers.GoogleCallback)

	// protected routes
	protected := router.Group("/", middleware.AuthRequired, middleware.PositionRequired())
	{
		protected.GET("/positions", controllers.ShowPositionsPage)
		protected.POST("/position/claim", controllers.ClaimPosition)
		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)

	// webSocket Route for Live Updates
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// start the WebSocket message handler in a separate goroutine
	go websocket.HandleMessages()

	// start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// loadUpcomingMeetsJSON
func loadUpcomingMeetsJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &upcomingMeets)
}

// ensureMeetSelected middleware
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

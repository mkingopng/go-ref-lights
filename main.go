// main.go
package main

import (
	"fmt"
	"go-ref-lights/services"
	"io"
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
	"go-ref-lights/logger"
	"go-ref-lights/middleware"
	"go-ref-lights/websocket"
)

func main() {
	// Set Gin to release mode and disable logging
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	creds, err := controllers.LoadMeetCreds()
	if err != nil {
		fmt.Println("Error loading credentials:", err)
	} else {
		fmt.Println("Loaded meets:", creds.Meets)
	}

	// initialise the centralised logger.
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger.Info.Println("[main] Starting application on port :8080")

	// initialise router
	router := gin.Default()
	logger.Info.Println("[main] Setting up routes & sessions...")

	// add logging endpoint:
	router.POST("/log", func(c *gin.Context) {
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			logger.Warn.Printf("Invalid log payload: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}
		// depending on the level, log with the appropriate logger:
		switch payload.Level {
		case "error":
			logger.Error.Println(payload.Message)
		case "warn":
			logger.Warn.Println(payload.Message)
		case "debug":
			//logger.Debug.Println(payload.Message)
		case "info":
			fallthrough
		default:
			logger.Info.Println(payload.Message)
		}
		c.Status(http.StatusOK)
	})

	// use logger.Info, logger.Warn
	logger.Info.Println("Application started successfully.")

	// load environment variables from .env file
	err = godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	}

	// set Gin to release mode for production (optional but recommended)
	gin.SetMode(gin.ReleaseMode)

	// read environment variables
	applicationURL := os.Getenv("APPLICATION_URL")
	if applicationURL == "" {
		applicationURL = "http://localhost:8080"
	}
	websocketURL := os.Getenv("WEBSOCKET_URL")
	if websocketURL == "" {
		websocketURL = "ws://localhost:8080/referee-updates"
	}

	// pass these values to controllers
	controllers.SetConfig(applicationURL, websocketURL)

	// set security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set(
			"X-Frame-Options",
			"ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// add health check route
	router.GET("/health", controllers.Health)

	// initialize session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to false for development (true in production)
		SameSite: http.SameSiteLaxMode,
	})
	router.Use(sessions.Sessions("mySession", store))

	// determine absolute path for templates
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "templates")

	// validate that the templates directory exists
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("Templates directory does not exist: %s", templatesDir)
	}

	// load HTML templates
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	pc := controllers.NewPositionController(&services.OccupancyService{})

	// serve static files
	router.Static("/static", "./static")
	router.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join(basePath, "static", "images", "favicon.ico")
		c.File(faviconPath)
	})

	// public routes
	router.GET("/", controllers.ShowMeets)
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)
	router.GET("/logout", controllers.Logout)

	// middleware: Ensure meetName is set before login
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/meets" || c.Request.URL.Path == "/login" {
			return // Allow meet selection and login pages
		}

		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
		}
	})

	// protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthRequired) // Check auth first
	protected.Use(func(c *gin.Context) {   // Custom middleware to check meetName
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/meets")
			c.Abort()
			return
		}
	})
	protected.Use(middleware.PositionRequired()) // Then check position
	{
		protected.GET("/dashboard", controllers.Index)
		protected.GET("/positions", controllers.ShowPositionsPage)
		protected.POST("/position/claim", pc.ClaimPosition)
		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)
	}

	// webSocket Route for Live Updates
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// start the WebSocket message handler in a separate goroutine
	go websocket.HandleMessages()

	// start the server
	logger.Info.Println("[main] About to run gin server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

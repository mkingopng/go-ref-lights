// main.go
package main

import (
	"fmt"
	"go-ref-lights/services"
	"html/template"
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
	// Load environment variables.
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	applicationURL := "http://localhost:8080"
	websocketURL := "ws://localhost:8080/referee-updates"
	if env == "production" {
		applicationURL = "https://referee-lights.michaelkingston.com.au"
		websocketURL = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	}

	// Pass computed URLs to controllers.
	controllers.SetConfig(applicationURL, websocketURL)

	// Load credentials.
	creds, err := controllers.LoadMeetCreds()
	if err != nil {
		fmt.Println("Error loading credentials:", err)
	} else {
		fmt.Println("Loaded meets:", creds.Meets)
	}

	// Initialize logger.
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger.Info.Println("[main] Starting application on port :8080")

	// Setup the router.
	router := SetupRouter(env)

	// Start the WebSocket message handler in a separate goroutine.
	go websocket.HandleMessages()

	logger.Info.Println("[main] About to run gin server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// SetupRouter creates and configures a Gin router.
func SetupRouter(env string) *gin.Engine {
	// Set Gin mode.
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}
	router := gin.Default()

	// Optionally disable Gin’s own logging.
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	// Configure session store.
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days.
		HttpOnly: true,
		Secure:   env == "production",
	})
	router.Use(sessions.Sessions("mySession", store))

	// Set security headers.
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// Health endpoint.
	router.GET("/health", controllers.Health)

	// Log endpoint.
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
		switch payload.Level {
		case "error":
			logger.Error.Println(payload.Message)
		case "warn":
			logger.Warn.Println(payload.Message)
		case "debug":
			logger.Debug.Println(payload.Message)
		case "info":
			fallthrough
		default:
			logger.Info.Println(payload.Message)
		}
		c.Status(http.StatusOK)
	})

	occupancyService := &services.OccupancyService{}
	pc := controllers.NewPositionController(occupancyService)

	// Public routes.
	router.GET("/", controllers.ShowMeets)
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)
	router.GET("/logout", controllers.Logout)
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*.html")))

	// Middleware to ensure "meetName" is set.
	router.Use(func(c *gin.Context) {
		if c.Request.URL.Path == "/meets" || c.Request.URL.Path == "/login" {
			return
		}
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
	})

	// Protected routes.
	protected := router.Group("/")
	protected.Use(middleware.AuthRequired)
	protected.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/meets")
			c.Abort()
			return
		}
		c.Next()
	})
	protected.Use(middleware.PositionRequired())
	{
		protected.GET("/dashboard", controllers.Index)
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/positions", controllers.ShowPositionsPage)
		protected.POST("/position/claim", pc.ClaimPosition)
		protected.GET("/left", controllers.Left)
		protected.GET("/center", controllers.Center)
		protected.GET("/right", controllers.Right)
		protected.GET("/occupancy", pc.GetOccupancyAPI)
		protected.POST("/position/vacate", pc.VacatePosition)

	}

	// WebSocket route.
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// Serve static files.
	router.Static("/static", "./static")

	// Determine the absolute path for templates.
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("Templates directory does not exist: %s", templatesDir)
	}
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(filepath.Join(templatesDir, "*.html"))

	return router
}

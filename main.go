// main.go
package main

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go-ref-lights/controllers"
	"go-ref-lights/heartbeat"
	"go-ref-lights/logger"
	"go-ref-lights/middleware"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// GinHeartbeatHandler is a wrapper that calls HeartbeatHandler from your heartbeat.go file
func GinHeartbeatHandler(c *gin.Context) {
	heartbeat.HeartbeatHandler(c.Writer, c.Request)
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		logger.Warn.Println("[main] No .env file found. Using system environment variables.")
	}

	// Determine the environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	// Set your logging level based on environment
	logger.SetLogLevel(env)

	// Log the environment
	logger.Info.Printf("[main] Running in %s mode", env)

	// Set application & websocket URLs based on environment
	var applicationURL, websocketURL string
	if env == "production" {
		applicationURL = "https://referee-lights.michaelkingston.com.au"
		websocketURL = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	} else {
		applicationURL = "http://0.0.0.0:8080"
		websocketURL = "ws://0.0.0.0:8080/referee-updates"
	}

	// Pass computed URLs to controllers
	controllers.SetConfig(applicationURL, websocketURL)

	// Load credentials
	creds, err := controllers.LoadMeetCreds()
	if err != nil {
		logger.Error.Printf("[main] Error loading credentials: %v", err)
	} else {
		logger.Info.Printf("[main] Loaded meets: %+v", creds.Meets)
	}

	// Announce start
	logger.Info.Println("[main] Starting application on port :8080")

	// Setup the router
	router := SetupRouter(env)

	// Start background routines
	hbManager := heartbeat.NewHeartbeatManager()
	go hbManager.CleanupInactiveSessions(30 * time.Second)
	go websocket.HandleMessages()

	router.GET("/heartbeat", GinHeartbeatHandler)

	// Read host/port from environment or default
	host := os.Getenv("APP_HOST")
	if host == "" {
		if env == "production" {
			host = "0.0.0.0"
		} else {
			host = "localhost"
		}
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	addr := host + ":" + port

	// Create an HTTP server with timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Info.Printf("[main] Server running on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		// If the server fails to start, we can log a fatal error
		log.Fatalf("[main] Failed to start server: %v", err)
	}
}

// SetupRouter creates and configures a Gin router.
func SetupRouter(env string) *gin.Engine {
	// Configure Gin mode
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}
	router := gin.Default()

	// Serve /favicon.ico directly
	router.StaticFile("/favicon.ico", "./static/images/favicon.ico")

	// Reduce logs in non-production
	if env != "production" {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		logger.Debug.Println("[SetupRouter] Gin logs have been discarded for non-production mode.")
	}

	// Configure session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   true,
	})
	router.Use(sessions.Sessions("mySession", store))

	// Set security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// Disable caching for all responses
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, must-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Next()
	})

	// Health endpoint
	router.GET("/health", controllers.Health)

	// Log endpoint
	router.POST("/log", func(c *gin.Context) {
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			logger.Warn.Printf("[SetupRouter /log] Invalid log payload: %v", err)
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

	// Initialize your service layer
	occupancyService := services.NewOccupancyService()

	// build the SudoController
	sudoController := controllers.NewSudoController(occupancyService)
	sudoRoutes := router.Group("/sudo")
	{
		// Must be logged in
		sudoRoutes.Use(middleware.AuthRequired)
		// Must be superuser
		sudoRoutes.Use(middleware.SudoRequired())
		{
			sudoRoutes.GET("/", sudoController.SudoPanel)
			sudoRoutes.POST("/force-vacate-ref", sudoController.ForceVacateRefForAnyMeet)
			sudoRoutes.POST("/force-logout-meet-director", sudoController.ForceLogoutMeetDirector)
			sudoRoutes.POST("/restart-meet", sudoController.RestartAndClearMeet)
		}
	}

	// define other controllers
	positionController := controllers.NewPositionController(occupancyService)
	adminController := controllers.NewAdminController(occupancyService, positionController)
	pc := controllers.NewPositionController(occupancyService)

	// Public routes
	router.GET("/", controllers.ShowMeets)
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/meet", controllers.MeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)
	router.GET("/index", controllers.Index)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		controllers.RefereeHandler(c, occupancyService)
	})

	// Load templates
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*.html")))

	// Ensure "meetName" is set (except for a few routes)
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

	// Protected routes
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
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/positions", controllers.ShowPositionsPage)
		protected.POST("/position/claim", pc.ClaimPosition)
		protected.GET("/left", controllers.Left)
		protected.GET("/center", controllers.Center)
		protected.GET("/right", controllers.Right)
		protected.GET("/occupancy", pc.GetOccupancyAPI)
		protected.POST("/position/vacate", pc.VacatePosition)

		// If you restore your /home routes:
		// protected.GET("/home", func(c *gin.Context) {
		//     controllers.Home(c, occupancyService)
		// })
		// protected.POST("/home", func(c *gin.Context) {
		//     controllers.Home(c, occupancyService)
		// })

		protected.POST("/logout", func(c *gin.Context) {
			controllers.Logout(c, occupancyService)
		})
		protected.GET("/logout", func(c *gin.Context) {
			controllers.Logout(c, occupancyService)
		})
		protected.POST("/force-logout", controllers.ForceLogoutHandler)
		protected.GET("/active-users", controllers.ActiveUsersHandler)
	}

	// Admin routes
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AdminRequired())
	{
		adminRoutes.GET("", adminController.AdminPanel)
		adminRoutes.POST("/force-vacate", adminController.ForceVacate)
		adminRoutes.POST("/reset-instance", adminController.ResetInstance)
	}

	// WebSocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// Serve static files
	router.Static("/static", "./static")

	// Confirm templates path
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("[SetupRouter] Templates directory does not exist: %s", templatesDir)
	}

	logger.Debug.Printf("[SetupRouter] Templates Path: %s", templatesDir)
	return router
}

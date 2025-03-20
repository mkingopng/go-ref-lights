// main.go
package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go-ref-lights/controllers"
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

func GinHeartbeatHandler(c *gin.Context) {
	HeartbeatHandler(c.Writer, c.Request)
}

func main() {
	// load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found. Using system environment variables.")
	}

	// determine the environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "production" // Default to production
	}
	fmt.Println("Running in", env, "mode")

	// set application and websocket URLs
	var applicationURL, websocketURL string
	if env == "production" {
		applicationURL = "https://referee-lights.michaelkingston.com.au"
		websocketURL = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	} else {
		applicationURL = "http://localhost:8080"
		websocketURL = "ws://localhost:8080/referee-updates"
	}

	// pass computed URLs to controllers
	controllers.SetConfig(applicationURL, websocketURL)

	// load credentials
	creds, err := controllers.LoadMeetCreds()
	if err != nil {
		fmt.Println("Error loading credentials:", err)
	} else {
		fmt.Println("Loaded meets:", creds.Meets)
	}

	// initialize logger
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	logger.Info.Println("[main] Starting application on port :8080")

	// setup the router
	router := SetupRouter(env)

	// start background routines
	hbManager := NewHeartbeatManager()
	go hbManager.CleanupInactiveSessions(30 * time.Second)
	go websocket.HandleMessages()

	router.GET("/heartbeat", GinHeartbeatHandler)

	// read host/port from environment
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
	addr := fmt.Sprintf("%s:%s", host, port)

	// create an HTTP server with timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Info.Printf("Server running on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// SetupRouter creates and configures a Gin router
func SetupRouter(env string) *gin.Engine {
	// set Gin mode
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}
	router := gin.Default()

	// optionally reduce logs in non-production
	if env != "production" {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
	}

	// configure session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   true, // use secure cookies
	})
	router.Use(sessions.Sessions("mySession", store))

	// set security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// -----------------------------------------------------------------------
	// NEW: disable caching for all responses
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, must-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Next()
	})
	// -----------------------------------------------------------------------

	// health endpoint
	router.GET("/health", controllers.Health)

	// log endpoint
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

	occupancyService := services.NewOccupancyService()
	positionController := controllers.NewPositionController(occupancyService)
	adminController := controllers.NewAdminController(occupancyService, positionController)
	pc := controllers.NewPositionController(occupancyService)

	// public routes
	router.GET("/", controllers.ShowMeets)
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/meet", controllers.MeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)
	router.GET("/index", controllers.Index)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) {
		controllers.RefereeHandler(c, occupancyService)
	})
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*.html")))

	// ensure "meetName" is set
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

	// protected routes
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
		protected.GET("/home", func(c *gin.Context) {
			controllers.Home(c, occupancyService)
		})
		protected.POST("/home", func(c *gin.Context) {
			controllers.Home(c, occupancyService)
		})
		protected.POST("/logout", func(c *gin.Context) {
			controllers.Logout(c, occupancyService)
		})
		protected.GET("/logout", func(c *gin.Context) {
			controllers.Logout(c, occupancyService)
		})
		protected.POST("/force-logout", controllers.ForceLogoutHandler)
		protected.GET("/active-users", controllers.ActiveUsersHandler)
	}

	// admin routes
	adminRoutes := router.Group("/admin")
	adminRoutes.Use(middleware.AdminRequired())
	{
		adminRoutes.GET("", adminController.AdminPanel)
		adminRoutes.POST("/force-vacate", adminController.ForceVacate)
		adminRoutes.POST("/reset-instance", adminController.ResetInstance)
	}

	// websocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// serve static files
	router.Static("/static", "./static")

	// check templates path
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("Templates directory does not exist: %s", templatesDir)
	}
	fmt.Println("Templates Path:", templatesDir)
	return router
}

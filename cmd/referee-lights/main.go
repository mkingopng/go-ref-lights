// Package main
// File: cmd/referee-lights/main.go
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"go-ref-lights/controllers"
	"go-ref-lights/logger"
	"go-ref-lights/middleware"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

// SetupRouter creates and configures a Gin router.
func SetupRouter(env string) *gin.Engine {
	// configure Gin mode
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	router := gin.Default()

	// serve static files
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/images/favicon.ico")

	// reduce logs in non-production
	if env != "production" {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		logger.Debug.Println("[SetupRouter] Gin logs have been discarded for non-production mode.")
	}

	// configure session store for meet-director login
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: true,
		Secure:   true,
	})
	router.Use(sessions.Sessions("mySession", store))

	// basic security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// disable HTTP caching
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, must-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Next()
	})

	// health endpoint
	router.GET("/health", controllers.Health)

	// simple log endpoint
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
		default: // "info" + any unknown
			logger.Info.Println(payload.Message)
		}
		c.Status(http.StatusOK)
	})

	// create services and controllers
	occupancyService := services.NewOccupancyService()
	positionController := controllers.NewPositionController(occupancyService)
	meetDirectorController := controllers.NewAdminController(occupancyService, positionController)

	// ------------------ public routes ------------------
	router.GET("/", controllers.ChooseMeetHandler)
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)
	router.GET("/logged-out", func(c *gin.Context) {
		// render a super simple page saying "You are now logged out"
		c.HTML(http.StatusOK, "logged-out.html", gin.H{
			"Title": "You are now logged out",
		})
	})
	router.GET("/referee/:meetName/:position", func(c *gin.Context) { controllers.RefereeHandler(c, occupancyService) })
	router.GET("/heartbeat", func(c *gin.Context) { Handler(c.Writer, c.Request) })
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*.html")))

	router.Use(func(c *gin.Context) {
		// skip for static
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Next()
			return
		}

		// skip for /login, /logout, /referee-updates
		if c.Request.URL.Path == "/login" ||
			c.Request.URL.Path == "/logout" ||
			c.Request.URL.Path == "/referee-updates" {
			c.Next()
			return
		}

		// otherwise enforce meetName in session
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	})

	// ------------------ protected meet director routes ------------------
	protected := router.Group("/")
	protected.Use(middleware.AuthRequired)
	protected.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
	})
	{
		protected.GET("/index", controllers.Index)
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/occupancy", positionController.GetOccupancyAPI)
		protected.POST("/position/vacate", positionController.VacatePosition)
		protected.GET("/active-users", controllers.ActiveUsersHandler)
		protected.GET("/admin-panel", meetDirectorController.AdminPanel)
		protected.POST("/force-vacate", meetDirectorController.ForceVacate)
		protected.POST("/reset-instance", meetDirectorController.ResetInstance)
		protected.GET("/logout", func(c *gin.Context) { controllers.Logout(c, occupancyService) })
		protected.POST("/logout", func(c *gin.Context) { controllers.Logout(c, occupancyService) })
	}

	// ------------------ sudo routes ------------------
	sudoController := controllers.NewSudoController(occupancyService)
	sudoRoutes := router.Group("/sudo")
	{
		sudoRoutes.Use(middleware.AuthRequired)
		sudoRoutes.Use(middleware.SudoRequired())
		{
			sudoRoutes.GET("/", sudoController.SudoPanel)
			sudoRoutes.POST("/force-vacate-ref", sudoController.ForceVacateRefForAnyMeet)
			sudoRoutes.POST("/force-logout-meet-director", sudoController.ForceLogoutMeetDirector)
			sudoRoutes.POST("/restart-meet", sudoController.RestartAndClearMeet)
		}
	}

	// WebSocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// confirm template path
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "../../templates")
	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("[SetupRouter] Templates directory does not exist: %s", templatesDir)
	}
	router.SetHTMLTemplate(template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html"))))
	logger.Debug.Printf("[SetupRouter] Templates Path: %s", templatesDir)
	return router
}

var (
	refereeSessions = make(map[string]time.Time)
	sessionLock     = sync.Mutex{}
)

// Manager tracks active referees
type Manager struct {
	activeSessions map[string]time.Time
	mu             sync.Mutex
}

// Handler updates the last seen timestamp of a referee
func Handler(w http.ResponseWriter, r *http.Request) {
	// Start a subsegment, but it may return nil if there's no parent segment
	ctx, seg := xray.BeginSubsegment(r.Context(), "HeartbeatHandler")
	if seg != nil {
		defer seg.Close(nil)
	}

	// update request's context in case anything else reads it
	r = r.WithContext(ctx)

	refereeID := r.URL.Query().Get("referee_id")
	if refereeID != "" && seg != nil {
		_ = seg.AddAnnotation("refereeID", refereeID)
	}

	if refereeID == "" {
		logger.Warn.Println("[Handler] Missing referee ID in query params")
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	sessionLock.Lock()
	refereeSessions[refereeID] = time.Now()
	sessionLock.Unlock()

	logger.Debug.Printf("[Handler] Updated heartbeat for referee=%s at %v", refereeID, time.Now())

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "Heartbeat received"); err != nil {
		logger.Warn.Printf("[Handler] Error writing response for referee=%s: %v", refereeID, err)
	}
}

// NewHeartbeatManager initializes a heartbeat tracker
func NewHeartbeatManager() *Manager {
	return &Manager{
		activeSessions: make(map[string]time.Time),
	}
}

// UpdateHeartbeat marks a referee as active
func (h *Manager) UpdateHeartbeat(refereeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeSessions[refereeID] = time.Now()
	logger.Debug.Printf("[Manager.UpdateHeartbeat] Referee=%s updated at %v", refereeID, time.Now())
}

// CleanupInactiveSessions removes inactive referees
func (h *Manager) CleanupInactiveSessions(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			for id, lastSeen := range h.activeSessions {
				if time.Since(lastSeen) > timeout {
					logger.Info.Printf("[Manager.CleanupInactiveSessions] Removing inactive referee=%s (timeout=%v)", id, timeout)
					delete(h.activeSessions, id)
				}
			}
			h.mu.Unlock()
		}
	}()
}

func main() {
	// load env variables
	_ = godotenv.Load()

	// explicitly initialise the logger
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// determine the environment
	env := os.Getenv("ENV")
	if env == "" {
		env = "production"
	}
	logger.SetLogLevel(env)

	// optionally defer close:
	defer func() {
		if err := logger.CloseLogger(); err != nil {
			log.Printf("Error closing logger: %v", err)
		}
	}()

	// log the environment
	logger.Info.Printf("[main] Running in %s mode", env)

	// set application & websocket URLs based on environment
	var applicationURL, websocketURL string
	if env == "production" {
		applicationURL = "https://referee-lights.michaelkingston.com.au"
		websocketURL = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	} else {
		applicationURL = "http://0.0.0.0:8080"
		websocketURL = "ws://0.0.0.0:8080/referee-updates"
	}

	// pass computed URLs to controllers
	controllers.SetConfig(applicationURL, websocketURL)

	// load credentials
	creds, err := services.LoadMeetCredentials()
	if err != nil {
		logger.Error.Printf("[main] Error loading credentials: %v", err)
	} else {
		services.SetGlobalMeetCredentials(creds)
		logger.Info.Printf("[main] Loaded meets: %+v", creds.Meets)
	}

	// announce start
	logger.Info.Println("[main] Starting application on port :8080")

	// setup the router
	router := SetupRouter(env)

	// optional: Set X-Ray config
	err = xray.Configure(xray.Config{
		ServiceVersion: "1.0.0",
	})
	if err != nil {
		return
	}

	// wrap router with X-Ray middleware
	xraySegmentNamer := xray.NewFixedSegmentNamer("RefereeLightsApp")
	xrayHandler := xray.Handler(xraySegmentNamer, router)

	// start background routines
	hbManager := NewHeartbeatManager()
	go hbManager.CleanupInactiveSessions(30 * time.Second)
	go websocket.HandleMessages()

	// read host/port from environment or default
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

	// create an HTTP server with timeouts
	server := &http.Server{
		Addr:         addr,
		Handler:      xrayHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	logger.Info.Printf("[main] Server running on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		// if the server fails to start, we can log a fatal error
		log.Fatalf("[main] Failed to start server: %v", err)
	}
}

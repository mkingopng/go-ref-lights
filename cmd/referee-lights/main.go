// Package main
// File: cmd/referee-lights/main.go
package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

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
		MaxAge:   5400,
		HttpOnly: true, // In production, set Secure: true (requires HTTPS)
		Secure:   true,
		SameSite: http.SameSiteNoneMode, // Ensure that your environment actually supports cross-site usage if needed
	})
	router.Use(sessions.Sessions("mySession", store))

	// basic security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Content-Security-Policy", "frame-ancestors 'self' https://referee-lights.michaelkingston.com.au;")
		c.Next()
	})

	// ---------------- Security Headers Middleware ----------------
	router.Use(func(c *gin.Context) {
		// Strict-Transport-Security (only if you serve HTTPS in prod)
		// tells browsers to only connect via HTTPS for the next 5400 seconds
		if env == "production" {
			c.Writer.Header().Set("Strict-Transport-Security", "max-age=5400; includeSubDomains; preload")
		}

		// Content-Security-Policy (example: limit frames to self and a specific domain)
		// Adjust frame-ancestors or other directives as needed:
		c.Writer.Header().Set("Content-Security-Policy", "frame-ancestors 'self' https://referee-lights.michaelkingston.com.au;")

		// X-Frame-Options (older header for clickjacking protection) — SAMEORIGIN or DENY are common
		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// X-Content-Type-Options helps prevent MIME-type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer-Policy (control what referrer info is sent)
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy (formerly Feature-Policy): restrict camera/mic, etc.
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// End security headers; proceed to next handler
		c.Next()
	})

	// disable HTTP caching (optional: suitable if you want no caching of dynamic pages)
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

		// parse the JSON payload
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}

		// bind JSON payload
		if err := c.ShouldBindJSON(&payload); err != nil {
			logger.Warn.Printf("[SetupRouter /log] Invalid log payload: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		// log the message based on level
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
	router.GET("/", controllers.ChooseMeetHandler)       // meet director
	router.POST("/set-meet", controllers.SetMeetHandler) // meet director
	router.GET("/login", controllers.PerformLogin)       // meet director
	router.POST("/login", controllers.LoginHandler)      // meet director
	router.POST("/force-my-login", controllers.ForceMyLogin)
	router.GET("/referee/:meetName/:position", func(c *gin.Context) { controllers.RefereeHandler(c, occupancyService) }) // referee
	router.GET("/heartbeat", func(c *gin.Context) { Handler(c.Writer, c.Request) })                                      // all devices and users
	router.GET("/logged-out", func(c *gin.Context) {
		c.HTML(http.StatusOK, "logged-out.html", gin.H{"Title": "You are now logged out"})
	}) // referee and meet director
	// Enforcement: require meetName in session for protected routes
	router.Use(func(c *gin.Context) {
		// skip if static or login/logout
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Next()
			return
		}

		// let "/referee/:meetName/:position" proceed
		if strings.HasPrefix(c.Request.URL.Path, "/referee/") {
			c.Next()
			return
		}

		if c.Request.URL.Path == "/login" ||
			c.Request.URL.Path == "/logout" ||
			c.Request.URL.Path == "/referee-updates" {
			c.Next()
			return
		}

		// otherwise enforce that meetName is in session
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
		protected.GET("/index", controllers.Index)                                                  // meet director
		protected.GET("/qrcode", controllers.GetQRCode)                                             // meet director
		protected.GET("/lights", controllers.Lights)                                                // meet director
		protected.GET("/occupancy", positionController.GetOccupancyAPI)                             // meet director
		protected.POST("/position/vacate", positionController.VacatePosition)                       // meet director
		protected.GET("/active-users", controllers.ActiveUsersHandler)                              // meet director
		protected.GET("/admin", meetDirectorController.AdminPanel)                                  // meet director
		protected.POST("/force-vacate", meetDirectorController.ForceVacate)                         // meet director
		protected.POST("/reset-instance", meetDirectorController.ResetInstance)                     // meet director
		protected.GET("/system-health", meetDirectorController.GetSystemHealthMetrics)              // meet director
		protected.GET("/logout", func(c *gin.Context) { controllers.Logout(c, occupancyService) })  // meet director
		protected.POST("/logout", func(c *gin.Context) { controllers.Logout(c, occupancyService) }) // meet director
	}

	// ------------------ sudo routes ------------------
	sudoController := controllers.NewSudoController(occupancyService) // sudo
	sudoRoutes := router.Group("/sudo")
	{
		sudoRoutes.Use(middleware.AuthRequired)   //sudo
		sudoRoutes.Use(middleware.SudoRequired()) // sudo
		{
			sudoRoutes.GET("/", sudoController.SudoPanel)                                          // sudo
			sudoRoutes.POST("/force-vacate-ref", sudoController.ForceVacateRefForAnyMeet)          // sudo
			sudoRoutes.POST("/force-logout-meet-director", sudoController.ForceLogoutMeetDirector) // sudo
			sudoRoutes.POST("/restart-meet", sudoController.RestartAndClearMeet)                   // sudo
		}
	}

	// WebSocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	}) // referees

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

// Heartbeat tracking
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

	refereeID := r.URL.Query().Get("referee_id")

	if refereeID == "" {
		logger.Warn.Println("[Handler] Missing referee ID in query params")
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	// Notify WebSocket system about the heartbeat
	// This integrates the separate heartbeat system with the WebSocket connection tracking
	websocket.UpdateRefereeHeartbeat(refereeID)

	// Record system activity for the heartbeat (only if this is a real referee, not a health check agent)
	if !strings.HasPrefix(refereeID, "healthcheck-") {
		websocket.RecordSystemActivity()
	}

	// Keep original behavior for backward compatibility
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
	//logger.SetLogLevel(env)

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

	// start background routines
	hbManager := NewHeartbeatManager()
	go hbManager.CleanupInactiveSessions(30 * time.Second)
	go websocket.HandleMessages()

	// Start the activity tracker for auto-scaling to zero after inactivity
	activityTracker := websocket.GetActivityTracker()
	activityTracker.Start()

	// Set up graceful shutdown
	gracefulShutdown := make(chan os.Signal, 1)
	signal.Notify(gracefulShutdown, syscall.SIGTERM, syscall.SIGINT)

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

	router := SetupRouter(env)

	// create an HTTP server with timeouts
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      router,
	}

	// Start the server in a goroutine
	go func() {
		logger.Info.Printf("[main] Server running on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// if the server fails to start, we can log a fatal error
			logger.Error.Fatalf("[main] Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-gracefulShutdown
	logger.Info.Println("[main] Received shutdown signal, initiating graceful shutdown")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop the activity tracker
	activityTracker.Stop()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		logger.Error.Fatalf("[main] Server shutdown failed: %v", err)
	}

	logger.Info.Println("[main] Server gracefully stopped")
}

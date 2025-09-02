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

// initializeLoggingSystem sets up the logging configuration system with environment-based settings
// and graceful fallback handling for invalid configurations
func initializeLoggingSystem() error {
	// Initialize the logger with environment-based configuration
	if err := logger.InitLogger(); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Validate and log the configuration that was applied
	env := getValidatedEnvironment()
	logLevel := getValidatedLogLevel()

	// Create startup context for configuration logging
	startupContext := logger.NewSystemContext("initialization", "logging")
	startupContext["configuredEnvironment"] = env
	startupContext["configuredLogLevel"] = logLevel

	// Log successful initialization with configuration details
	logger.LogInfoWithContext(startupContext,
		"Logging system initialized successfully (env=%s, level=%s)", env, logLevel)

	// Log any configuration fallbacks that occurred
	if originalEnv := os.Getenv("ENV"); originalEnv != "" && originalEnv != env {
		logger.LogWarnWithContext(startupContext,
			"Invalid ENV value '%s' provided, falling back to '%s'", originalEnv, env)
	}

	if originalLogLevel := os.Getenv("LOG_LEVEL"); originalLogLevel != "" && originalLogLevel != logLevel {
		logger.LogWarnWithContext(startupContext,
			"Invalid LOG_LEVEL value '%s' provided, using environment default '%s'", originalLogLevel, logLevel)
	}

	return nil
}

// getValidatedEnvironment returns a validated environment string with fallback to production
func getValidatedEnvironment() string {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("ENV")))

	// Validate environment value
	switch env {
	case "production", "prod":
		return "production"
	case "development", "dev":
		return "development"
	case "test":
		return "test"
	case "":
		// Default to production for safety when no ENV is set
		return "production"
	default:
		// Invalid environment value, fall back to production for safety
		return "production"
	}
}

// getValidatedLogLevel returns the current effective log level as a string
func getValidatedLogLevel() string {
	if globalLogger := logger.GetGlobalLogger(); globalLogger != nil {
		return globalLogger.GetLevel().String()
	}
	return "WARN" // Default production-safe level
}

// SetupRouter creates and configures a Gin router.
func SetupRouter(env string) *gin.Engine {
	// configure Gin mode
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	router := gin.Default()

	// Add custom HTTP logging middleware (environment-aware)
	router.Use(middleware.HTTPLoggingMiddleware())
	router.Use(middleware.AuthenticationLoggingMiddleware())

	// serve static files
	router.Static("/static", "./static")
	router.StaticFile("/favicon.ico", "./static/images/favicon.ico")

	// Configure Gin logging based on environment with structured logging
	ginContext := logger.NewSystemContext("gin_config", "router")
	ginContext["environment"] = env
	ginContext["ginMode"] = gin.Mode()

	if env == "production" {
		// In production, disable Gin's default logging to reduce noise
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		ginContext["defaultLogging"] = "disabled"
		logger.LogDebugWithContext(ginContext, "Gin framework configured for production (default logging disabled)")
	} else {
		// In development, keep Gin's default logging for debugging
		ginContext["defaultLogging"] = "enabled"
		logger.LogDebugWithContext(ginContext, "Gin framework configured for development (default logging enabled)")
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

		// Content-Security-Policy
		// adjust frame-ancestors or other directives as needed:
		c.Writer.Header().Set("Content-Security-Policy", "frame-ancestors 'self' https://referee-lights.michaelkingston.com.au;")

		// X-Frame-Options (older header for clickjacking protection) — SAMEORIGIN or DENY are common
		c.Writer.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// X-Content-Type-Options helps prevent MIME-type sniffing
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer-Policy (control what referrer info is sent)
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// permissions-Policy (formerly Feature-Policy): restrict camera/mic, etc.
		c.Writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// end security headers; proceed to next handler
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

	// Client-side log endpoint with enhanced structured logging
	router.POST("/log", func(c *gin.Context) {
		// Parse the JSON payload
		var payload struct {
			Message string `json:"message"`
			Level   string `json:"level"`
		}

		// Bind JSON payload with enhanced error context
		if err := c.ShouldBindJSON(&payload); err != nil {
			errorContext := logger.NewHTTPContext("POST", "/log", c.Request.UserAgent(), c.ClientIP(), http.StatusBadRequest)
			errorContext["component"] = "client_logging"
			errorContext["error"] = err.Error()
			errorContext["contentType"] = c.GetHeader("Content-Type")
			logger.LogWarnWithContext(errorContext, "Invalid client log payload received: %v", err)
			c.Status(http.StatusBadRequest)
			return
		}

		// Create enhanced context for client-side logging
		clientLogContext := logger.NewHTTPContext("POST", "/log", c.Request.UserAgent(), c.ClientIP(), http.StatusOK)
		clientLogContext["component"] = "client"
		clientLogContext["clientLogLevel"] = payload.Level
		clientLogContext["messageLength"] = len(payload.Message)

		// Add session context if available
		session := sessions.Default(c)
		if meetName, ok := session.Get("meetName").(string); ok {
			clientLogContext["meetName"] = meetName
		}
		if user, ok := session.Get("user").(string); ok {
			clientLogContext["user"] = user
		}

		// Log the client message with appropriate level and enhanced context
		switch strings.ToLower(payload.Level) {
		case "error":
			logger.LogErrorWithContext(clientLogContext, "Client error: %s", payload.Message)
		case "warn", "warning":
			logger.LogWarnWithContext(clientLogContext, "Client warning: %s", payload.Message)
		case "debug":
			logger.LogDebugWithContext(clientLogContext, "Client debug: %s", payload.Message)
		default: // "info" + any unknown levels
			// Client info messages only logged in development (DEBUG level)
			logger.LogDebugWithContext(clientLogContext, "Client info: %s", payload.Message)
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
	protected.Use(middleware.MeetRequired())
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

	// Configure HTML templates with structured logging
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "../../templates")

	templateContext := logger.NewSystemContext("template_config", "router")
	templateContext["templatesDirectory"] = templatesDir
	templateContext["basePath"] = basePath

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		templateContext["error"] = err.Error()
		logger.LogErrorWithContext(templateContext, "Templates directory does not exist: %s", templatesDir)
	} else {
		router.SetHTMLTemplate(template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html"))))
		logger.LogDebugWithContext(templateContext, "HTML templates loaded successfully from %s", templatesDir)
	}

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
		heartbeatContext := logger.NewHTTPContext("GET", "/heartbeat", r.UserAgent(), r.RemoteAddr, http.StatusBadRequest)
		logger.LogWarnWithContext(heartbeatContext, "Heartbeat request missing referee_id parameter")
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	sessionLock.Lock()
	refereeSessions[refereeID] = time.Now()
	sessionLock.Unlock()

	// Log heartbeat update with structured context (DEBUG level for production noise reduction)
	heartbeatContext := logger.NewHTTPContext("GET", "/heartbeat", r.UserAgent(), r.RemoteAddr, http.StatusOK)
	heartbeatContext["refereeId"] = refereeID
	heartbeatContext["timestamp"] = time.Now()
	logger.LogDebugWithContext(heartbeatContext, "Heartbeat updated for referee %s", refereeID)

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "Heartbeat received"); err != nil {
		errorContext := logger.NewHTTPContext("GET", "/heartbeat", r.UserAgent(), r.RemoteAddr, http.StatusOK)
		errorContext["refereeId"] = refereeID
		errorContext["error"] = err.Error()
		logger.LogWarnWithContext(errorContext, "Error writing heartbeat response for referee %s: %v", refereeID, err)
	}
}

// loadAndValidateMeetCredentials loads and validates meet credentials with proper error handling
func loadAndValidateMeetCredentials() error {
	creds, err := services.LoadMeetCredentials()
	if err != nil {
		return fmt.Errorf("failed to load meet credentials: %w", err)
	}

	// Validate credentials structure
	if len(creds.Meets) == 0 {
		return fmt.Errorf("no meets configured in credentials file")
	}

	// Set global credentials
	services.SetGlobalMeetCredentials(creds)

	// Log successful loading with context
	credentialsContext := logger.NewSystemContext("startup", "credentials")
	credentialsContext["meetCount"] = len(creds.Meets)

	// Extract meet names for logging (avoid logging sensitive data)
	meetNames := make([]string, len(creds.Meets))
	for i, meet := range creds.Meets {
		meetNames[i] = meet.Name
	}
	credentialsContext["meetNames"] = meetNames

	logger.LogInfoWithContext(credentialsContext,
		"Successfully loaded %d meet configurations", len(creds.Meets))

	return nil
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

	// Use structured logging for heartbeat tracking (DEBUG level to avoid production noise)
	heartbeatContext := logger.NewSystemContext("heartbeat", "manager")
	heartbeatContext["refereeId"] = refereeID
	heartbeatContext["timestamp"] = time.Now()
	logger.LogDebugWithContext(heartbeatContext, "Heartbeat manager updated for referee %s", refereeID)
}

// CleanupInactiveSessions removes inactive referees
func (h *Manager) CleanupInactiveSessions(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			for id, lastSeen := range h.activeSessions {
				if time.Since(lastSeen) > timeout {
					// Log session cleanup with structured context
					cleanupContext := logger.NewSystemContext("cleanup", "heartbeat_manager")
					cleanupContext["refereeId"] = id
					cleanupContext["lastSeen"] = lastSeen
					cleanupContext["timeout"] = timeout.String()
					cleanupContext["inactiveDuration"] = time.Since(lastSeen).String()
					logger.LogInfoWithContext(cleanupContext,
						"Removing inactive referee session %s (inactive for %v)", id, time.Since(lastSeen))
					delete(h.activeSessions, id)
				}
			}
			h.mu.Unlock()
		}
	}()
}

func main() {
	// load env variables first (before any logging initialization)
	_ = godotenv.Load()

	// Initialize logging configuration system BEFORE any other application startup
	// This ensures all subsequent operations use the properly configured logger
	if err := initializeLoggingSystem(); err != nil {
		// Use standard log for fatal errors during logger initialization
		log.Fatalf("Failed to initialize logging system: %v", err)
	}

	// Defer logger cleanup with proper error handling
	defer func() {
		if err := logger.CloseLogger(); err != nil {
			// Use structured logging for cleanup errors
			logger.LogErrorWithContext(
				logger.NewSystemContext("shutdown", "logger"),
				"Error closing logger: %v", err,
			)
		}
	}()

	// Get environment configuration (already validated during logger init)
	env := getValidatedEnvironment()

	// Log application startup with structured logging
	logger.LogInfoWithContext(
		logger.NewSystemContext("startup", "main"),
		"Application starting in %s mode", env,
	)

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

	// load credentials with structured error logging and graceful degradation
	if err := loadAndValidateMeetCredentials(); err != nil {
		// Log error but continue with limited functionality
		logger.LogErrorWithContext(
			logger.NewSystemContext("startup", "credentials"),
			"Failed to load meet credentials, continuing with limited functionality: %v", err,
		)
	}

	// announce application startup
	startupContext := logger.NewSystemContext("startup", "server")
	startupContext["port"] = "8080"
	logger.LogInfoWithContext(startupContext, "Starting RefLights application server")

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

	router := SetupRouter(env)

	// create an HTTP server with timeouts
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
		Handler:      router,
	}

	// Log server startup with full configuration context
	serverContext := logger.NewSystemContext("startup", "http_server")
	serverContext["address"] = addr
	serverContext["host"] = host
	serverContext["port"] = port
	serverContext["environment"] = env
	serverContext["readTimeout"] = "10s"
	serverContext["writeTimeout"] = "10s"
	serverContext["idleTimeout"] = "30s"
	logger.LogInfoWithContext(serverContext, "HTTP server starting on %s", addr)

	if err := server.ListenAndServe(); err != nil {
		// Log server startup failure with structured error context
		errorContext := logger.NewSystemContext("startup", "http_server")
		errorContext["address"] = addr
		errorContext["error"] = err.Error()
		logger.LogErrorWithContext(errorContext, "Failed to start HTTP server: %v", err)
		os.Exit(1)
	}
}

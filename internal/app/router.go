// Package app
// File: internal/app/router.go
package app

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
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
	"strings"
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

	// serve /favicon.ico directly
	router.StaticFile("/favicon.ico", "./static/images/favicon.ico")

	// reduce logs in non-production
	if env != "production" {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		logger.Debug.Println("[SetupRouter] Gin logs have been discarded for non-production mode.")
	}

	// configure session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 1 day
		HttpOnly: true,
		Secure:   true,
	})
	router.Use(sessions.Sessions("mySession", store))

	// set security headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Frame-Options", "ALLOW-FROM https://referee-lights.michaelkingston.com.au")
		c.Next()
	})

	// disable caching for all responses
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Cache-Control", "no-store, must-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")
		c.Next()
	})

	// health endpoint
	router.GET("/health", controllers.Health)

	// log endpoint
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

	// initialise your service layer
	occupancyService := services.NewOccupancyService()

	// build the SudoController
	sudoController := controllers.NewSudoController(occupancyService)
	sudoRoutes := router.Group("/sudo")
	{
		// must be logged in
		sudoRoutes.Use(middleware.AuthRequired)
		// must be superuser
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
	router.GET("/heartbeat", func(c *gin.Context) {
		heartbeat.Handler(c.Writer, c.Request)
	})

	// load templates
	router.SetHTMLTemplate(template.Must(template.ParseGlob("templates/*.html")))

	// ensure "meetName" is set (except for a few routes + /static)
	router.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Next()
			return
		}
		if c.Request.URL.Path == "/meets" || c.Request.URL.Path == "/login" {
			c.Next()
			return
		}
		session := sessions.Default(c)
		if _, ok := session.Get("meetName").(string); !ok {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}
		c.Next()
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

	// webSocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// serve static files
	router.Static("/static", "./static")

	// confirm templates path
	_, b, _, _ := runtime.Caller(0)
	basePath := filepath.Dir(b)
	templatesDir := filepath.Join(basePath, "../../templates")

	if _, err := os.Stat(templatesDir); os.IsNotExist(err) {
		log.Fatalf("[SetupRouter] Templates directory does not exist: %s", templatesDir)
	}

	router.SetHTMLTemplate(template.Must(
		template.ParseGlob(filepath.Join(templatesDir, "*.html"))))

	logger.Debug.Printf("[SetupRouter] Templates Path: %s", templatesDir)

	return router
}

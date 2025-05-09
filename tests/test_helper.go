// tests/test_helper.go
package test

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	"go-ref-lights/controllers"
	"go-ref-lights/models"
	"go-ref-lights/services"
	"go-ref-lights/websocket"
)

// SetupTestRouter creates and configures a Gin router for testing.
// This is a minimal version of the SetupRouter function in main.go.
func SetupTestRouter(env string) *gin.Engine {
	// Configure Gin mode
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.TestMode)
	}

	// Setup test credentials
	setupTestCredentials()

	router := gin.Default()

	// Add error handling middleware
	router.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC RECOVERED in %s: %v\n", c.Request.URL.Path, r)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Internal error: %v", r),
				})
				c.Abort()
			}
		}()
		c.Next()
	})

	// Serve static files
	router.Static("/static", "./static")

	// Configure session store
	store := cookie.NewStore([]byte("secret"))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   5400,
		HttpOnly: true,
	})
	router.Use(sessions.Sessions("mySession", store))

	// Create services and controllers
	occupancyService := services.NewOccupancyService()
	positionController := controllers.NewPositionController(occupancyService)
	meetDirectorController := controllers.NewAdminController(occupancyService, positionController)

	// Set application URLs for controllers
	controllers.SetConfig("http://localhost:8080", "ws://localhost:8080/referee-updates")

	// Add simplified root route for testing
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "choose-meet.html", gin.H{
			"Title": "Choose Meet",
		})
	})

	// Public routes
	router.POST("/set-meet", controllers.SetMeetHandler)
	router.GET("/login", controllers.PerformLogin)
	router.POST("/login", controllers.LoginHandler)

	// Special test route for logging in without password verification
	router.POST("/test-login", func(c *gin.Context) {
		meetName := c.PostForm("meetName")
		username := c.PostForm("username")

		if meetName == "" {
			session := sessions.Default(c)
			meetNameFromSession, exists := session.Get("meetName").(string)
			if exists {
				meetName = meetNameFromSession
			}
		}

		if meetName == "" || username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "meetName and username are required"})
			return
		}

		// set up the session without password verification
		session := sessions.Default(c)
		session.Set("meetName", meetName)
		session.Set("username", username)
		session.Set("authenticated", true)
		session.Set("loggedIn", true)
		session.Set("isAdmin", true)

		if err := session.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
			return
		}

		fmt.Printf("Test login successful for user %s at meet %s\n", username, meetName)
		c.Redirect(http.StatusFound, "/index")
	})

	router.GET("/referee/:meetName/:position", func(c *gin.Context) { controllers.RefereeHandler(c, occupancyService) })
	router.GET("/logged-out", func(c *gin.Context) {
		c.HTML(http.StatusOK, "logged-out.html", gin.H{"Title": "You are now logged out"})
	})

	// WebSocket route
	router.GET("/referee-updates", func(c *gin.Context) {
		websocket.ServeWs(c.Writer, c.Request)
	})

	// Protected routes
	protected := router.Group("/")
	protected.Use(func(c *gin.Context) {
		session := sessions.Default(c)
		meetName, hasMeetName := session.Get("meetName").(string)
		authenticated, hasAuth := session.Get("authenticated").(bool)

		if !hasMeetName || !hasAuth || !authenticated {
			c.Redirect(http.StatusFound, "/")
			c.Abort()
			return
		}

		// Add meetName to every template render
		c.Set("meetName", meetName)
		c.Next()
	})
	{
		protected.GET("/index", controllers.Index)
		protected.GET("/occupancy", positionController.GetOccupancyAPI)
		protected.POST("/position/vacate", positionController.VacatePosition)
		protected.POST("/force-vacate", meetDirectorController.ForceVacate)
		protected.POST("/reset-instance", meetDirectorController.ResetInstance)
		protected.GET("/logout", func(c *gin.Context) { controllers.Logout(c, occupancyService) })
	}

	// Create a minimal templates for testing
	tmpl := `
	{{define "choose-meet.html"}}<!DOCTYPE html><html><body>
	<form method="POST" action="/set-meet">
		<input type="text" name="meetName" />
		<button type="submit">Set Meet</button>
	</form>
	</body></html>{{end}}

	{{define "login.html"}}<!DOCTYPE html><html><body>
	<form method="POST" action="/login">
		<input type="text" name="username" />
		<input type="password" name="password" />
		<button type="submit">Login</button>
	</form>
	</body></html>{{end}}

	{{define "index.html"}}<!DOCTYPE html><html><body>
	<div class="qr-code-item">QR Code 1</div>
	<div class="qr-code-item">QR Code 2</div>
	<div class="qr-code-item">QR Code 3</div>
	</body></html>{{end}}

	{{define "referee.html"}}<!DOCTYPE html><html><body>
	<link rel="stylesheet" href="/static/css/style.css" />
	<script src="/static/js/script.js"></script>
	Referee Page
	</body></html>{{end}}

	{{define "logged-out.html"}}<!DOCTYPE html><html><body>
	You are now logged out
	</body></html>{{end}}
	`

	t, err := template.New("").Parse(tmpl)
	if err != nil {
		fmt.Printf("ERROR parsing templates: %v\n", err)
	} else {
		router.SetHTMLTemplate(t)
	}

	return router
}

// setupTestCredentials initializes test credentials for meets
func setupTestCredentials() {
	// Set the path to the credentials file since we're running from the tests directory
	if err := os.Setenv("MEET_CREDS_PATH", "../config/meet_creds.json"); err != nil {
		fmt.Printf("ERROR setting environment variable: %v\n", err)
	}

	// Load credentials from config files
	creds, err := services.LoadMeetCredentials()
	if err != nil {
		fmt.Printf("ERROR loading meet credentials: %v\n", err)
		fmt.Println("Falling back to hardcoded test credentials")

		// Fallback to hardcoded credentials for test_mule
		creds = &models.MeetCreds{
			Meets: []models.Meet{
				{
					Name: "test_mule",
					Date: "2025-04-28",
					Admin: models.Admin{
						Username: "test",
						Password: "test", // Note: In real config this would be hashed
						IsAdmin:  true,
					},
					Logo: "/static/images/APL_logo_white.png",
				},
			},
		}
	}

	services.SetGlobalMeetCredentials(creds)
	fmt.Println("Loaded the following meets for testing:")
	for _, meet := range creds.Meets {
		fmt.Printf("- %s (user: %s)\n", meet.Name, meet.Admin.Username)
	}
}

// main.go
package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go-ref-lights/controllers"
	"go-ref-lights/middleware"
	"go-ref-lights/websocket"
	"log"
	"path/filepath"
	"runtime"
)

func main() {
	// Initialize viper
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error reading config file: %w", err))
	}

	// Now you can use viper.GetString("key") to get config values
	applicationURL := viper.GetString("application_url")
	websocketURL := viper.GetString("websocket_url")

	// Pass these values to controllers or wherever needed
	controllers.SetConfig(applicationURL, websocketURL)

	router := gin.Default()

	// Initialize session store
	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("mysession", store))

	// Determine the absolute path to the templates directory
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	templatesDir := filepath.Join(basepath, "templates", "*.html")

	// load HTML templates
	fmt.Println("Templates Path:", templatesDir)
	router.LoadHTMLGlob(templatesDir)

	// static files
	router.Static("/static", "./static")

	// public routes
	router.GET("/login", controllers.ShowLoginPage)
	router.POST("/login", controllers.PerformLogin)

	// protected routes
	protected := router.Group("/", middleware.AuthRequired)
	{
		protected.GET("/", controllers.Index)
		protected.GET("/left", controllers.Left)
		protected.GET("/centre", controllers.Centre)
		protected.GET("/right", controllers.Right)
		protected.GET("/lights", controllers.Lights)
		protected.GET("/qrcode", controllers.GetQRCode)
		protected.GET("/referee-updates", controllers.RefereeUpdates)
	}

	// Start the WebSocket handler
	go websocket.HandleMessages()

	// Start the server
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

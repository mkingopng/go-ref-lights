// cmd/referee-lights/main.go
package main

import (
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"time"

	"go-ref-lights/controllers"
	"go-ref-lights/heartbeat"
	"go-ref-lights/internal/app"
	"go-ref-lights/logger"
	"go-ref-lights/websocket"
)

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
	router := app.SetupRouter(env)

	// Start background routines
	hbManager := heartbeat.NewHeartbeatManager()
	go hbManager.CleanupInactiveSessions(30 * time.Second)
	go websocket.HandleMessages()

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

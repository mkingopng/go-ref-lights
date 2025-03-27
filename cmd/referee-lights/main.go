// Package main
// File: cmd/referee-lights/main.go
package main

import (
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/joho/godotenv"
	"go-ref-lights/internal/app"
	"log"
	"net/http"
	"os"
	"time"

	"go-ref-lights/controllers"
	"go-ref-lights/heartbeat"
	"go-ref-lights/logger"
	"go-ref-lights/websocket"
)

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
	creds, err := controllers.LoadMeetCreds()
	if err != nil {
		logger.Error.Printf("[main] Error loading credentials: %v", err)
	} else {
		logger.Info.Printf("[main] Loaded meets: %+v", creds.Meets)
	}

	// announce start
	logger.Info.Println("[main] Starting application on port :8080")

	// setup the router
	router := app.SetupRouter(env)

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
	hbManager := heartbeat.NewHeartbeatManager()
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

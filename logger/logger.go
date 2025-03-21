// Package logger provides centralized logging for the application.
// File: logger/logger.go
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ------------------- global loggers -------------------

// four logger levels accessible throughout the application
var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Debug *log.Logger
)

// ------------------- logger initialization -------------------

// InitLogger creates or reinitializes the logging system. It:
// - Ensures `./logs` exists.
// - Creates a timestamped log file in `logs/`.
// - Writes logs to both the file and stdout by default.
// - Configures separate loggers (Info, Warn, Error, Debug) with consistent prefixes & flags.
func InitLogger() error {
	// ensure logs directory exists
	if err := os.MkdirAll("./logs", 0700); err != nil {
		return err
	}

	// create a timestamped log file
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec
	if err != nil {
		return err
	}

	// write logs to both stdout and the file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// configure each logger
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

// SetLogLevel adjusts the Debug logger’s output depending on environment.
// For example, in production you might want to disable Debug logs
// by discarding them entirely. In staging or development, you keep them.
func SetLogLevel(env string) {
	if env == "production" {
		// Discard all debug output in production:
		Debug.SetOutput(io.Discard)
	} else {
		// by default, keep debug logs on:
		// (No-op here, because it’s already set to multiWriter in InitLogger.)
	}
}

// init is called automatically at package load time. It attempts to initialize
// the logger. If initialization fails, we log a fatal error via the standard
// library logger (because our custom ones wouldn’t be ready).
func init() {
	if err := InitLogger(); err != nil {
		log.Fatalf("Failed to initialise custom logger: %v", err)
	}
}

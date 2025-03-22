// Package logger provides centralized logging for the application.
// It exposes four loggers (Info, Warn, Error, Debug) and supports switching debug
// logging on or off via SetLogLevel. By default, logs are written to both stdout
// and a timestamped file in `./logs`.
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// ------------------- Global Loggers -------------------
//
// These four loggers represent different verbosity levels.
// They will be initialized by InitLogger() at package load time.
var (
	// Info is used for high-level events that occur under normal conditions,
	// such as successful startup, routine status messages, or user actions.
	Info *log.Logger

	// Warn is for non-critical issues that may indicate potential problems,
	// e.g., missing environment variables or suspicious requests that still succeed.
	Warn *log.Logger

	// Error is for critical failures that require attention. E.g., inability
	// to read a config file, a database connection drop, etc.
	Error *log.Logger

	// Debug is for low-level diagnostics. Typically disabled in production
	// to avoid performance overhead and log bloat.
	Debug *log.Logger
)

// ------------------- Logger Initialization -------------------

// InitLogger creates or reinitializes the logging system. It:
//
//   - Ensures `./logs` exists.
//   - Creates a timestamped log file in `logs/` named using the format `YYYY-MM-DD_HH-MM-SS.log`.
//   - Writes logs to both the newly created file and stdout by default.
//   - Configures four separate loggers (Info, Warn, Error, Debug) with consistent prefixes & flags.
//
// By default, Debug logs will also appear. You can disable them by calling
// SetLogLevel("production") or a similar environment-based choice.
func InitLogger() error {
	// Ensure logs directory exists
	if err := os.MkdirAll("./logs", 0700); err != nil {
		return err
	}

	// Create a timestamped log file
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec
	if err != nil {
		return err
	}

	// Write logs to both stdout and the file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Configure each logger
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

// ------------------- Log Level Control -------------------

// SetLogLevel allows you to adjust the Debug logger’s behavior based on the
// environment or any other runtime condition.
//
// Typical usage patterns:
//
//	// Production environment: discard debug logs entirely
//	SetLogLevel("production")
//
//	// Development or staging environment: keep debug logs
//	SetLogLevel("development")
//
// If you want more fine-grained control, you can add additional conditions here
// (for example, different levels for staging vs. QA vs. dev), or call
// Debug.SetOutput(...) directly to redirect logs as needed.
func SetLogLevel(env string) {
	if env == "production" {
		// Discard all debug output in production:
		Debug.SetOutput(io.Discard)
	} else {
		// By default, keep debug logs on. There's nothing more to do here,
		// because the debug logger has already been set to multiWriter in InitLogger().
		// This is a no-op for non-production environments.
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

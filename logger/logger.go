package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Global loggers
var (
	Info    *log.Logger
	Warn    *log.Logger
	Error   *log.Logger
	Debug   *log.Logger
	logFile *os.File // logFile is the file handle for our logs, so we can close it later.
)

// InitLogger creates or re-initializes the loggers.
//
// If ENV=test, it only logs to stdout/stderr (no file).
// Otherwise, it creates ./logs if needed, opens a new timestamped file, and logs to both stdout and that file.
func InitLogger() error {
	env := os.Getenv("ENV")

	// If test mode, skip file logging
	if env == "test" {
		Info = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		Warn = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
		Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		Debug = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		return nil
	}

	// Otherwise, create the logs directory and open a file
	// #nosec G301
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return err
	}

	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	var err error
	// #nosec G304
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create each logger that writes to multiWriter
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// CloseLogger closes the open log file, if any
func CloseLogger() error {
	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		return err
	}
	return nil
}

// SetLogLevel modifies the Debug logger’s output. Typically called after InitLogger, e.g.:
//
//	logger.SetLogLevel(os.Getenv("ENV"))
//func SetLogLevel(env string) {
//	if env == "production" {
// Discard Debug logs in production
// Debug.SetOutput(io.Discard)  // fix_me
//}
// Otherwise, do nothing – Debug remains on for dev/test.
//}

func init() {
	// If we’re in a test build, or if no ENV is set, default to “test”
	if os.Getenv("ENV") == "" {
		err := os.Setenv("ENV", "test")
		if err != nil {
			return
		}
	}
	if err := InitLogger(); err != nil {
		// If we can’t init the logger, panic so the tests crash fast
		log.Fatalf("Failed to initialize logger in test init(): %v", err)
	}
}

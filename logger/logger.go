// logger/logger.go
package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	// Info Global loggers accessible throughout your application.
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
	Debug *log.Logger
)

// InitLogger sets up the logging system.
// It creates a "logs" directory (if not present) and opens a new log file with a timestamp.
// All logs are written both to the console and to the log file.
func InitLogger() error {
	// Ensure the logs directory exists.
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}

	// Create a log file name based on the current timestamp.
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Use io.MultiWriter to write simultaneously to stdout and the file.
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Initialize our loggers with prefixes and flags.
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

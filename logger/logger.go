// Package logger logger/logger.go
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
func InitLogger() error {
	if err := os.MkdirAll("logs", 0755); err != nil {
		return err
	}
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	multiWriter := io.MultiWriter(os.Stdout, file)
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

// Ensure the logger is initialised as soon as the package is imported.
func init() {
	if err := InitLogger(); err != nil {
		// Use the standard logger here since our custom one isn't set up.
		log.Fatalf("Failed to initialise custom logger: %v", err)
	}
}

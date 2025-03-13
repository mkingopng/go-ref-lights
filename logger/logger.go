// Package logger provides centralised logging functionality for the application.
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

// Predefined loggers accessible throughout the application.
var (
	Info  *log.Logger // Logs informational messages
	Warn  *log.Logger // Logs warnings
	Error *log.Logger // Logs errors
	Debug *log.Logger // Logs debug messages
)

// ------------------- logger initialisation -------------------

// InitLogger initializes the logging system and creates a log file.
// It ensures that:
// - A `logs/` directory exists.
// - Log messages are written both to a file and stdout.
// - Log files are named using the timestamp format `YYYY-MM-DD_HH-MM-SS.log`.
// - Each logger (Info, Warn, Error, Debug) is configured with a standardised format.
// Returns an error if the log directory or file cannot be created.
func InitLogger() error {
	// ensure the logs directory exists
	if err := os.MkdirAll("./logs", 0700); err != nil {
		return err
	}

	// generate a timestamped log file name
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) // #nosec G304
	if err != nil {
		return err
	}

	// log output will be written to both stdout and the log file
	multiWriter := io.MultiWriter(os.Stdout, file)

	// initialise loggers with appropriate prefixes
	Info = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	return nil
}

// ------------------------- automatic logger setup -------------------------

// init ensures that the logger is initialised when the package is imported.
// if logger initialisation fails, the application will log a fatal error
// using the standard `log` package (since our custom loggers won't be available).
func init() {
	if err := InitLogger(); err != nil {
		// use the standard logger here since our custom one isn't set up.
		log.Fatalf("Failed to initialise custom logger: %v", err)
	}
}

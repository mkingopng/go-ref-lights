package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LogLevel represents logging severity levels
type LogLevel int

const (
	// DebugLevel logs detailed information for debugging
	DebugLevel LogLevel = iota
	// InfoLevel logs general operational messages
	InfoLevel
	// WarnLevel logs potential issues that don't prevent operation
	WarnLevel
	// ErrorLevel logs errors that may impact functionality
	ErrorLevel
)

// LogConfig contains configuration for the logger
type LogConfig struct {
	// CurrentLevel is the minimum severity level to log
	CurrentLevel LogLevel
	// RotationSize is the max log file size in bytes before rotation (default 50MB)
	RotationSize int64
	// MaxLogFiles is how many log files to keep (default 5)
	MaxLogFiles int
	// ExcludePatterns contains text patterns to exclude from logs
	ExcludePatterns []string
	// LogDirectory specifies where logs are stored
	LogDirectory string
	// mutex for thread safety
	mu sync.Mutex
}

// Global loggers
var (
	Info    *log.Logger
	Warn    *log.Logger
	Error   *log.Logger
	Debug   *log.Logger
	logFile *os.File // logFile is the file handle for our logs, so we can close it later.
	config  LogConfig
)

// init initializes the default configuration
func init() {
	// If we're in a test build, or if no ENV is set, default to "test"
	if os.Getenv("ENV") == "" {
		_ = os.Setenv("ENV", "test")
	}

	// Default config
	config = LogConfig{
		CurrentLevel:    InfoLevel,
		RotationSize:    50 * 1024 * 1024, // 50MB
		MaxLogFiles:     5,
		ExcludePatterns: []string{"health check", "heartbeat"},
		LogDirectory:    "logs",
	}

	// Load environment-specific settings
	applyEnvConfig()

	if err := InitLogger(); err != nil {
		// If we can't init the logger, panic so the app crashes fast
		log.Fatalf("Failed to initialize logger: %v", err)
	}
}

// applyEnvConfig loads environment-specific configuration
func applyEnvConfig() {
	env := os.Getenv("ENV")

	// In production, exclude more routine messages
	if env == "production" {
		config.CurrentLevel = InfoLevel
		config.ExcludePatterns = append(config.ExcludePatterns,
			"ping", "readPump normal close", "writePump normal close")
	} else if env == "development" {
		config.CurrentLevel = DebugLevel
		config.RotationSize = 20 * 1024 * 1024 // 20MB
	}

	// Allow override from environment variables
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		switch strings.ToLower(logLevel) {
		case "debug":
			config.CurrentLevel = DebugLevel
		case "info":
			config.CurrentLevel = InfoLevel
		case "warn":
			config.CurrentLevel = WarnLevel
		case "error":
			config.CurrentLevel = ErrorLevel
		}
	}
}

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
	if err := os.MkdirAll(config.LogDirectory, 0o755); err != nil {
		return err
	}

	// Clean up old log files if we're over the limit
	if config.MaxLogFiles > 0 {
		cleanupOldLogs()
	}

	logFileName := filepath.Join(config.LogDirectory, fmt.Sprintf("app-%s.log", time.Now().Format("2006-01-02_15-04-05")))
	var err error
	// #nosec G304
	logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Create wrapper writers that filter unwanted logs
	debugWriter := newFilteredWriter(multiWriter, DebugLevel)
	infoWriter := newFilteredWriter(multiWriter, InfoLevel)
	warnWriter := newFilteredWriter(multiWriter, WarnLevel)
	errorWriter := newFilteredWriter(multiWriter, ErrorLevel)

	// Create each logger that writes to its filtered writer
	Info = log.New(infoWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(warnWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(debugWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

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

// SetLogLevel updates the minimum log level at runtime
func SetLogLevel(level LogLevel) {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.CurrentLevel = level
}

// AddExcludePattern adds a text pattern to filter out of logs
func AddExcludePattern(pattern string) {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.ExcludePatterns = append(config.ExcludePatterns, pattern)
}

// cleanupOldLogs removes old log files if we have more than the configured maximum
func cleanupOldLogs() {
	files, err := filepath.Glob(filepath.Join(config.LogDirectory, "app-*.log"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding log files: %v\n", err)
		return
	}

	if len(files) <= config.MaxLogFiles {
		return
	}

	// Get file info for sorting by modification time
	type fileInfo struct {
		path string
		time time.Time
	}

	fileInfos := make([]fileInfo, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{file, info.ModTime()})
	}

	// Sort files by modification time (oldest first)
	for i := 0; i < len(fileInfos)-1; i++ {
		for j := i + 1; j < len(fileInfos); j++ {
			if fileInfos[j].time.Before(fileInfos[i].time) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
			}
		}
	}

	// Delete oldest files that exceed the limit
	for i := 0; i < len(fileInfos)-config.MaxLogFiles; i++ {
		if err := os.Remove(fileInfos[i].path); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing old log file %s: %v\n", fileInfos[i].path, err)
		}
	}
}

// filteredWriter is an io.Writer that filters logs based on level and patterns
type filteredWriter struct {
	out     io.Writer
	level   LogLevel
	exclude []string
}

// newFilteredWriter creates a new writer that filters based on log level and patterns
func newFilteredWriter(out io.Writer, level LogLevel) io.Writer {
	return &filteredWriter{
		out:     out,
		level:   level,
		exclude: config.ExcludePatterns,
	}
}

// Write implements io.Writer interface and filters messages
func (fw *filteredWriter) Write(p []byte) (n int, err error) {
	// Return full length to satisfy Writer interface, but actually write nothing
	// if this message should be filtered out

	config.mu.Lock()
	currentLevel := config.CurrentLevel
	excludePatterns := make([]string, len(config.ExcludePatterns))
	copy(excludePatterns, config.ExcludePatterns)
	config.mu.Unlock()

	// Skip if below current minimum level
	if fw.level < currentLevel {
		return len(p), nil
	}

	// Check for excluded patterns
	for _, pattern := range excludePatterns {
		if strings.Contains(string(p), pattern) {
			return len(p), nil
		}
	}

	// Message passes filters, write it
	return fw.out.Write(p)
}

// LogPerformanceMetric logs performance data with appropriate filtering
func LogPerformanceMetric(operation string, duration time.Duration) {
	// Always log slow operations as warnings
	if duration > 500*time.Millisecond {
		Warn.Printf("[PERF] Slow operation: %s took %v", operation, duration)
		return
	}

	// Log normal operations at debug level
	Debug.Printf("[PERF] Operation: %s took %v", operation, duration)
}

// LogExceptional logs important exception-type information that should always be logged
func LogExceptional(format string, args ...interface{}) {
	// This bypasses normal filtering since we always want to see these
	msg := fmt.Sprintf(format, args...)
	formattedMsg := fmt.Sprintf("EXCEPTION: %s\n", msg)

	// Write directly to both outputs
	if logFile != nil {
		_, _ = fmt.Fprint(logFile, formattedMsg)
	}
	_, _ = fmt.Fprint(os.Stdout, formattedMsg)
}

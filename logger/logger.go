package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// LogEntry represents a structured log entry with context
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Source    string                 `json:"source"`
	MeetName  string                 `json:"meetName,omitempty"`
	RefereeID string                 `json:"refereeId,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// ContextCategory represents different types of operations for logging context
type ContextCategory string

const (
	WebSocketContext      ContextCategory = "websocket"
	TimerContext          ContextCategory = "timer"
	AuthenticationContext ContextCategory = "authentication"
	PositionContext       ContextCategory = "position"
	HTTPContext           ContextCategory = "http"
	SystemContext         ContextCategory = "system"
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger represents the enhanced logger with configurable levels
type Logger struct {
	level   LogLevel
	env     string
	mu      sync.RWMutex
	logFile *os.File

	// Individual loggers for each level
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger

	// Performance monitoring
	logCount    int64 // Atomic counter for total logs written
	bytesLogged int64 // Atomic counter for bytes written
	startTime   time.Time

	// File size monitoring
	maxFileSize int64 // Maximum file size in bytes (default 10MB/hour)

	// Lazy evaluation support
	lazyEvalEnabled bool

	// Advanced performance monitoring
	rotationCount   int64     // Number of log rotations performed
	lastRotation    time.Time // Time of last rotation
	rotationEnabled bool      // Whether automatic rotation is enabled

	// Performance thresholds
	maxLogsPerSecond  float64 // Maximum logs per second threshold
	maxBytesPerSecond float64 // Maximum bytes per second threshold
	performanceAlerts bool    // Whether to log performance alerts
}

// Global logger instance and legacy loggers for backward compatibility
var (
	globalLogger *Logger
	Info         *log.Logger
	Warn         *log.Logger
	Error        *log.Logger
	Debug        *log.Logger
	logFile      *os.File // logFile is the file handle for our logs, so we can close it later.
)

// NewLogger creates a new Logger instance with environment-based configuration
func NewLogger() *Logger {
	env := os.Getenv("ENV")
	if env == "" {
		env = "production" // Default to production for safety
	}

	logger := &Logger{
		env:               env,
		startTime:         time.Now(),
		maxFileSize:       10 * 1024 * 1024, // 10MB default
		lazyEvalEnabled:   true,
		rotationEnabled:   true,
		maxLogsPerSecond:  1000,        // Default threshold
		maxBytesPerSecond: 1024 * 1024, // 1MB/sec default threshold
		performanceAlerts: false,       // Disabled by default to avoid log spam
	}

	// Set log level based on environment
	logger.setLevelFromEnvironment()

	return logger
}

// setLevelFromEnvironment sets the log level based on ENV and LOG_LEVEL environment variables
func (l *Logger) setLevelFromEnvironment() {
	// Check for explicit LOG_LEVEL override first
	if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
		if level := parseLogLevel(logLevelStr); level != -1 {
			l.level = LogLevel(level)
			return
		}
	}

	// Set level based on environment
	switch strings.ToLower(l.env) {
	case "production":
		l.level = WARN // Production: ERROR, WARN, and critical INFO only
	case "development", "dev":
		l.level = DEBUG // Development: All levels including DEBUG
	case "test":
		l.level = WARN // Test: ERROR and WARN only
	default:
		l.level = WARN // Default to production-safe level
	}
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(levelStr string) int {
	switch strings.ToUpper(strings.TrimSpace(levelStr)) {
	case "DEBUG":
		return int(DEBUG)
	case "INFO":
		return int(INFO)
	case "WARN", "WARNING":
		return int(WARN)
	case "ERROR":
		return int(ERROR)
	default:
		return -1 // Invalid level
	}
}

// SetLevel sets the logging level at runtime
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// ShouldLog returns true if a message at the given level should be logged
func (l *Logger) ShouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// LazyLogFunc represents a function that generates log content lazily
type LazyLogFunc func() (message string, context map[string]interface{})

// LogLazy logs a message using lazy evaluation to avoid expensive operations when logging is disabled
func (l *Logger) LogLazy(level LogLevel, lazyFunc LazyLogFunc) {
	if !l.ShouldLog(level) {
		return // Early return without calling expensive lazyFunc
	}

	message, context := lazyFunc()
	l.LogWithContext(level, context, message)
}

// ConditionalLogFunc represents a function that performs conditional logging with expensive operations
type ConditionalLogFunc func() (level LogLevel, context map[string]interface{}, message string, args []interface{})

// LogConditional performs conditional logging with pre-check optimization
func (l *Logger) LogConditional(checkLevel LogLevel, logFunc ConditionalLogFunc) {
	if !l.ShouldLog(checkLevel) {
		return // Early return without calling expensive logFunc
	}

	level, context, message, args := logFunc()
	if l.ShouldLog(level) {
		l.LogWithContext(level, context, message, args...)
	}
}

// LogIfEnabled logs only if the specified level is enabled, avoiding expensive operations
func (l *Logger) LogIfEnabled(level LogLevel, expensiveContextFunc func() map[string]interface{}, message string, args ...interface{}) {
	if !l.ShouldLog(level) {
		return
	}

	var context map[string]interface{}
	if expensiveContextFunc != nil {
		context = expensiveContextFunc()
	}

	l.LogWithContext(level, context, message, args...)
}

// GetPerformanceStats returns current performance statistics
func (l *Logger) GetPerformanceStats() (logCount int64, bytesLogged int64, logsPerSecond float64, bytesPerSecond float64) {
	logCount = atomic.LoadInt64(&l.logCount)
	bytesLogged = atomic.LoadInt64(&l.bytesLogged)

	elapsed := time.Since(l.startTime).Seconds()
	if elapsed > 0 {
		logsPerSecond = float64(logCount) / elapsed
		bytesPerSecond = float64(bytesLogged) / elapsed
	}

	return
}

// GetDetailedPerformanceStats returns comprehensive performance statistics
func (l *Logger) GetDetailedPerformanceStats() map[string]interface{} {
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := l.GetPerformanceStats()
	rotationCount := atomic.LoadInt64(&l.rotationCount)

	l.mu.RLock()
	lastRotation := l.lastRotation
	maxFileSize := l.maxFileSize
	rotationEnabled := l.rotationEnabled
	l.mu.RUnlock()

	currentSize, _, percentUsed := l.GetFileSizeInfo()

	return map[string]interface{}{
		"logCount":        logCount,
		"bytesLogged":     bytesLogged,
		"logsPerSecond":   logsPerSecond,
		"bytesPerSecond":  bytesPerSecond,
		"rotationCount":   rotationCount,
		"lastRotation":    lastRotation,
		"currentFileSize": currentSize,
		"maxFileSize":     maxFileSize,
		"filePercentUsed": percentUsed,
		"rotationEnabled": rotationEnabled,
		"uptime":          time.Since(l.startTime),
		"averageBytesPerLog": func() float64 {
			if logCount > 0 {
				return float64(bytesLogged) / float64(logCount)
			}
			return 0
		}(),
	}
}

// CheckPerformanceThresholds checks if performance thresholds are exceeded
func (l *Logger) CheckPerformanceThresholds() (exceeded bool, alerts []string) {
	_, _, logsPerSecond, bytesPerSecond := l.GetPerformanceStats()

	l.mu.RLock()
	maxLogsPerSecond := l.maxLogsPerSecond
	maxBytesPerSecond := l.maxBytesPerSecond
	performanceAlerts := l.performanceAlerts
	l.mu.RUnlock()

	if !performanceAlerts {
		return false, nil
	}

	if logsPerSecond > maxLogsPerSecond {
		exceeded = true
		alerts = append(alerts, fmt.Sprintf("Logs per second (%.2f) exceeds threshold (%.2f)", logsPerSecond, maxLogsPerSecond))
	}

	if bytesPerSecond > maxBytesPerSecond {
		exceeded = true
		alerts = append(alerts, fmt.Sprintf("Bytes per second (%.2f) exceeds threshold (%.2f)", bytesPerSecond, maxBytesPerSecond))
	}

	return exceeded, alerts
}

// SetPerformanceThresholds sets performance monitoring thresholds
func (l *Logger) SetPerformanceThresholds(maxLogsPerSecond, maxBytesPerSecond float64, enableAlerts bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.maxLogsPerSecond = maxLogsPerSecond
	l.maxBytesPerSecond = maxBytesPerSecond
	l.performanceAlerts = enableAlerts
}

// EnableRotation enables or disables automatic log rotation
func (l *Logger) EnableRotation(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotationEnabled = enabled
}

// GetFileSizeInfo returns current log file size information
func (l *Logger) GetFileSizeInfo() (currentSize int64, maxSize int64, percentUsed float64) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	maxSize = l.maxFileSize

	if l.logFile != nil {
		if stat, err := l.logFile.Stat(); err == nil {
			currentSize = stat.Size()
			if maxSize > 0 {
				percentUsed = float64(currentSize) / float64(maxSize) * 100
			}
		}
	}

	return
}

// SetMaxFileSize sets the maximum file size for monitoring
func (l *Logger) SetMaxFileSize(maxSize int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxFileSize = maxSize
}

// IsFileSizeExceeded checks if the log file has exceeded the maximum size
func (l *Logger) IsFileSizeExceeded() bool {
	currentSize, maxSize, _ := l.GetFileSizeInfo()
	return maxSize > 0 && currentSize > maxSize
}

// RotateLogFile rotates the current log file if it exceeds the maximum size
func (l *Logger) RotateLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile == nil {
		return nil // No file to rotate
	}

	// Get current file info for rotation tracking
	oldStat, _ := l.logFile.Stat()

	// Close current file
	if err := l.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Create new log file with timestamp (including microseconds for uniqueness)
	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05.000000")+".log")
	var err error
	// #nosec G304
	l.logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// Update rotation tracking
	atomic.AddInt64(&l.rotationCount, 1)
	l.lastRotation = time.Now()

	// Update global logFile reference
	logFile = l.logFile

	// Recreate loggers with new file
	multiWriter := io.MultiWriter(os.Stdout, l.logFile)
	l.infoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.warnLogger = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.errorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.debugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Store current level for legacy logger setup (avoid calling ShouldLog while locked)
	currentLevel := l.level

	// Release lock before calling methods that might need it
	l.mu.Unlock()

	// Update legacy loggers without holding the lock
	l.setupLegacyLoggersWithLevel(currentLevel)

	// Log rotation event (to new file)
	if oldStat != nil {
		l.LogWithContext(INFO, NewSystemContext("log_rotation", "logger"),
			"Log file rotated: %s (size: %d bytes)", oldStat.Name(), oldStat.Size())
	}

	// Re-acquire lock for defer unlock
	l.mu.Lock()

	return nil
}

// ForceRotation forces log rotation regardless of file size
func (l *Logger) ForceRotation() error {
	return l.RotateLogFile()
}

// GetRotationStats returns log rotation statistics
func (l *Logger) GetRotationStats() (rotationCount int64, lastRotation time.Time, timeSinceLastRotation time.Duration) {
	rotationCount = atomic.LoadInt64(&l.rotationCount)

	l.mu.RLock()
	lastRotation = l.lastRotation
	l.mu.RUnlock()

	if !lastRotation.IsZero() {
		timeSinceLastRotation = time.Since(lastRotation)
	}

	return
}

// LogWithContext logs a message with structured context information
func (l *Logger) LogWithContext(level LogLevel, context map[string]interface{}, message string, args ...interface{}) {
	if !l.ShouldLog(level) {
		return
	}

	// Get caller information for source (optimized to avoid expensive calls when not needed)
	_, file, line, ok := runtime.Caller(2)
	source := "unknown"
	if ok {
		source = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Create structured log entry with lazy message formatting
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level.String(),
		Message:   l.formatMessage(message, args...),
		Context:   context,
		Source:    source,
	}

	// Extract common fields from context if present (optimized with type assertions)
	if context != nil {
		if meetName, ok := context["meetName"].(string); ok {
			entry.MeetName = meetName
		}
		if refereeID, ok := context["refereeId"].(string); ok {
			entry.RefereeID = refereeID
		}
		if err, ok := context["error"].(string); ok {
			entry.Error = err
		} else if err, ok := context["error"].(error); ok {
			entry.Error = err.Error()
		}
	}

	// Format and log the entry
	l.logStructuredEntry(level, entry)
}

// formatMessage provides optimized message formatting with conditional processing
func (l *Logger) formatMessage(message string, args ...interface{}) string {
	if len(args) == 0 {
		return message
	}

	// Only format if there are arguments to avoid unnecessary sprintf calls
	return fmt.Sprintf(message, args...)
}

// logStructuredEntry formats and outputs a structured log entry
func (l *Logger) logStructuredEntry(level LogLevel, entry LogEntry) {
	// Create JSON representation for structured logging
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		l.logSimple(level, "Failed to marshal log entry: %v | Original message: %s", err, entry.Message)
		return
	}

	// Update performance counters
	atomic.AddInt64(&l.logCount, 1)
	atomic.AddInt64(&l.bytesLogged, int64(len(jsonBytes)))

	// Log the JSON entry using the appropriate logger
	switch level {
	case DEBUG:
		if l.debugLogger != nil {
			l.debugLogger.Print(string(jsonBytes))
		}
	case INFO:
		if l.infoLogger != nil {
			l.infoLogger.Print(string(jsonBytes))
		}
	case WARN:
		if l.warnLogger != nil {
			l.warnLogger.Print(string(jsonBytes))
		}
	case ERROR:
		if l.errorLogger != nil {
			l.errorLogger.Print(string(jsonBytes))
		}
	}

	// Check for file size rotation if enabled
	l.mu.RLock()
	rotationEnabled := l.rotationEnabled
	l.mu.RUnlock()

	if rotationEnabled && l.maxFileSize > 0 && l.IsFileSizeExceeded() {
		// Rotate in a separate goroutine to avoid blocking logging
		go func() {
			if err := l.RotateLogFile(); err != nil {
				// Log rotation error to stderr as fallback
				fmt.Fprintf(os.Stderr, "Log rotation failed: %v\n", err)
			}
		}()
	}

	// Check performance thresholds if enabled
	if exceeded, alerts := l.CheckPerformanceThresholds(); exceeded {
		// Log performance alerts to stderr to avoid recursive logging
		for _, alert := range alerts {
			fmt.Fprintf(os.Stderr, "PERFORMANCE ALERT: %s\n", alert)
		}
	}
}

// logSimple provides fallback logging without structured format
func (l *Logger) logSimple(level LogLevel, message string, args ...interface{}) {
	if !l.ShouldLog(level) {
		return
	}

	switch level {
	case DEBUG:
		if l.debugLogger != nil {
			l.debugLogger.Printf(message, args...)
		}
	case INFO:
		if l.infoLogger != nil {
			l.infoLogger.Printf(message, args...)
		}
	case WARN:
		if l.warnLogger != nil {
			l.warnLogger.Printf(message, args...)
		}
	case ERROR:
		if l.errorLogger != nil {
			l.errorLogger.Printf(message, args...)
		}
	}
}

// InitLogger creates or re-initializes the loggers.
//
// If ENV=test, it only logs to stdout/stderr (no file).
// Otherwise, it creates ./logs if needed, opens a new timestamped file, and logs to both stdout and that file.
func InitLogger() error {
	// Initialize global logger instance
	globalLogger = NewLogger()

	env := os.Getenv("ENV")

	// If test mode, skip file logging
	if env == "test" {
		return globalLogger.initTestMode()
	}

	// Otherwise, create the logs directory and open a file
	return globalLogger.initFileMode()
}

// initTestMode initializes logging for test environment (stdout/stderr only)
func (l *Logger) initTestMode() error {
	l.infoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.warnLogger = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.debugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Set up legacy loggers for backward compatibility
	l.setupLegacyLoggers()

	return nil
}

// initFileMode initializes logging for production/development with file output
func (l *Logger) initFileMode() error {
	// Create the logs directory
	// #nosec G301
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return err
	}

	logFileName := filepath.Join("logs", time.Now().Format("2006-01-02_15-04-05")+".log")
	var err error
	// #nosec G304
	l.logFile, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}

	// Set global logFile for backward compatibility
	logFile = l.logFile

	multiWriter := io.MultiWriter(os.Stdout, l.logFile)

	// Create each logger that writes to multiWriter
	l.infoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.warnLogger = log.New(multiWriter, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.errorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	l.debugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Set up legacy loggers for backward compatibility
	l.setupLegacyLoggers()

	return nil
}

// setupLegacyLoggers configures the global legacy loggers based on current level
func (l *Logger) setupLegacyLoggers() {
	l.mu.RLock()
	currentLevel := l.level
	l.mu.RUnlock()
	l.setupLegacyLoggersWithLevel(currentLevel)
}

// setupLegacyLoggersWithLevel configures the global legacy loggers with a specific level (no locking)
func (l *Logger) setupLegacyLoggersWithLevel(level LogLevel) {
	// Always set up the loggers, but conditionally discard output based on level
	if INFO >= level {
		Info = l.infoLogger
	} else {
		Info = log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	if WARN >= level {
		Warn = l.warnLogger
	} else {
		Warn = log.New(io.Discard, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	if ERROR >= level {
		Error = l.errorLogger
	} else {
		Error = log.New(io.Discard, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	}

	if DEBUG >= level {
		Debug = l.debugLogger
	} else {
		Debug = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

// CloseLogger closes the open log file, if any
func CloseLogger() error {
	if globalLogger != nil && globalLogger.logFile != nil {
		err := globalLogger.logFile.Close()
		globalLogger.logFile = nil
		logFile = nil // Clear legacy reference
		return err
	}
	if logFile != nil {
		err := logFile.Close()
		logFile = nil
		return err
	}
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// SetGlobalLogLevel sets the log level for the global logger
func SetGlobalLogLevel(level LogLevel) {
	if globalLogger != nil {
		globalLogger.SetLevel(level)
		// Update legacy loggers to reflect new level
		globalLogger.setupLegacyLoggers()
	}
}

// ShouldLog checks if the global logger should log at the given level
func ShouldLog(level LogLevel) bool {
	if globalLogger != nil {
		return globalLogger.ShouldLog(level)
	}
	return level >= WARN // Default to production-safe level
}

// SetLogLevel modifies the logger's output based on environment.
// This function is kept for backward compatibility but now uses the new level system.
func SetLogLevel(env string) {
	if globalLogger == nil {
		return
	}

	// Update environment and reconfigure level
	globalLogger.mu.Lock()
	globalLogger.env = env
	globalLogger.mu.Unlock()

	globalLogger.setLevelFromEnvironment()
	globalLogger.setupLegacyLoggers()
}

// Context helper functions for common logging patterns

// NewWebSocketContext creates context for WebSocket operations
func NewWebSocketContext(action, meetName, refereeID, remoteAddr string) map[string]interface{} {
	return map[string]interface{}{
		"component":  string(WebSocketContext),
		"action":     action,
		"meetName":   meetName,
		"refereeId":  refereeID,
		"remoteAddr": remoteAddr,
	}
}

// NewTimerContext creates context for timer operations
func NewTimerContext(action, meetName, timerType string, timerID interface{}) map[string]interface{} {
	return map[string]interface{}{
		"component": string(TimerContext),
		"action":    action,
		"meetName":  meetName,
		"timerType": timerType,
		"timerId":   timerID,
	}
}

// NewAuthenticationContext creates context for authentication operations
func NewAuthenticationContext(action, username, ipAddress string) map[string]interface{} {
	return map[string]interface{}{
		"component": string(AuthenticationContext),
		"action":    action,
		"username":  username,
		"ipAddress": ipAddress,
	}
}

// NewPositionContext creates context for position operations
func NewPositionContext(action, meetName, position, refereeID string) map[string]interface{} {
	return map[string]interface{}{
		"component": string(PositionContext),
		"action":    action,
		"meetName":  meetName,
		"position":  position,
		"refereeId": refereeID,
	}
}

// NewHTTPContext creates context for HTTP operations
func NewHTTPContext(method, path, userAgent, ipAddress string, statusCode int) map[string]interface{} {
	return map[string]interface{}{
		"component":  string(HTTPContext),
		"method":     method,
		"path":       path,
		"userAgent":  userAgent,
		"ipAddress":  ipAddress,
		"statusCode": statusCode,
	}
}

// NewSystemContext creates context for system operations
func NewSystemContext(action, component string) map[string]interface{} {
	return map[string]interface{}{
		"component": string(SystemContext),
		"action":    action,
		"subsystem": component,
	}
}

// AddError adds error information to an existing context
func AddError(context map[string]interface{}, err error) map[string]interface{} {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["error"] = err.Error()
	return context
}

// AddMeetContext adds meet information to an existing context
func AddMeetContext(context map[string]interface{}, meetName string) map[string]interface{} {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["meetName"] = meetName
	return context
}

// AddRefereeContext adds referee information to an existing context
func AddRefereeContext(context map[string]interface{}, refereeID string) map[string]interface{} {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["refereeId"] = refereeID
	return context
}

// Error categorization system for different types of failures

// ErrorCategory represents different types of system failures
type ErrorCategory string

const (
	AuthenticationError ErrorCategory = "authentication"
	AuthorizationError  ErrorCategory = "authorization"
	ValidationError     ErrorCategory = "validation"
	NetworkError        ErrorCategory = "network"
	DatabaseError       ErrorCategory = "database"
	ConfigurationError  ErrorCategory = "configuration"
	BusinessLogicError  ErrorCategory = "business_logic"
	SystemError         ErrorCategory = "system"
	WebSocketError      ErrorCategory = "websocket"
	TimerError          ErrorCategory = "timer"
	PositionError       ErrorCategory = "position"
	SessionError        ErrorCategory = "session"
	MarshalingError     ErrorCategory = "marshaling"
	FileSystemError     ErrorCategory = "filesystem"
)

// ErrorSeverity represents the severity level of errors
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
)

// ErrorContext provides structured error information
type ErrorContext struct {
	Category   ErrorCategory          `json:"category"`
	Severity   ErrorSeverity          `json:"severity"`
	Code       string                 `json:"code,omitempty"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source"`
	StackTrace string                 `json:"stackTrace,omitempty"`
	UserID     string                 `json:"userId,omitempty"`
	SessionID  string                 `json:"sessionId,omitempty"`
	RequestID  string                 `json:"requestId,omitempty"`
	IPAddress  string                 `json:"ipAddress,omitempty"`
	UserAgent  string                 `json:"userAgent,omitempty"`
	MeetName   string                 `json:"meetName,omitempty"`
	RefereeID  string                 `json:"refereeId,omitempty"`
}

// NewErrorContext creates a new structured error context
func NewErrorContext(category ErrorCategory, severity ErrorSeverity, message string) *ErrorContext {
	// Get caller information for source
	_, file, line, ok := runtime.Caller(1)
	source := "unknown"
	if ok {
		source = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	return &ErrorContext{
		Category:  category,
		Severity:  severity,
		Message:   message,
		Details:   make(map[string]interface{}),
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Source:    source,
	}
}

// WithCode adds an error code to the error context
func (ec *ErrorContext) WithCode(code string) *ErrorContext {
	ec.Code = code
	return ec
}

// WithDetail adds a detail field to the error context
func (ec *ErrorContext) WithDetail(key string, value interface{}) *ErrorContext {
	if ec.Details == nil {
		ec.Details = make(map[string]interface{})
	}
	ec.Details[key] = value
	return ec
}

// WithContext adds context information to the error context
func (ec *ErrorContext) WithContext(key string, value interface{}) *ErrorContext {
	if ec.Context == nil {
		ec.Context = make(map[string]interface{})
	}
	ec.Context[key] = value
	return ec
}

// WithUser adds user information to the error context
func (ec *ErrorContext) WithUser(userID, sessionID string) *ErrorContext {
	ec.UserID = userID
	ec.SessionID = sessionID
	return ec
}

// WithRequest adds request information to the error context
func (ec *ErrorContext) WithRequest(requestID, ipAddress, userAgent string) *ErrorContext {
	ec.RequestID = requestID
	ec.IPAddress = ipAddress
	ec.UserAgent = userAgent
	return ec
}

// WithMeet adds meet information to the error context
func (ec *ErrorContext) WithMeet(meetName, refereeID string) *ErrorContext {
	ec.MeetName = meetName
	ec.RefereeID = refereeID
	return ec
}

// WithStackTrace adds stack trace information to the error context
func (ec *ErrorContext) WithStackTrace() *ErrorContext {
	// Capture stack trace (simplified version)
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	ec.StackTrace = string(buf[:n])
	return ec
}

// WithError adds error information from a Go error
func (ec *ErrorContext) WithError(err error) *ErrorContext {
	if err != nil {
		ec.WithDetail("error", err.Error())
	}
	return ec
}

// ToLogContext converts ErrorContext to a log context map
func (ec *ErrorContext) ToLogContext() map[string]interface{} {
	logContext := make(map[string]interface{})

	// Add core error information
	logContext["errorCategory"] = string(ec.Category)
	logContext["errorSeverity"] = string(ec.Severity)
	logContext["errorMessage"] = ec.Message

	if ec.Code != "" {
		logContext["errorCode"] = ec.Code
	}

	// Add user and request information
	if ec.UserID != "" {
		logContext["userId"] = ec.UserID
	}
	if ec.SessionID != "" {
		logContext["sessionId"] = ec.SessionID
	}
	if ec.RequestID != "" {
		logContext["requestId"] = ec.RequestID
	}
	if ec.IPAddress != "" {
		logContext["ipAddress"] = ec.IPAddress
	}
	if ec.UserAgent != "" {
		logContext["userAgent"] = ec.UserAgent
	}
	if ec.MeetName != "" {
		logContext["meetName"] = ec.MeetName
	}
	if ec.RefereeID != "" {
		logContext["refereeId"] = ec.RefereeID
	}
	if ec.StackTrace != "" {
		logContext["stackTrace"] = ec.StackTrace
	}

	// Add details
	if ec.Details != nil {
		for k, v := range ec.Details {
			logContext[k] = v
		}
	}

	// Add context
	if ec.Context != nil {
		for k, v := range ec.Context {
			logContext[k] = v
		}
	}

	return logContext
}

// LogError logs the error context using the global logger
func (ec *ErrorContext) LogError() {
	LogErrorWithContext(ec.ToLogContext(), ec.Message)
}

// LogWarn logs the error context as a warning using the global logger
func (ec *ErrorContext) LogWarn() {
	LogWarnWithContext(ec.ToLogContext(), ec.Message)
}

// Predefined error context creators for common scenarios

// NewAuthenticationErrorContext creates an error context for authentication failures
func NewAuthenticationErrorContext(message, username, ipAddress, userAgent string) *ErrorContext {
	return NewErrorContext(AuthenticationError, SeverityHigh, message).
		WithDetail("username", username).
		WithDetail("usernameLength", len(username)).
		WithRequest("", ipAddress, userAgent)
}

// NewAuthorizationErrorContext creates an error context for authorization failures
func NewAuthorizationErrorContext(message, userID, resource, ipAddress string) *ErrorContext {
	return NewErrorContext(AuthorizationError, SeverityHigh, message).
		WithDetail("resource", resource).
		WithUser(userID, "").
		WithDetail("ipAddress", ipAddress)
}

// NewWebSocketErrorContext creates an error context for WebSocket failures
func NewWebSocketErrorContext(message, meetName, refereeID, remoteAddr string) *ErrorContext {
	return NewErrorContext(WebSocketError, SeverityMedium, message).
		WithMeet(meetName, refereeID).
		WithDetail("remoteAddr", remoteAddr)
}

// NewPositionErrorContext creates an error context for position occupancy failures
func NewPositionErrorContext(message, meetName, position, userID string) *ErrorContext {
	return NewErrorContext(PositionError, SeverityMedium, message).
		WithMeet(meetName, "").
		WithDetail("position", position).
		WithDetail("userId", userID)
}

// NewTimerErrorContext creates an error context for timer operation failures
func NewTimerErrorContext(message, meetName, timerType string, timerID interface{}) *ErrorContext {
	return NewErrorContext(TimerError, SeverityMedium, message).
		WithMeet(meetName, "").
		WithDetail("timerType", timerType).
		WithDetail("timerId", timerID)
}

// NewValidationErrorContext creates an error context for validation failures
func NewValidationErrorContext(message, field string, value interface{}) *ErrorContext {
	return NewErrorContext(ValidationError, SeverityLow, message).
		WithDetail("field", field).
		WithDetail("value", value)
}

// NewSystemErrorContext creates an error context for system-level failures
func NewSystemErrorContext(message, component string) *ErrorContext {
	return NewErrorContext(SystemError, SeverityCritical, message).
		WithDetail("component", component).
		WithStackTrace()
}

// Global convenience functions for structured logging

// LogWithContextGlobal logs with context using the global logger
func LogWithContextGlobal(level LogLevel, context map[string]interface{}, message string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.LogWithContext(level, context, message, args...)
	}
}

// LogErrorWithContext logs an error with context using the global logger
func LogErrorWithContext(context map[string]interface{}, message string, args ...interface{}) {
	LogWithContextGlobal(ERROR, context, message, args...)
}

// LogWarnWithContext logs a warning with context using the global logger
func LogWarnWithContext(context map[string]interface{}, message string, args ...interface{}) {
	LogWithContextGlobal(WARN, context, message, args...)
}

// LogInfoWithContext logs info with context using the global logger
func LogInfoWithContext(context map[string]interface{}, message string, args ...interface{}) {
	LogWithContextGlobal(INFO, context, message, args...)
}

// LogDebugWithContext logs debug info with context using the global logger
func LogDebugWithContext(context map[string]interface{}, message string, args ...interface{}) {
	LogWithContextGlobal(DEBUG, context, message, args...)
}

// Global lazy logging functions

// LogLazyError logs an error using lazy evaluation
func LogLazyError(lazyFunc LazyLogFunc) {
	if globalLogger != nil {
		globalLogger.LogLazy(ERROR, lazyFunc)
	}
}

// LogLazyWarn logs a warning using lazy evaluation
func LogLazyWarn(lazyFunc LazyLogFunc) {
	if globalLogger != nil {
		globalLogger.LogLazy(WARN, lazyFunc)
	}
}

// LogLazyInfo logs info using lazy evaluation
func LogLazyInfo(lazyFunc LazyLogFunc) {
	if globalLogger != nil {
		globalLogger.LogLazy(INFO, lazyFunc)
	}
}

// LogLazyDebug logs debug info using lazy evaluation
func LogLazyDebug(lazyFunc LazyLogFunc) {
	if globalLogger != nil {
		globalLogger.LogLazy(DEBUG, lazyFunc)
	}
}

// Performance monitoring functions

// GetGlobalPerformanceStats returns performance statistics from the global logger
func GetGlobalPerformanceStats() (logCount int64, bytesLogged int64, logsPerSecond float64, bytesPerSecond float64) {
	if globalLogger != nil {
		return globalLogger.GetPerformanceStats()
	}
	return 0, 0, 0, 0
}

// GetGlobalFileSizeInfo returns file size information from the global logger
func GetGlobalFileSizeInfo() (currentSize int64, maxSize int64, percentUsed float64) {
	if globalLogger != nil {
		return globalLogger.GetFileSizeInfo()
	}
	return 0, 0, 0
}

// SetGlobalMaxFileSize sets the maximum file size for the global logger
func SetGlobalMaxFileSize(maxSize int64) {
	if globalLogger != nil {
		globalLogger.SetMaxFileSize(maxSize)
	}
}

// IsGlobalFileSizeExceeded checks if the global log file has exceeded the maximum size
func IsGlobalFileSizeExceeded() bool {
	if globalLogger != nil {
		return globalLogger.IsFileSizeExceeded()
	}
	return false
}

// RotateGlobalLogFile rotates the global log file
func RotateGlobalLogFile() error {
	if globalLogger != nil {
		return globalLogger.RotateLogFile()
	}
	return fmt.Errorf("no global logger initialized")
}

func init() {
	// If we're in a test build, or if no ENV is set, default to "test"
	if os.Getenv("ENV") == "" {
		err := os.Setenv("ENV", "test")
		if err != nil {
			return
		}
	}
	if err := InitLogger(); err != nil {
		// If we can't init the logger, panic so the tests crash fast
		log.Fatalf("Failed to initialize logger in test init(): %v", err)
	}
}

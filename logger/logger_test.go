package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, test := range tests {
		if got := test.level.String(); got != test.expected {
			t.Errorf("LogLevel(%d).String() = %q, want %q", test.level, got, test.expected)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"DEBUG", int(DEBUG)},
		{"debug", int(DEBUG)},
		{"  DEBUG  ", int(DEBUG)},
		{"INFO", int(INFO)},
		{"info", int(INFO)},
		{"WARN", int(WARN)},
		{"warn", int(WARN)},
		{"WARNING", int(WARN)},
		{"warning", int(WARN)},
		{"ERROR", int(ERROR)},
		{"error", int(ERROR)},
		{"invalid", -1},
		{"", -1},
	}

	for _, test := range tests {
		if got := parseLogLevel(test.input); got != test.expected {
			t.Errorf("parseLogLevel(%q) = %d, want %d", test.input, got, test.expected)
		}
	}
}

func TestNewLogger(t *testing.T) {
	// Save original ENV
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	tests := []struct {
		env           string
		expectedLevel LogLevel
	}{
		{"production", WARN},
		{"development", DEBUG},
		{"dev", DEBUG},
		{"test", WARN},
		{"invalid", WARN},
		{"", WARN}, // Default to production
	}

	for _, test := range tests {
		if test.env == "" {
			os.Unsetenv("ENV")
		} else {
			os.Setenv("ENV", test.env)
		}

		logger := NewLogger()
		if logger.GetLevel() != test.expectedLevel {
			t.Errorf("NewLogger() with ENV=%q: got level %v, want %v", test.env, logger.GetLevel(), test.expectedLevel)
		}
		if logger.env != test.env && test.env != "" {
			t.Errorf("NewLogger() with ENV=%q: got env %q, want %q", test.env, logger.env, test.env)
		}
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger := NewLogger()

	// Test setting different levels
	levels := []LogLevel{DEBUG, INFO, WARN, ERROR}
	for _, level := range levels {
		logger.SetLevel(level)
		if got := logger.GetLevel(); got != level {
			t.Errorf("SetLevel(%v): got %v, want %v", level, got, level)
		}
	}
}

func TestLogger_ShouldLog(t *testing.T) {
	logger := NewLogger()

	tests := []struct {
		loggerLevel  LogLevel
		messageLevel LogLevel
		expected     bool
	}{
		{DEBUG, DEBUG, true},
		{DEBUG, INFO, true},
		{DEBUG, WARN, true},
		{DEBUG, ERROR, true},
		{INFO, DEBUG, false},
		{INFO, INFO, true},
		{INFO, WARN, true},
		{INFO, ERROR, true},
		{WARN, DEBUG, false},
		{WARN, INFO, false},
		{WARN, WARN, true},
		{WARN, ERROR, true},
		{ERROR, DEBUG, false},
		{ERROR, INFO, false},
		{ERROR, WARN, false},
		{ERROR, ERROR, true},
	}

	for _, test := range tests {
		logger.SetLevel(test.loggerLevel)
		if got := logger.ShouldLog(test.messageLevel); got != test.expected {
			t.Errorf("Logger(level=%v).ShouldLog(%v) = %v, want %v",
				test.loggerLevel, test.messageLevel, got, test.expected)
		}
	}
}

func TestSetLevelFromEnvironment(t *testing.T) {
	// Save original environment variables
	originalEnv := os.Getenv("ENV")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
	}()

	tests := []struct {
		env           string
		logLevel      string
		expectedLevel LogLevel
	}{
		// LOG_LEVEL override tests
		{"production", "DEBUG", DEBUG},
		{"production", "INFO", INFO},
		{"production", "WARN", WARN},
		{"production", "ERROR", ERROR},
		{"production", "invalid", WARN}, // Falls back to env-based level

		// Environment-based tests (no LOG_LEVEL override)
		{"production", "", WARN},
		{"development", "", DEBUG},
		{"dev", "", DEBUG},
		{"test", "", WARN},
		{"invalid", "", WARN},
	}

	for _, test := range tests {
		os.Setenv("ENV", test.env)
		if test.logLevel == "" {
			os.Unsetenv("LOG_LEVEL")
		} else {
			os.Setenv("LOG_LEVEL", test.logLevel)
		}

		logger := &Logger{env: test.env}
		logger.setLevelFromEnvironment()

		if logger.level != test.expectedLevel {
			t.Errorf("setLevelFromEnvironment() with ENV=%q, LOG_LEVEL=%q: got %v, want %v",
				test.env, test.logLevel, logger.level, test.expectedLevel)
		}
	}
}

func TestGlobalLoggerFunctions(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Test with nil global logger
	globalLogger = nil
	if ShouldLog(DEBUG) {
		t.Error("ShouldLog(DEBUG) with nil globalLogger should return false")
	}
	if !ShouldLog(WARN) {
		t.Error("ShouldLog(WARN) with nil globalLogger should return true (default to production-safe)")
	}

	// Test with actual global logger
	globalLogger = NewLogger()
	globalLogger.SetLevel(INFO)

	if ShouldLog(DEBUG) {
		t.Error("ShouldLog(DEBUG) with INFO level should return false")
	}
	if !ShouldLog(INFO) {
		t.Error("ShouldLog(INFO) with INFO level should return true")
	}

	// Test SetGlobalLogLevel
	SetGlobalLogLevel(ERROR)
	if globalLogger.GetLevel() != ERROR {
		t.Errorf("SetGlobalLogLevel(ERROR): got %v, want %v", globalLogger.GetLevel(), ERROR)
	}

	// Test GetGlobalLogger
	if GetGlobalLogger() != globalLogger {
		t.Error("GetGlobalLogger() should return the global logger instance")
	}
}

func TestSetLogLevel(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Test with nil global logger
	globalLogger = nil
	SetLogLevel("production") // Should not panic

	// Test with actual global logger
	globalLogger = NewLogger()
	SetLogLevel("development")

	if globalLogger.env != "development" {
		t.Errorf("SetLogLevel('development'): got env %q, want 'development'", globalLogger.env)
	}
	if globalLogger.GetLevel() != DEBUG {
		t.Errorf("SetLogLevel('development'): got level %v, want %v", globalLogger.GetLevel(), DEBUG)
	}
}

func TestInitLogger_TestMode(t *testing.T) {
	// Save original environment and global state
	originalEnv := os.Getenv("ENV")
	originalGlobalLogger := globalLogger
	originalLogFile := logFile
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
		globalLogger = originalGlobalLogger
		logFile = originalLogFile
	}()

	// Set test environment
	os.Setenv("ENV", "test")

	err := InitLogger()
	if err != nil {
		t.Fatalf("InitLogger() in test mode failed: %v", err)
	}

	if globalLogger == nil {
		t.Fatal("InitLogger() should initialize globalLogger")
	}

	// Verify loggers are set up
	if Info == nil || Warn == nil || Error == nil || Debug == nil {
		t.Error("InitLogger() should initialize all legacy loggers")
	}

	// Verify no log file is created in test mode
	if globalLogger.logFile != nil {
		t.Error("InitLogger() in test mode should not create a log file")
	}
}

func TestInitLogger_FileMode(t *testing.T) {
	// Save original environment and global state
	originalEnv := os.Getenv("ENV")
	originalGlobalLogger := globalLogger
	originalLogFile := logFile
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
		globalLogger = originalGlobalLogger
		logFile = originalLogFile
		// Clean up any created log files
		if globalLogger != nil && globalLogger.logFile != nil {
			globalLogger.logFile.Close()
		}
		os.RemoveAll("logs")
	}()

	// Set production environment
	os.Setenv("ENV", "production")

	err := InitLogger()
	if err != nil {
		t.Fatalf("InitLogger() in production mode failed: %v", err)
	}

	if globalLogger == nil {
		t.Fatal("InitLogger() should initialize globalLogger")
	}

	// Verify log file is created
	if globalLogger.logFile == nil {
		t.Error("InitLogger() in production mode should create a log file")
	}

	// Verify logs directory exists
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		t.Error("InitLogger() should create logs directory")
	}
}

func TestLegacyLoggerFiltering(t *testing.T) {
	// Save original global state
	originalGlobalLogger := globalLogger
	originalInfo := Info
	originalWarn := Warn
	originalError := Error
	originalDebug := Debug
	defer func() {
		globalLogger = originalGlobalLogger
		Info = originalInfo
		Warn = originalWarn
		Error = originalError
		Debug = originalDebug
	}()

	// Create a logger with WARN level (production mode)
	globalLogger = NewLogger()
	globalLogger.SetLevel(WARN)

	// Capture output for testing
	var buf bytes.Buffer
	globalLogger.infoLogger = log.New(&buf, "INFO: ", 0)
	globalLogger.warnLogger = log.New(&buf, "WARN: ", 0)
	globalLogger.errorLogger = log.New(&buf, "ERROR: ", 0)
	globalLogger.debugLogger = log.New(&buf, "DEBUG: ", 0)

	// Set up legacy loggers
	globalLogger.setupLegacyLoggers()

	// Test that DEBUG and INFO are discarded
	Debug.Println("debug message")
	Info.Println("info message")

	// Test that WARN and ERROR are logged
	Warn.Println("warn message")
	Error.Println("error message")

	output := buf.String()

	// DEBUG and INFO should not appear in output
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG messages should be discarded in WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO messages should be discarded in WARN level")
	}

	// WARN and ERROR should appear in output
	if !strings.Contains(output, "warn message") {
		t.Error("WARN messages should be logged in WARN level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR messages should be logged in WARN level")
	}
}

func TestCloseLogger(t *testing.T) {
	// Save original global state
	originalGlobalLogger := globalLogger
	originalLogFile := logFile
	defer func() {
		globalLogger = originalGlobalLogger
		logFile = originalLogFile
		os.RemoveAll("logs")
	}()

	// Test with no logger
	globalLogger = nil
	logFile = nil
	if err := CloseLogger(); err != nil {
		t.Errorf("CloseLogger() with no logger should not return error, got: %v", err)
	}

	// Test with global logger having a file
	globalLogger = NewLogger()
	// Create a temporary file for testing
	os.MkdirAll("logs", 0755)
	file, err := os.Create("logs/test.log")
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}
	globalLogger.logFile = file
	logFile = file

	if err := CloseLogger(); err != nil {
		t.Errorf("CloseLogger() should not return error, got: %v", err)
	}

	// Verify file is closed
	if globalLogger.logFile != nil {
		t.Error("CloseLogger() should set globalLogger.logFile to nil")
	}
	if logFile != nil {
		t.Error("CloseLogger() should set logFile to nil")
	}
}

func TestConcurrentAccess(t *testing.T) {
	logger := NewLogger()

	// Test concurrent access to ShouldLog and SetLevel
	done := make(chan bool, 2)

	// Goroutine 1: Continuously check ShouldLog
	go func() {
		for i := 0; i < 1000; i++ {
			logger.ShouldLog(INFO)
		}
		done <- true
	}()

	// Goroutine 2: Continuously change log level
	go func() {
		levels := []LogLevel{DEBUG, INFO, WARN, ERROR}
		for i := 0; i < 1000; i++ {
			logger.SetLevel(levels[i%len(levels)])
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// If we get here without deadlock or race conditions, the test passes
}

func TestPerformanceOptimization(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Only ERROR messages should be logged

	// Capture output
	var buf bytes.Buffer
	logger.errorLogger = log.New(&buf, "ERROR: ", 0)
	logger.setupLegacyLoggers()

	// This expensive operation should not be executed when DEBUG is disabled
	expensiveOperation := func() string {
		time.Sleep(1 * time.Millisecond) // Simulate expensive operation
		return "expensive result"
	}

	start := time.Now()

	// This should be fast because ShouldLog(DEBUG) returns false immediately
	if ShouldLog(DEBUG) {
		Debug.Printf("Debug message: %s", expensiveOperation())
	}

	elapsed := time.Since(start)

	// Should complete very quickly (much less than 1ms) because expensive operation is not called
	if elapsed > 500*time.Microsecond {
		t.Errorf("Conditional logging took too long: %v (expensive operation may have been called)", elapsed)
	}

	// Verify no debug output
	if strings.Contains(buf.String(), "expensive result") {
		t.Error("Expensive operation should not have been called for disabled log level")
	}
}

// Tests for structured logging functionality

func TestLogEntry_JSONMarshaling(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:     "ERROR",
		Message:   "Test message",
		Context: map[string]interface{}{
			"component": "websocket",
			"action":    "connection_failed",
		},
		Source:    "test.go:123",
		MeetName:  "Test Meet",
		RefereeID: "left",
		Error:     "connection timeout",
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	var unmarshaled LogEntry
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal LogEntry: %v", err)
	}

	if unmarshaled.Level != entry.Level {
		t.Errorf("Level mismatch: got %q, want %q", unmarshaled.Level, entry.Level)
	}
	if unmarshaled.Message != entry.Message {
		t.Errorf("Message mismatch: got %q, want %q", unmarshaled.Message, entry.Message)
	}
	if unmarshaled.MeetName != entry.MeetName {
		t.Errorf("MeetName mismatch: got %q, want %q", unmarshaled.MeetName, entry.MeetName)
	}
	if unmarshaled.RefereeID != entry.RefereeID {
		t.Errorf("RefereeID mismatch: got %q, want %q", unmarshaled.RefereeID, entry.RefereeID)
	}
	if unmarshaled.Error != entry.Error {
		t.Errorf("Error mismatch: got %q, want %q", unmarshaled.Error, entry.Error)
	}
}

func TestLogger_LogWithContext(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG) // Enable all logging levels

	// Capture output
	var buf bytes.Buffer
	logger.infoLogger = log.New(&buf, "", 0)
	logger.warnLogger = log.New(&buf, "", 0)
	logger.errorLogger = log.New(&buf, "", 0)
	logger.debugLogger = log.New(&buf, "", 0)

	// Test logging with context
	context := map[string]interface{}{
		"component": "websocket",
		"action":    "connection_failed",
		"meetName":  "Test Meet",
		"refereeId": "left",
		"error":     "connection timeout",
	}

	logger.LogWithContext(ERROR, context, "WebSocket connection failed for referee %s", "left")

	output := buf.String()
	if output == "" {
		t.Fatal("Expected log output, got empty string")
	}

	// Parse the JSON output
	var logEntry LogEntry
	err := json.Unmarshal([]byte(output), &logEntry)
	if err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v\nOutput: %s", err, output)
	}

	// Verify log entry fields
	if logEntry.Level != "ERROR" {
		t.Errorf("Expected level ERROR, got %s", logEntry.Level)
	}
	if !strings.Contains(logEntry.Message, "WebSocket connection failed for referee left") {
		t.Errorf("Expected message to contain formatted text, got: %s", logEntry.Message)
	}
	if logEntry.MeetName != "Test Meet" {
		t.Errorf("Expected MeetName 'Test Meet', got %s", logEntry.MeetName)
	}
	if logEntry.RefereeID != "left" {
		t.Errorf("Expected RefereeID 'left', got %s", logEntry.RefereeID)
	}
	if logEntry.Error != "connection timeout" {
		t.Errorf("Expected Error 'connection timeout', got %s", logEntry.Error)
	}
	if logEntry.Source == "" {
		t.Error("Expected Source to be populated")
	}
}

func TestLogger_LogWithContext_LevelFiltering(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(WARN) // Only WARN and ERROR should be logged

	// Capture output
	var buf bytes.Buffer
	logger.infoLogger = log.New(&buf, "", 0)
	logger.warnLogger = log.New(&buf, "", 0)
	logger.errorLogger = log.New(&buf, "", 0)
	logger.debugLogger = log.New(&buf, "", 0)

	context := map[string]interface{}{
		"component": "test",
	}

	// These should be filtered out
	logger.LogWithContext(DEBUG, context, "Debug message")
	logger.LogWithContext(INFO, context, "Info message")

	// These should be logged
	logger.LogWithContext(WARN, context, "Warn message")
	logger.LogWithContext(ERROR, context, "Error message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should only have 2 lines (WARN and ERROR)
	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d: %v", len(lines), lines)
	}

	// Verify WARN message
	var warnEntry LogEntry
	err := json.Unmarshal([]byte(lines[0]), &warnEntry)
	if err != nil {
		t.Fatalf("Failed to parse WARN log entry: %v", err)
	}
	if warnEntry.Level != "WARN" || !strings.Contains(warnEntry.Message, "Warn message") {
		t.Errorf("Expected WARN message, got: %+v", warnEntry)
	}

	// Verify ERROR message
	var errorEntry LogEntry
	err = json.Unmarshal([]byte(lines[1]), &errorEntry)
	if err != nil {
		t.Fatalf("Failed to parse ERROR log entry: %v", err)
	}
	if errorEntry.Level != "ERROR" || !strings.Contains(errorEntry.Message, "Error message") {
		t.Errorf("Expected ERROR message, got: %+v", errorEntry)
	}
}

func TestContextHelperFunctions(t *testing.T) {
	// Test WebSocket context
	wsContext := NewWebSocketContext("connection_failed", "Test Meet", "left", "192.168.1.100")
	expected := map[string]interface{}{
		"component":  "websocket",
		"action":     "connection_failed",
		"meetName":   "Test Meet",
		"refereeId":  "left",
		"remoteAddr": "192.168.1.100",
	}
	if !contextEqual(wsContext, expected) {
		t.Errorf("NewWebSocketContext() = %+v, want %+v", wsContext, expected)
	}

	// Test Timer context
	timerContext := NewTimerContext("start_failed", "Test Meet", "platformReady", 123)
	expected = map[string]interface{}{
		"component": "timer",
		"action":    "start_failed",
		"meetName":  "Test Meet",
		"timerType": "platformReady",
		"timerId":   123,
	}
	if !contextEqual(timerContext, expected) {
		t.Errorf("NewTimerContext() = %+v, want %+v", timerContext, expected)
	}

	// Test Authentication context
	authContext := NewAuthenticationContext("login_failed", "admin", "192.168.1.100")
	expected = map[string]interface{}{
		"component": "authentication",
		"action":    "login_failed",
		"username":  "admin",
		"ipAddress": "192.168.1.100",
	}
	if !contextEqual(authContext, expected) {
		t.Errorf("NewAuthenticationContext() = %+v, want %+v", authContext, expected)
	}

	// Test Position context
	posContext := NewPositionContext("occupy_failed", "Test Meet", "left", "referee1")
	expected = map[string]interface{}{
		"component": "position",
		"action":    "occupy_failed",
		"meetName":  "Test Meet",
		"position":  "left",
		"refereeId": "referee1",
	}
	if !contextEqual(posContext, expected) {
		t.Errorf("NewPositionContext() = %+v, want %+v", posContext, expected)
	}

	// Test HTTP context
	httpContext := NewHTTPContext("POST", "/api/login", "Mozilla/5.0", "192.168.1.100", 401)
	expected = map[string]interface{}{
		"component":  "http",
		"method":     "POST",
		"path":       "/api/login",
		"userAgent":  "Mozilla/5.0",
		"ipAddress":  "192.168.1.100",
		"statusCode": 401,
	}
	if !contextEqual(httpContext, expected) {
		t.Errorf("NewHTTPContext() = %+v, want %+v", httpContext, expected)
	}

	// Test System context
	sysContext := NewSystemContext("startup", "database")
	expected = map[string]interface{}{
		"component": "system",
		"action":    "startup",
		"subsystem": "database",
	}
	if !contextEqual(sysContext, expected) {
		t.Errorf("NewSystemContext() = %+v, want %+v", sysContext, expected)
	}
}

func TestContextModifierFunctions(t *testing.T) {
	// Test AddError
	context := map[string]interface{}{"component": "test"}
	err := fmt.Errorf("test error")
	result := AddError(context, err)
	if result["error"] != "test error" {
		t.Errorf("AddError() did not add error correctly: %+v", result)
	}

	// Test AddError with nil context
	result = AddError(nil, err)
	if result["error"] != "test error" {
		t.Errorf("AddError() with nil context did not work: %+v", result)
	}

	// Test AddMeetContext
	context = map[string]interface{}{"component": "test"}
	result = AddMeetContext(context, "Test Meet")
	if result["meetName"] != "Test Meet" {
		t.Errorf("AddMeetContext() did not add meetName correctly: %+v", result)
	}

	// Test AddRefereeContext
	context = map[string]interface{}{"component": "test"}
	result = AddRefereeContext(context, "left")
	if result["refereeId"] != "left" {
		t.Errorf("AddRefereeContext() did not add refereeId correctly: %+v", result)
	}
}

func TestGlobalStructuredLoggingFunctions(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Set up test logger
	globalLogger = NewLogger()
	globalLogger.SetLevel(DEBUG)

	// Capture output
	var buf bytes.Buffer
	globalLogger.errorLogger = log.New(&buf, "", 0)
	globalLogger.warnLogger = log.New(&buf, "", 0)
	globalLogger.infoLogger = log.New(&buf, "", 0)
	globalLogger.debugLogger = log.New(&buf, "", 0)

	context := map[string]interface{}{
		"component": "test",
		"meetName":  "Test Meet",
	}

	// Test all global functions
	LogErrorWithContext(context, "Error message")
	LogWarnWithContext(context, "Warn message")
	LogInfoWithContext(context, "Info message")
	LogDebugWithContext(context, "Debug message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	// Verify each line contains expected level
	levels := []string{"ERROR", "WARN", "INFO", "DEBUG"}
	for i, line := range lines {
		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		if err != nil {
			t.Fatalf("Failed to parse log line %d: %v", i, err)
		}
		if entry.Level != levels[i] {
			t.Errorf("Line %d: expected level %s, got %s", i, levels[i], entry.Level)
		}
		if entry.MeetName != "Test Meet" {
			t.Errorf("Line %d: expected meetName 'Test Meet', got %s", i, entry.MeetName)
		}
	}
}

func TestLogWithContext_ErrorHandling(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Capture output
	var buf bytes.Buffer
	logger.errorLogger = log.New(&buf, "", 0)

	// Test with error as string in context
	context := map[string]interface{}{
		"error": "string error",
	}
	logger.LogWithContext(ERROR, context, "Test message")

	// Test with error as error type in context
	context = map[string]interface{}{
		"error": fmt.Errorf("error type error"),
	}
	logger.LogWithContext(ERROR, context, "Test message 2")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	// Check first entry (string error)
	var entry1 LogEntry
	err := json.Unmarshal([]byte(lines[0]), &entry1)
	if err != nil {
		t.Fatalf("Failed to parse first log entry: %v", err)
	}
	if entry1.Error != "string error" {
		t.Errorf("Expected error 'string error', got %s", entry1.Error)
	}

	// Check second entry (error type)
	var entry2 LogEntry
	err = json.Unmarshal([]byte(lines[1]), &entry2)
	if err != nil {
		t.Fatalf("Failed to parse second log entry: %v", err)
	}
	if entry2.Error != "error type error" {
		t.Errorf("Expected error 'error type error', got %s", entry2.Error)
	}
}

func TestLogWithContext_NilGlobalLogger(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Set global logger to nil
	globalLogger = nil

	// These should not panic
	LogErrorWithContext(map[string]interface{}{"test": "value"}, "Test message")
	LogWarnWithContext(map[string]interface{}{"test": "value"}, "Test message")
	LogInfoWithContext(map[string]interface{}{"test": "value"}, "Test message")
	LogDebugWithContext(map[string]interface{}{"test": "value"}, "Test message")
	LogWithContextGlobal(ERROR, map[string]interface{}{"test": "value"}, "Test message")
}

func TestLazyLogging(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Capture output
	var buf bytes.Buffer
	logger.infoLogger = log.New(&buf, "", 0)

	// Test lazy logging with enabled level
	functionCalled := false
	logger.LogLazy(INFO, func() (string, map[string]interface{}) {
		functionCalled = true
		return "Lazy message", map[string]interface{}{
			"component": "test",
			"lazy":      true,
		}
	})

	if !functionCalled {
		t.Error("Lazy function should have been called for enabled log level")
	}

	output := buf.String()
	if !strings.Contains(output, "Lazy message") {
		t.Error("Expected lazy message in output")
	}

	// Test lazy logging with disabled level
	buf.Reset()
	functionCalled = false
	logger.SetLevel(ERROR) // INFO will be filtered out

	logger.LogLazy(INFO, func() (string, map[string]interface{}) {
		functionCalled = true
		return "Should not be called", map[string]interface{}{}
	})

	if functionCalled {
		t.Error("Lazy function should not have been called for disabled log level")
	}

	output = buf.String()
	if strings.Contains(output, "Should not be called") {
		t.Error("Disabled lazy message should not appear in output")
	}
}

func TestGlobalLazyLogging(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Set up test logger
	globalLogger = NewLogger()
	globalLogger.SetLevel(DEBUG)

	// Capture output
	var buf bytes.Buffer
	globalLogger.errorLogger = log.New(&buf, "", 0)
	globalLogger.warnLogger = log.New(&buf, "", 0)
	globalLogger.infoLogger = log.New(&buf, "", 0)
	globalLogger.debugLogger = log.New(&buf, "", 0)

	// Test all global lazy functions
	LogLazyError(func() (string, map[string]interface{}) {
		return "Lazy error", map[string]interface{}{"level": "error"}
	})
	LogLazyWarn(func() (string, map[string]interface{}) {
		return "Lazy warn", map[string]interface{}{"level": "warn"}
	})
	LogLazyInfo(func() (string, map[string]interface{}) {
		return "Lazy info", map[string]interface{}{"level": "info"}
	})
	LogLazyDebug(func() (string, map[string]interface{}) {
		return "Lazy debug", map[string]interface{}{"level": "debug"}
	})

	output := buf.String()
	if !strings.Contains(output, "Lazy error") {
		t.Error("Expected lazy error message in output")
	}
	if !strings.Contains(output, "Lazy warn") {
		t.Error("Expected lazy warn message in output")
	}
	if !strings.Contains(output, "Lazy info") {
		t.Error("Expected lazy info message in output")
	}
	if !strings.Contains(output, "Lazy debug") {
		t.Error("Expected lazy debug message in output")
	}

	// Test with nil global logger
	globalLogger = nil
	// These should not panic
	LogLazyError(func() (string, map[string]interface{}) {
		return "Should not crash", map[string]interface{}{}
	})
	LogLazyWarn(func() (string, map[string]interface{}) {
		return "Should not crash", map[string]interface{}{}
	})
	LogLazyInfo(func() (string, map[string]interface{}) {
		return "Should not crash", map[string]interface{}{}
	})
	LogLazyDebug(func() (string, map[string]interface{}) {
		return "Should not crash", map[string]interface{}{}
	})
}

func TestPerformanceMonitoringFunctions(t *testing.T) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	defer func() {
		globalLogger = originalGlobalLogger
	}()

	// Test with nil global logger
	globalLogger = nil
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := GetGlobalPerformanceStats()
	if logCount != 0 || bytesLogged != 0 || logsPerSecond != 0 || bytesPerSecond != 0 {
		t.Error("Performance stats should be zero with nil global logger")
	}

	currentSize, maxSize, percentUsed := GetGlobalFileSizeInfo()
	if currentSize != 0 || maxSize != 0 || percentUsed != 0 {
		t.Error("File size info should be zero with nil global logger")
	}

	if IsGlobalFileSizeExceeded() {
		t.Error("File size should not be exceeded with nil global logger")
	}

	// Test setting max file size with nil logger (should not panic)
	SetGlobalMaxFileSize(1024)

	// Test rotation with nil logger
	err := RotateGlobalLogFile()
	if err == nil {
		t.Error("Expected error when rotating with nil global logger")
	}

	// Test with actual global logger
	globalLogger = NewLogger()
	globalLogger.SetLevel(DEBUG)
	globalLogger.infoLogger = log.New(io.Discard, "", 0)

	// Log some messages
	context := map[string]interface{}{"component": "test"}
	for i := 0; i < 5; i++ {
		globalLogger.LogWithContext(INFO, context, "Test message %d", i)
	}

	// Test performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond = GetGlobalPerformanceStats()
	if logCount != 5 {
		t.Errorf("Expected log count 5, got %d", logCount)
	}
	if bytesLogged <= 0 {
		t.Errorf("Expected positive bytes logged, got %d", bytesLogged)
	}
	if logsPerSecond <= 0 {
		t.Errorf("Expected positive logs per second, got %f", logsPerSecond)
	}
	if bytesPerSecond <= 0 {
		t.Errorf("Expected positive bytes per second, got %f", bytesPerSecond)
	}

	// Test file size functions
	SetGlobalMaxFileSize(2048)
	_, maxSize, _ = GetGlobalFileSizeInfo()
	if maxSize != 2048 {
		t.Errorf("Expected max size 2048, got %d", maxSize)
	}
}

// Helper function to compare context maps
func contextEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

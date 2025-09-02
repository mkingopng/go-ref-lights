//go:build integration
// +build integration

package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestEnvironmentBasedLoggingIntegration tests the complete logging system
// with different environment configurations
func TestEnvironmentBasedLoggingIntegration(t *testing.T) {
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
		// Clean up any test log files
		os.RemoveAll("test_logs")
	}()

	testCases := []struct {
		name              string
		env               string
		logLevel          string
		expectedLevel     LogLevel
		shouldLogDebug    bool
		shouldLogInfo     bool
		shouldLogWarn     bool
		shouldLogError    bool
		expectFileLogging bool
	}{
		{
			name:              "Production Environment Default",
			env:               "production",
			logLevel:          "",
			expectedLevel:     WARN,
			shouldLogDebug:    false,
			shouldLogInfo:     false,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "Development Environment Default",
			env:               "development",
			logLevel:          "",
			expectedLevel:     DEBUG,
			shouldLogDebug:    true,
			shouldLogInfo:     true,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "Test Environment Default",
			env:               "test",
			logLevel:          "",
			expectedLevel:     WARN,
			shouldLogDebug:    false,
			shouldLogInfo:     false,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: false,
		},
		{
			name:              "Production with DEBUG Override",
			env:               "production",
			logLevel:          "DEBUG",
			expectedLevel:     DEBUG,
			shouldLogDebug:    true,
			shouldLogInfo:     true,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "Development with ERROR Override",
			env:               "development",
			logLevel:          "ERROR",
			expectedLevel:     ERROR,
			shouldLogDebug:    false,
			shouldLogInfo:     false,
			shouldLogWarn:     false,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "Invalid Environment Fallback",
			env:               "invalid",
			logLevel:          "",
			expectedLevel:     WARN,
			shouldLogDebug:    false,
			shouldLogInfo:     false,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "No Environment Fallback",
			env:               "",
			logLevel:          "",
			expectedLevel:     WARN,
			shouldLogDebug:    false,
			shouldLogInfo:     false,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
		{
			name:              "Invalid LOG_LEVEL Fallback",
			env:               "development",
			logLevel:          "invalid",
			expectedLevel:     DEBUG,
			shouldLogDebug:    true,
			shouldLogInfo:     true,
			shouldLogWarn:     true,
			shouldLogError:    true,
			expectFileLogging: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment
			if tc.env == "" {
				os.Unsetenv("ENV")
			} else {
				os.Setenv("ENV", tc.env)
			}
			if tc.logLevel == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tc.logLevel)
			}

			// Create a new logger instance
			logger := NewLogger()

			// Verify log level is set correctly
			if logger.GetLevel() != tc.expectedLevel {
				t.Errorf("Expected log level %v, got %v", tc.expectedLevel, logger.GetLevel())
			}

			// Test ShouldLog behavior
			if logger.ShouldLog(DEBUG) != tc.shouldLogDebug {
				t.Errorf("ShouldLog(DEBUG) = %v, want %v", logger.ShouldLog(DEBUG), tc.shouldLogDebug)
			}
			if logger.ShouldLog(INFO) != tc.shouldLogInfo {
				t.Errorf("ShouldLog(INFO) = %v, want %v", logger.ShouldLog(INFO), tc.shouldLogInfo)
			}
			if logger.ShouldLog(WARN) != tc.shouldLogWarn {
				t.Errorf("ShouldLog(WARN) = %v, want %v", logger.ShouldLog(WARN), tc.shouldLogWarn)
			}
			if logger.ShouldLog(ERROR) != tc.shouldLogError {
				t.Errorf("ShouldLog(ERROR) = %v, want %v", logger.ShouldLog(ERROR), tc.shouldLogError)
			}

			// Test file logging behavior
			testFileLogging(t, tc.expectedLevel, tc.expectFileLogging)
		})
	}
}

// testFileLogging verifies that file logging works as expected for different log levels
func testFileLogging(t *testing.T, expectedLevel LogLevel, expectFileLogging bool) {
	// Save original global logger
	originalGlobalLogger := globalLogger
	originalLogFile := logFile
	defer func() {
		globalLogger = originalGlobalLogger
		logFile = originalLogFile
		CloseLogger()
		os.RemoveAll("test_logs")
	}()

	// Change to a test logs directory to avoid conflicts
	originalDir, _ := os.Getwd()
	testDir := filepath.Join(originalDir, "test_logs")
	os.MkdirAll(testDir, 0755)
	os.Chdir(testDir)
	defer os.Chdir(originalDir)

	// Initialize logger
	err := InitLogger()
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	if expectFileLogging {
		// Verify log file was created
		if globalLogger.logFile == nil {
			t.Error("Expected log file to be created, but it was nil")
		}

		// Verify logs directory exists
		if _, err := os.Stat("logs"); os.IsNotExist(err) {
			t.Error("Expected logs directory to be created")
		}
	} else {
		// Verify no log file was created (test mode)
		if globalLogger.logFile != nil {
			t.Error("Expected no log file in test mode, but one was created")
		}
	}

	// Test actual logging
	testContext := map[string]interface{}{
		"component": "integration_test",
		"meetName":  "Test Meet",
		"refereeId": "left",
	}

	// Log messages at different levels
	LogDebugWithContext(testContext, "Debug message for integration test")
	LogInfoWithContext(testContext, "Info message for integration test")
	LogWarnWithContext(testContext, "Warn message for integration test")
	LogErrorWithContext(testContext, "Error message for integration test")

	// If file logging is expected, verify log file contents
	if expectFileLogging && globalLogger.logFile != nil {
		// Close the logger to flush any buffered content
		CloseLogger()

		// Find the log file
		logFiles, err := filepath.Glob("logs/*.log")
		if err != nil || len(logFiles) == 0 {
			t.Fatalf("Failed to find log files: %v", err)
		}

		// Read and verify log file contents
		logContent, err := os.ReadFile(logFiles[0])
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		verifyLogContent(t, string(logContent), expectedLevel)
	}
}

// verifyLogContent checks that the log file contains expected messages based on log level
func verifyLogContent(t *testing.T, content string, expectedLevel LogLevel) {
	lines := strings.Split(strings.TrimSpace(content), "\n")

	// Filter out empty lines
	var logLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			logLines = append(logLines, line)
		}
	}

	// Count messages by level
	debugCount := 0
	infoCount := 0
	warnCount := 0
	errorCount := 0

	for _, line := range logLines {
		// Skip non-JSON lines (like legacy format lines)
		if !strings.Contains(line, "{") {
			continue
		}

		// Extract JSON part (after the prefix like "INFO: ")
		jsonStart := strings.Index(line, "{")
		if jsonStart == -1 {
			continue
		}
		jsonPart := line[jsonStart:]

		var logEntry LogEntry
		if err := json.Unmarshal([]byte(jsonPart), &logEntry); err != nil {
			// Skip lines that aren't valid JSON log entries
			continue
		}

		// Only count our integration test messages
		if logEntry.Context != nil {
			if component, ok := logEntry.Context["component"].(string); ok && component == "integration_test" {
				switch logEntry.Level {
				case "DEBUG":
					debugCount++
				case "INFO":
					infoCount++
				case "WARN":
					warnCount++
				case "ERROR":
					errorCount++
				}
			}
		}
	}

	// Verify expected message counts based on log level
	switch expectedLevel {
	case DEBUG:
		if debugCount == 0 {
			t.Error("DEBUG level should log DEBUG messages, found none")
		}
		if infoCount == 0 {
			t.Error("DEBUG level should log INFO messages, found none")
		}
		if warnCount == 0 {
			t.Error("DEBUG level should log WARN messages, found none")
		}
		if errorCount == 0 {
			t.Error("DEBUG level should log ERROR messages, found none")
		}
	case INFO:
		if debugCount > 0 {
			t.Errorf("INFO level should not log DEBUG messages, found %d", debugCount)
		}
		if infoCount == 0 {
			t.Error("INFO level should log INFO messages, found none")
		}
		if warnCount == 0 {
			t.Error("INFO level should log WARN messages, found none")
		}
		if errorCount == 0 {
			t.Error("INFO level should log ERROR messages, found none")
		}
	case WARN:
		if debugCount > 0 {
			t.Errorf("WARN level should not log DEBUG messages, found %d", debugCount)
		}
		if infoCount > 0 {
			t.Errorf("WARN level should not log INFO messages, found %d", infoCount)
		}
		if warnCount == 0 {
			t.Error("WARN level should log WARN messages, found none")
		}
		if errorCount == 0 {
			t.Error("WARN level should log ERROR messages, found none")
		}
	case ERROR:
		if debugCount > 0 {
			t.Errorf("ERROR level should not log DEBUG messages, found %d", debugCount)
		}
		if infoCount > 0 {
			t.Errorf("ERROR level should not log INFO messages, found %d", infoCount)
		}
		if warnCount > 0 {
			t.Errorf("ERROR level should not log WARN messages, found %d", warnCount)
		}
		if errorCount == 0 {
			t.Error("ERROR level should log ERROR messages, found none")
		}
	}
}

// TestCompleteLoggingWorkflow tests the entire logging workflow from initialization to cleanup
func TestCompleteLoggingWorkflow(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
		CloseLogger()
		os.RemoveAll("workflow_test_logs")
	}()

	// Test in a separate directory
	originalDir, _ := os.Getwd()
	testDir := filepath.Join(originalDir, "workflow_test_logs")
	os.MkdirAll(testDir, 0755)
	os.Chdir(testDir)
	defer os.Chdir(originalDir)

	// Set production environment
	os.Setenv("ENV", "production")

	// Step 1: Initialize logger
	err := InitLogger()
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	// Step 2: Verify global logger is set up
	if globalLogger == nil {
		t.Fatal("Global logger should be initialized")
	}

	// Step 3: Test structured logging with various contexts
	contexts := []map[string]interface{}{
		NewWebSocketContext("connection_failed", "Test Meet", "left", "192.168.1.100"),
		NewTimerContext("start_failed", "Test Meet", "platformReady", 123),
		NewAuthenticationContext("login_failed", "admin", "192.168.1.100"),
		NewPositionContext("occupy_failed", "Test Meet", "center", "referee2"),
		NewHTTPContext("POST", "/api/login", "Mozilla/5.0", "192.168.1.100", 401),
		NewSystemContext("startup", "database"),
	}

	for i, ctx := range contexts {
		LogErrorWithContext(ctx, "Test error message %d", i+1)
		LogWarnWithContext(ctx, "Test warning message %d", i+1)
		// These should be filtered out in production
		LogInfoWithContext(ctx, "Test info message %d", i+1)
		LogDebugWithContext(ctx, "Test debug message %d", i+1)
	}

	// Step 4: Test runtime level changes
	originalLevel := globalLogger.GetLevel()
	SetGlobalLogLevel(DEBUG)
	if globalLogger.GetLevel() != DEBUG {
		t.Error("SetGlobalLogLevel should change the log level")
	}

	// Log a debug message that should now be visible
	LogDebugWithContext(NewSystemContext("test", "level_change"), "Debug message after level change")

	// Restore original level
	SetGlobalLogLevel(originalLevel)

	// Step 5: Test legacy logger compatibility
	if Error == nil || Warn == nil {
		t.Error("Legacy loggers should be initialized")
	}

	Error.Println("Legacy error message")
	Warn.Println("Legacy warning message")

	// Step 6: Close logger and verify cleanup
	err = CloseLogger()
	if err != nil {
		t.Errorf("CloseLogger() failed: %v", err)
	}

	// Step 7: Verify log file was created and contains expected content
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatalf("Expected log file to be created, found: %v", logFiles)
	}

	logContent, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(logContent)

	// Verify structured log entries are present
	if !strings.Contains(content, "websocket") {
		t.Error("Expected WebSocket context in log file")
	}
	if !strings.Contains(content, "timer") {
		t.Error("Expected Timer context in log file")
	}
	if !strings.Contains(content, "authentication") {
		t.Error("Expected Authentication context in log file")
	}

	// Verify production filtering (no DEBUG/INFO messages from our structured logging)
	lines := strings.Split(content, "\n")
	structuredDebugCount := 0
	structuredInfoCount := 0

	for _, line := range lines {
		if strings.Contains(line, "{") {
			jsonStart := strings.Index(line, "{")
			if jsonStart != -1 {
				jsonPart := line[jsonStart:]
				var logEntry LogEntry
				if json.Unmarshal([]byte(jsonPart), &logEntry) == nil {
					if logEntry.Level == "DEBUG" && strings.Contains(logEntry.Message, "Test debug message") {
						structuredDebugCount++
					}
					if logEntry.Level == "INFO" && strings.Contains(logEntry.Message, "Test info message") {
						structuredInfoCount++
					}
				}
			}
		}
	}

	// In production mode, we should have filtered out the initial DEBUG/INFO messages
	// but allowed the one DEBUG message after level change
	if structuredInfoCount > 0 {
		t.Errorf("Production mode should filter INFO messages, found %d", structuredInfoCount)
	}
}

// TestEnvironmentConfigurationEdgeCases tests edge cases in environment configuration
func TestEnvironmentConfigurationEdgeCases(t *testing.T) {
	// Save original environment
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

	testCases := []struct {
		name          string
		env           string
		logLevel      string
		expectedLevel LogLevel
	}{
		{
			name:          "Case insensitive environment",
			env:           "PRODUCTION",
			logLevel:      "",
			expectedLevel: WARN,
		},
		{
			name:          "Case insensitive log level",
			env:           "production",
			logLevel:      "debug",
			expectedLevel: DEBUG,
		},
		{
			name:          "Whitespace in log level",
			env:           "production",
			logLevel:      "  INFO  ",
			expectedLevel: INFO,
		},
		{
			name:          "Alternative warning format",
			env:           "production",
			logLevel:      "WARNING",
			expectedLevel: WARN,
		},
		{
			name:          "Dev shorthand",
			env:           "dev",
			logLevel:      "",
			expectedLevel: DEBUG,
		},
		{
			name:          "Mixed case environment",
			env:           "Development",
			logLevel:      "",
			expectedLevel: DEBUG,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv("ENV", tc.env)
			if tc.logLevel == "" {
				os.Unsetenv("LOG_LEVEL")
			} else {
				os.Setenv("LOG_LEVEL", tc.logLevel)
			}

			logger := NewLogger()
			if logger.GetLevel() != tc.expectedLevel {
				t.Errorf("Expected log level %v, got %v", tc.expectedLevel, logger.GetLevel())
			}
		})
	}
}

// TestConcurrentEnvironmentConfiguration tests thread safety of environment configuration
func TestConcurrentEnvironmentConfiguration(t *testing.T) {
	logger := NewLogger()

	// Test concurrent access to environment-based configuration
	done := make(chan bool, 3)

	// Goroutine 1: Continuously check ShouldLog
	go func() {
		for i := 0; i < 1000; i++ {
			logger.ShouldLog(INFO)
			logger.ShouldLog(DEBUG)
			logger.ShouldLog(WARN)
			logger.ShouldLog(ERROR)
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

	// Goroutine 3: Continuously call setLevelFromEnvironment
	go func() {
		for i := 0; i < 1000; i++ {
			logger.setLevelFromEnvironment()
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out - possible deadlock")
		}
	}
}

// TestLogFileRotationAndSize tests log file creation and basic size monitoring
func TestLogFileRotationAndSize(t *testing.T) {
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
		CloseLogger()
		os.RemoveAll("size_test_logs")
	}()

	// Test in a separate directory
	originalDir, _ := os.Getwd()
	testDir := filepath.Join(originalDir, "size_test_logs")
	os.MkdirAll(testDir, 0755)
	os.Chdir(testDir)
	defer os.Chdir(originalDir)

	// Set production environment
	os.Setenv("ENV", "production")

	// Initialize logger
	err := InitLogger()
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	// Generate a reasonable amount of log data
	context := NewSystemContext("size_test", "logging")
	for i := 0; i < 100; i++ {
		LogErrorWithContext(context, "Error message %d with some additional context data to increase size", i)
		LogWarnWithContext(context, "Warning message %d with some additional context data to increase size", i)
	}

	// Close logger to flush content
	CloseLogger()

	// Check that log file was created
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatalf("Expected log file to be created")
	}

	// Check file size is reasonable (not empty, not excessively large)
	fileInfo, err := os.Stat(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to get log file info: %v", err)
	}

	if fileInfo.Size() == 0 {
		t.Error("Log file should not be empty")
	}

	// For 200 structured log messages, expect reasonable size (should be > 1KB but < 1MB)
	if fileInfo.Size() < 1024 {
		t.Errorf("Log file seems too small: %d bytes", fileInfo.Size())
	}
	if fileInfo.Size() > 1024*1024 {
		t.Errorf("Log file seems too large: %d bytes", fileInfo.Size())
	}

	t.Logf("Log file size: %d bytes for 200 messages", fileInfo.Size())
}

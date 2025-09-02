//go:build validation
// +build validation

package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestComprehensiveLoggingSystemValidation is the main comprehensive test suite
// that validates all logging optimization requirements are met
func TestComprehensiveLoggingSystemValidation(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

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

	// Run all validation test suites
	t.Run("ProductionModeValidation", testProductionModeValidation)
	t.Run("DevelopmentModeValidation", testDevelopmentModeValidation)
	t.Run("EnvironmentConfigurationValidation", testEnvironmentConfigurationValidation)
	t.Run("StructuredLoggingValidation", testStructuredLoggingValidation)
	t.Run("PerformanceOptimizationValidation", testPerformanceOptimizationValidation)
	t.Run("ErrorCategorizationValidation", testErrorCategorizationValidation)
	t.Run("LogFileSizeValidation", testLogFileSizeValidation)
	t.Run("ConcurrentLoggingValidation", testConcurrentLoggingValidation)
	t.Run("AllRequirementsValidation", testAllRequirementsValidation)
}

// testProductionModeValidation validates production logging behavior
func testProductionModeValidation(t *testing.T) {
	setupProductionEnvironment(t)
	validateProductionLogLevels(t)

	stats := runProductionLoggingSimulation(t)
	validateProductionLogFile(t, stats)
	validateProductionLogContent(t)
}

// setupProductionEnvironment prepares the test environment for production mode validation
func setupProductionEnvironment(t *testing.T) {
	os.RemoveAll("logs")
	os.Setenv("ENV", "production")
	os.Unsetenv("LOG_LEVEL")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger in production mode: %v", err)
	}
	t.Cleanup(func() { CloseLogger() })
}

// validateProductionLogLevels ensures production mode has correct log level filtering
func validateProductionLogLevels(t *testing.T) {
	if !ShouldLog(ERROR) {
		t.Error("ERROR should be logged in production mode")
	}
	if !ShouldLog(WARN) {
		t.Error("WARN should be logged in production mode")
	}
	if ShouldLog(INFO) {
		t.Error("INFO should not be logged in production mode by default")
	}
	if ShouldLog(DEBUG) {
		t.Error("DEBUG should not be logged in production mode")
	}
}

// ProductionSimulationStats holds metrics from production logging simulation
type ProductionSimulationStats struct {
	Duration      time.Duration
	ErrorCount    int
	WarnCount     int
	FilteredCount int
	ActualSize    int64
	MBPerHour     float64
}

// runProductionLoggingSimulation simulates realistic production logging patterns
func runProductionLoggingSimulation(t *testing.T) ProductionSimulationStats {
	start := time.Now()
	duration := 10 * time.Second

	stats := ProductionSimulationStats{}

	for time.Since(start) < duration {
		// Simulate production errors (rare)
		if stats.ErrorCount < 3 {
			LogErrorWithContext(
				NewWebSocketErrorContext("Connection failed", "ProductionMeet", "left", "192.168.1.100").ToLogContext(),
				"WebSocket connection failed for referee %s in meet %s", "left", "ProductionMeet")
			stats.ErrorCount++
		}

		// Simulate production warnings (occasional)
		if stats.WarnCount < 5 {
			LogWarnWithContext(
				NewAuthenticationContext("login_attempt", "admin", "192.168.1.100"),
				"Authentication attempt from IP %s", "192.168.1.100")
			stats.WarnCount++
		}

		// These should be filtered out in production
		LogInfoWithContext(map[string]interface{}{"component": "websocket"}, "WebSocket message processed")
		LogDebugWithContext(map[string]interface{}{"component": "timer"}, "Timer tick: %d", time.Now().Unix())
		stats.FilteredCount += 2

		time.Sleep(100 * time.Millisecond)
	}

	stats.Duration = time.Since(start)
	return stats
}

// validateProductionLogFile checks log file creation and size requirements
func validateProductionLogFile(t *testing.T, stats ProductionSimulationStats) {
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatal("Expected log file to be created in production mode")
	}

	stat, err := os.Stat(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	stats.ActualSize = stat.Size()

	// Extrapolate to 1 hour
	bytesPerHour := float64(stats.ActualSize) * (float64(time.Hour) / float64(stats.Duration))
	stats.MBPerHour = bytesPerHour / (1024 * 1024)

	t.Logf("Production mode validation results:")
	t.Logf("  Test duration: %v", stats.Duration)
	t.Logf("  Log file size: %d bytes", stats.ActualSize)
	t.Logf("  Extrapolated MB/hour: %.2f", stats.MBPerHour)
	t.Logf("  Errors logged: %d", stats.ErrorCount)
	t.Logf("  Warnings logged: %d", stats.WarnCount)
	t.Logf("  Messages filtered: %d", stats.FilteredCount)

	// Validate against 10MB/hour requirement
	if stats.MBPerHour > 10.0 {
		t.Errorf("Production logging exceeds 10MB/hour requirement: %.2f MB/hour", stats.MBPerHour)
	}
}

// validateProductionLogContent ensures log content meets production requirements
func validateProductionLogContent(t *testing.T) {
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatal("Expected log file to be created")
	}

	content, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Should contain structured error and warning messages
	if !strings.Contains(logContent, "WebSocket connection failed") {
		t.Error("Production logs should contain error messages")
	}
	if !strings.Contains(logContent, "Authentication attempt") {
		t.Error("Production logs should contain warning messages")
	}

	// Should not contain filtered debug/info messages
	if strings.Contains(logContent, "WebSocket message processed") {
		t.Error("Production logs should not contain INFO messages")
	}
	if strings.Contains(logContent, "Timer tick") {
		t.Error("Production logs should not contain DEBUG messages")
	}

	// Validate JSON structure of log entries
	validateJSONLogStructure(t, logContent)
}

// testDevelopmentModeValidation validates development logging behavior
func testDevelopmentModeValidation(t *testing.T) {
	// Clean up any existing logs
	os.RemoveAll("logs")

	// Set development environment
	os.Setenv("ENV", "development")
	os.Unsetenv("LOG_LEVEL")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger in development mode: %v", err)
	}
	defer CloseLogger()

	// Validate development log levels
	if !ShouldLog(ERROR) {
		t.Error("ERROR should be logged in development mode")
	}
	if !ShouldLog(WARN) {
		t.Error("WARN should be logged in development mode")
	}
	if !ShouldLog(INFO) {
		t.Error("INFO should be logged in development mode")
	}
	if !ShouldLog(DEBUG) {
		t.Error("DEBUG should be logged in development mode")
	}

	// Test comprehensive logging in development
	contexts := []map[string]interface{}{
		NewWebSocketContext("message_received", "DevMeet", "left", "192.168.1.100"),
		NewTimerContext("countdown_update", "DevMeet", "platformReady", 123),
		NewAuthenticationContext("login_success", "developer", "127.0.0.1"),
		NewPositionContext("position_occupied", "DevMeet", "center", "dev_referee"),
		NewHTTPContext("GET", "/api/status", "Mozilla/5.0", "127.0.0.1", 200),
		NewSystemContext("startup", "websocket_server"),
	}

	for i, ctx := range contexts {
		LogDebugWithContext(ctx, "Debug message %d for development", i+1)
		LogInfoWithContext(ctx, "Info message %d for development", i+1)
		LogWarnWithContext(ctx, "Warning message %d for development", i+1)
		LogErrorWithContext(ctx, "Error message %d for development", i+1)
	}

	// Validate log file was created
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatal("Expected log file to be created in development mode")
	}

	// Validate log content contains all levels
	content, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Should contain all log levels
	if !strings.Contains(logContent, "\"level\":\"DEBUG\"") {
		t.Error("Development logs should contain DEBUG messages")
	}
	if !strings.Contains(logContent, "\"level\":\"INFO\"") {
		t.Error("Development logs should contain INFO messages")
	}
	if !strings.Contains(logContent, "\"level\":\"WARN\"") {
		t.Error("Development logs should contain WARN messages")
	}
	if !strings.Contains(logContent, "\"level\":\"ERROR\"") {
		t.Error("Development logs should contain ERROR messages")
	}

	// Validate context information is preserved
	if !strings.Contains(logContent, "websocket") {
		t.Error("Development logs should contain WebSocket context")
	}
	if !strings.Contains(logContent, "timer") {
		t.Error("Development logs should contain Timer context")
	}
	if !strings.Contains(logContent, "authentication") {
		t.Error("Development logs should contain Authentication context")
	}

	validateJSONLogStructure(t, logContent)
}

// testEnvironmentConfigurationValidation validates environment-based configuration
func testEnvironmentConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name           string
		env            string
		logLevel       string
		expectedLevel  LogLevel
		shouldLogDebug bool
		shouldLogInfo  bool
		shouldLogWarn  bool
		shouldLogError bool
	}{
		{
			name:           "Production Default",
			env:            "production",
			logLevel:       "",
			expectedLevel:  WARN,
			shouldLogDebug: false,
			shouldLogInfo:  false,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
		{
			name:           "Development Default",
			env:            "development",
			logLevel:       "",
			expectedLevel:  DEBUG,
			shouldLogDebug: true,
			shouldLogInfo:  true,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
		{
			name:           "Test Environment",
			env:            "test",
			logLevel:       "",
			expectedLevel:  WARN,
			shouldLogDebug: false,
			shouldLogInfo:  false,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
		{
			name:           "Production with DEBUG Override",
			env:            "production",
			logLevel:       "DEBUG",
			expectedLevel:  DEBUG,
			shouldLogDebug: true,
			shouldLogInfo:  true,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
		{
			name:           "Development with ERROR Override",
			env:            "development",
			logLevel:       "ERROR",
			expectedLevel:  ERROR,
			shouldLogDebug: false,
			shouldLogInfo:  false,
			shouldLogWarn:  false,
			shouldLogError: true,
		},
		{
			name:           "Invalid Environment Fallback",
			env:            "invalid",
			logLevel:       "",
			expectedLevel:  WARN,
			shouldLogDebug: false,
			shouldLogInfo:  false,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
		{
			name:           "No Environment Fallback",
			env:            "",
			logLevel:       "",
			expectedLevel:  WARN,
			shouldLogDebug: false,
			shouldLogInfo:  false,
			shouldLogWarn:  true,
			shouldLogError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
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

			// Create new logger instance
			logger := NewLogger()

			// Validate log level
			if logger.GetLevel() != tc.expectedLevel {
				t.Errorf("Expected log level %v, got %v", tc.expectedLevel, logger.GetLevel())
			}

			// Validate ShouldLog behavior
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
		})
	}
}

// testStructuredLoggingValidation validates structured logging with context
func testStructuredLoggingValidation(t *testing.T) {
	// Clean up any existing logs
	os.RemoveAll("logs")

	os.Setenv("ENV", "development")
	os.Unsetenv("LOG_LEVEL")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Test all context helper functions
	testContexts := []struct {
		name    string
		context map[string]interface{}
		message string
	}{
		{
			name:    "WebSocket Context",
			context: NewWebSocketContext("connection_failed", "TestMeet", "left", "192.168.1.100"),
			message: "WebSocket connection failed for referee",
		},
		{
			name:    "Timer Context",
			context: NewTimerContext("start_failed", "TestMeet", "platformReady", 123),
			message: "Timer failed to start",
		},
		{
			name:    "Authentication Context",
			context: NewAuthenticationContext("login_failed", "admin", "192.168.1.100"),
			message: "Authentication failed for user",
		},
		{
			name:    "Position Context",
			context: NewPositionContext("occupy_failed", "TestMeet", "center", "referee1"),
			message: "Failed to occupy position",
		},
		{
			name:    "HTTP Context",
			context: NewHTTPContext("POST", "/api/login", "Mozilla/5.0", "192.168.1.100", 401),
			message: "HTTP request failed",
		},
		{
			name:    "System Context",
			context: NewSystemContext("startup", "database"),
			message: "System component started",
		},
	}

	for _, tc := range testContexts {
		t.Run(tc.name, func(t *testing.T) {
			LogErrorWithContext(tc.context, tc.message)
		})
	}

	// Test error categorization system
	errorContexts := []struct {
		name           string
		errorCtx       *ErrorContext
		expectedFields []string
	}{
		{
			name:           "WebSocket Error",
			errorCtx:       NewWebSocketErrorContext("Connection timeout", "TestMeet", "left", "192.168.1.100"),
			expectedFields: []string{"errorCategory", "meetName", "refereeId", "ipAddress"},
		},
		{
			name:           "Authentication Error",
			errorCtx:       NewAuthenticationErrorContext("Invalid credentials", "admin", "192.168.1.100", "Mozilla/5.0"),
			expectedFields: []string{"errorCategory", "username", "ipAddress", "userAgent"},
		},
		{
			name:           "Timer Error",
			errorCtx:       NewTimerErrorContext("Timer start failed", "TestMeet", "platformReady", 123),
			expectedFields: []string{"errorCategory", "meetName", "timerType", "timerId"},
		},
		{
			name:           "Position Error",
			errorCtx:       NewPositionErrorContext("Position occupied", "TestMeet", "left", "user123"),
			expectedFields: []string{"errorCategory", "meetName", "position", "userId"},
		},
	}

	for _, tc := range errorContexts {
		t.Run(tc.name, func(t *testing.T) {
			tc.errorCtx.LogError()

			// Validate error context structure
			logContext := tc.errorCtx.ToLogContext()
			for _, field := range tc.expectedFields {
				if _, exists := logContext[field]; !exists {
					t.Errorf("Expected field %s not found in error context", field)
				}
			}
		})
	}

	// Validate log file content
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatal("Expected log file to be created")
	}

	content, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	validateJSONLogStructure(t, string(content))
	validateContextInformation(t, string(content))
}

// testPerformanceOptimizationValidation validates performance optimizations
func testPerformanceOptimizationValidation(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Only ERROR messages should be logged

	// Test 1: Conditional logging performance
	t.Run("ConditionalLogging", func(t *testing.T) {
		expensiveOperationCount := 0
		expensiveOperation := func() string {
			expensiveOperationCount++
			time.Sleep(1 * time.Millisecond)
			return "expensive result"
		}

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			if logger.ShouldLog(DEBUG) {
				logger.LogWithContext(DEBUG, map[string]interface{}{"test": "value"},
					"Debug: %s", expensiveOperation())
			}
		}

		elapsed := time.Since(start)

		if elapsed > 10*time.Millisecond {
			t.Errorf("Conditional logging too slow: %v for %d iterations", elapsed, iterations)
		}

		if expensiveOperationCount > 0 {
			t.Errorf("Expensive operation called %d times, should be 0", expensiveOperationCount)
		}
	})

	// Test 2: Lazy evaluation performance
	t.Run("LazyEvaluation", func(t *testing.T) {
		lazyFunctionCallCount := 0

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
				lazyFunctionCallCount++
				time.Sleep(1 * time.Millisecond)
				return "lazy result", map[string]interface{}{"test": "value"}
			})
		}

		elapsed := time.Since(start)

		if elapsed > 10*time.Millisecond {
			t.Errorf("Lazy logging too slow: %v for %d iterations", elapsed, iterations)
		}

		if lazyFunctionCallCount > 0 {
			t.Errorf("Lazy function called %d times, should be 0", lazyFunctionCallCount)
		}
	})

	// Test 3: LogIfEnabled performance
	t.Run("LogIfEnabled", func(t *testing.T) {
		expensiveContextCallCount := 0

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			logger.LogIfEnabled(DEBUG, func() map[string]interface{} {
				expensiveContextCallCount++
				time.Sleep(1 * time.Millisecond)
				return map[string]interface{}{"expensive": "context"}
			}, "Message %d", i)
		}

		elapsed := time.Since(start)

		if elapsed > 10*time.Millisecond {
			t.Errorf("LogIfEnabled too slow: %v for %d iterations", elapsed, iterations)
		}

		if expensiveContextCallCount > 0 {
			t.Errorf("Expensive context function called %d times, should be 0", expensiveContextCallCount)
		}
	})

	// Test 4: Performance monitoring
	t.Run("PerformanceMonitoring", func(t *testing.T) {
		logger.SetLevel(DEBUG)
		logger.infoLogger = log.New(io.Discard, "", 0)

		context := map[string]interface{}{"component": "test"}
		for i := 0; i < 100; i++ {
			logger.LogWithContext(INFO, context, "Performance test message %d", i)
		}

		logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

		if logCount != 100 {
			t.Errorf("Expected 100 logs, got %d", logCount)
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

		// Test detailed performance stats
		stats := logger.GetDetailedPerformanceStats()
		expectedFields := []string{
			"logCount", "bytesLogged", "logsPerSecond", "bytesPerSecond",
			"rotationCount", "currentFileSize", "maxFileSize", "uptime",
		}

		for _, field := range expectedFields {
			if _, exists := stats[field]; !exists {
				t.Errorf("Expected field %s not found in detailed performance stats", field)
			}
		}
	})
}

// testErrorCategorizationValidation validates the error categorization system
func testErrorCategorizationValidation(t *testing.T) {
	// Test all error categories
	categories := []ErrorCategory{
		AuthenticationError,
		AuthorizationError,
		ValidationError,
		NetworkError,
		DatabaseError,
		ConfigurationError,
		BusinessLogicError,
		SystemError,
		WebSocketError,
		TimerError,
		PositionError,
		SessionError,
		MarshalingError,
		FileSystemError,
	}

	severities := []ErrorSeverity{
		SeverityCritical,
		SeverityHigh,
		SeverityMedium,
		SeverityLow,
	}

	for _, category := range categories {
		for _, severity := range severities {
			t.Run(fmt.Sprintf("%s_%s", category, severity), func(t *testing.T) {
				errorCtx := NewErrorContext(category, severity, "Test error message")

				if errorCtx.Category != category {
					t.Errorf("Expected category %s, got %s", category, errorCtx.Category)
				}

				if errorCtx.Severity != severity {
					t.Errorf("Expected severity %s, got %s", severity, errorCtx.Severity)
				}

				if errorCtx.Message == "" {
					t.Error("Error message should not be empty")
				}

				if errorCtx.Timestamp.IsZero() {
					t.Error("Timestamp should be set")
				}

				// Test method chaining
				errorCtx = errorCtx.
					WithCode("TEST001").
					WithDetail("testDetail", "testValue").
					WithContext("testContext", "testValue").
					WithUser("testUser", "testSession").
					WithRequest("testRequest", "192.168.1.100", "TestAgent").
					WithMeet("TestMeet", "testReferee").
					WithError(fmt.Errorf("underlying error"))

				// Validate chained values
				if errorCtx.Code != "TEST001" {
					t.Error("Code should be set via WithCode")
				}

				if errorCtx.Details["testDetail"] != "testValue" {
					t.Error("Detail should be set via WithDetail")
				}

				if errorCtx.UserID != "testUser" {
					t.Error("UserID should be set via WithUser")
				}

				// Test ToLogContext conversion
				logContext := errorCtx.ToLogContext()
				if logContext["errorCategory"] != string(category) {
					t.Error("Log context should contain error category")
				}

				if logContext["errorSeverity"] != string(severity) {
					t.Error("Log context should contain error severity")
				}
			})
		}
	}
}

// testLogFileSizeValidation validates log file size monitoring and rotation
func testLogFileSizeValidation(t *testing.T) {
	// Clean up any existing logs
	os.RemoveAll("logs")

	os.Setenv("ENV", "production")
	os.Unsetenv("LOG_LEVEL")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Test file size monitoring
	if globalLogger != nil {
		globalLogger.SetMaxFileSize(1024) // 1KB for testing

		currentSize, maxSize, percentUsed := globalLogger.GetFileSizeInfo()
		if maxSize != 1024 {
			t.Errorf("Expected max size 1024, got %d", maxSize)
		}

		if percentUsed < 0 || percentUsed > 100 {
			t.Errorf("Percent used should be 0-100, got %f", percentUsed)
		}

		// Log enough data to test size monitoring
		context := map[string]interface{}{
			"component": "test",
			"data":      strings.Repeat("x", 100),
		}

		for i := 0; i < 20; i++ {
			LogErrorWithContext(context, "Large test message %d with lots of data to increase file size", i)
		}

		// Check if file size increased
		newSize, _, newPercentUsed := globalLogger.GetFileSizeInfo()
		if newSize <= currentSize {
			t.Error("File size should have increased after logging")
		}

		if newPercentUsed <= percentUsed {
			t.Error("Percent used should have increased after logging")
		}

		// Test rotation functionality
		if globalLogger.IsFileSizeExceeded() {
			t.Log("File size exceeded, testing rotation")

			rotationCount, _, _ := globalLogger.GetRotationStats()
			initialRotationCount := rotationCount

			err := globalLogger.ForceRotation()
			if err != nil {
				t.Errorf("Failed to rotate log file: %v", err)
			}

			newRotationCount, lastRotation, timeSinceRotation := globalLogger.GetRotationStats()
			if newRotationCount != initialRotationCount+1 {
				t.Errorf("Expected rotation count to increase by 1, got %d -> %d", initialRotationCount, newRotationCount)
			}

			if lastRotation.IsZero() {
				t.Error("Last rotation time should be set after rotation")
			}

			if timeSinceRotation <= 0 {
				t.Error("Time since rotation should be positive")
			}
		}
	}
}

// testConcurrentLoggingValidation validates concurrent logging behavior
func testConcurrentLoggingValidation(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)
	logger.infoLogger = log.New(io.Discard, "", 0)
	logger.debugLogger = log.New(io.Discard, "", 0)
	logger.warnLogger = log.New(io.Discard, "", 0)
	logger.errorLogger = log.New(io.Discard, "", 0)

	numGoroutines := 20
	messagesPerGoroutine := 50
	var wg sync.WaitGroup

	wg.Add(numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			context := map[string]interface{}{
				"goroutine": goroutineID,
				"component": "concurrent_test",
			}

			for j := 0; j < messagesPerGoroutine; j++ {
				switch j % 4 {
				case 0:
					logger.LogWithContext(DEBUG, context, "Debug message %d from goroutine %d", j, goroutineID)
				case 1:
					logger.LogWithContext(INFO, context, "Info message %d from goroutine %d", j, goroutineID)
				case 2:
					logger.LogWithContext(WARN, context, "Warn message %d from goroutine %d", j, goroutineID)
				case 3:
					logger.LogWithContext(ERROR, context, "Error message %d from goroutine %d", j, goroutineID)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Validate performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

	expectedCount := int64(numGoroutines * messagesPerGoroutine)
	if logCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, logCount)
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

	t.Logf("Concurrent logging validation results:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Messages per goroutine: %d", messagesPerGoroutine)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Messages logged: %d", logCount)
	t.Logf("  Bytes logged: %d", bytesLogged)
	t.Logf("  Logs per second: %.2f", logsPerSecond)
	t.Logf("  Bytes per second: %.2f", bytesPerSecond)

	// Performance should be reasonable
	if logsPerSecond < 100 {
		t.Errorf("Concurrent logging performance too low: %.2f logs/sec", logsPerSecond)
	}
}

// testAllRequirementsValidation validates that all requirements are met
func testAllRequirementsValidation(t *testing.T) {
	requirements := []struct {
		name        string
		requirement string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "Requirement 1.1",
			requirement: "Production mode logs only ERROR, WARN, and critical INFO",
			testFunc: func(t *testing.T) {
				os.Setenv("ENV", "production")
				logger := NewLogger()
				if logger.ShouldLog(DEBUG) || logger.ShouldLog(INFO) {
					t.Error("Production mode should not log DEBUG or INFO by default")
				}
				if !logger.ShouldLog(WARN) || !logger.ShouldLog(ERROR) {
					t.Error("Production mode should log WARN and ERROR")
				}
			},
		},
		{
			name:        "Requirement 1.2",
			requirement: "DEBUG messages completely suppressed in production",
			testFunc: func(t *testing.T) {
				os.Setenv("ENV", "production")
				logger := NewLogger()
				if logger.ShouldLog(DEBUG) {
					t.Error("DEBUG messages should be suppressed in production")
				}
			},
		},
		{
			name:        "Requirement 1.3",
			requirement: "Routine operational messages not logged in production",
			testFunc: func(t *testing.T) {
				// This is validated by the production mode test above
				// where INFO and DEBUG messages are filtered out
				os.Setenv("ENV", "production")
				logger := NewLogger()
				if logger.ShouldLog(INFO) || logger.ShouldLog(DEBUG) {
					t.Error("Routine operational messages should not be logged in production")
				}
			},
		},
		{
			name:        "Requirement 1.4",
			requirement: "Log file size not exceed 10MB per hour in production",
			testFunc: func(t *testing.T) {
				// This is validated by the production log file size test
				// The test extrapolates from a shorter duration to validate the requirement
				t.Log("Log file size requirement validated by production mode test")
			},
		},
		{
			name:        "Requirement 2.1-2.5",
			requirement: "Structured error logging with sufficient context",
			testFunc: func(t *testing.T) {
				// Test that error contexts contain required fields
				wsError := NewWebSocketErrorContext("test", "meet", "ref", "ip")
				logCtx := wsError.ToLogContext()

				requiredFields := []string{"errorCategory", "meetName", "refereeId", "ipAddress"}
				for _, field := range requiredFields {
					if _, exists := logCtx[field]; !exists {
						t.Errorf("Missing required field: %s", field)
					}
				}
			},
		},
		{
			name:        "Requirement 3.1",
			requirement: "Development mode logs all levels including DEBUG",
			testFunc: func(t *testing.T) {
				os.Setenv("ENV", "development")
				logger := NewLogger()
				if !logger.ShouldLog(DEBUG) || !logger.ShouldLog(INFO) ||
					!logger.ShouldLog(WARN) || !logger.ShouldLog(ERROR) {
					t.Error("Development mode should log all levels")
				}
			},
		},
		{
			name:        "Requirement 4.1-4.5",
			requirement: "Environment-based configuration with fallbacks",
			testFunc: func(t *testing.T) {
				// Test various environment configurations
				testEnvs := []struct {
					env      string
					expected LogLevel
				}{
					{"production", WARN},
					{"development", DEBUG},
					{"test", WARN},
					{"invalid", WARN},
					{"", WARN},
				}

				for _, test := range testEnvs {
					if test.env == "" {
						os.Unsetenv("ENV")
					} else {
						os.Setenv("ENV", test.env)
					}

					logger := NewLogger()
					if logger.GetLevel() != test.expected {
						t.Errorf("ENV=%s: expected level %v, got %v", test.env, test.expected, logger.GetLevel())
					}
				}
			},
		},
		{
			name:        "Requirement 5.1-5.5",
			requirement: "Structured logging with timestamp, level, source, and context",
			testFunc: func(t *testing.T) {
				// This is validated by the JSON structure validation
				t.Log("Structured logging requirement validated by JSON structure test")
			},
		},
		{
			name:        "Requirement 6.1-6.5",
			requirement: "Noise reduction - routine operations not logged in production",
			testFunc: func(t *testing.T) {
				os.Setenv("ENV", "production")
				logger := NewLogger()

				// Routine operations should be DEBUG or INFO level, which are filtered in production
				if logger.ShouldLog(DEBUG) || logger.ShouldLog(INFO) {
					t.Error("Routine operations (DEBUG/INFO) should be filtered in production")
				}
			},
		},
	}

	for _, req := range requirements {
		t.Run(req.name, func(t *testing.T) {
			t.Logf("Validating: %s", req.requirement)
			req.testFunc(t)
		})
	}
}

// Helper functions for validation

// TestLogValidator provides reusable validation methods for log testing
type TestLogValidator struct {
	t *testing.T
}

// NewTestLogValidator creates a new log validator for testing
func NewTestLogValidator(t *testing.T) *TestLogValidator {
	return &TestLogValidator{t: t}
}

// validateJSONLogStructure validates that log entries are properly formatted JSON
func validateJSONLogStructure(t *testing.T, content string) {
	validator := NewTestLogValidator(t)
	validator.ValidateJSONStructure(content)
}

// ValidateJSONStructure validates JSON structure and required fields
func (v *TestLogValidator) ValidateJSONStructure(content string) {
	lines := v.extractLogLines(content)
	validJSONCount := 0

	for _, line := range lines {
		if entry := v.parseLogEntry(line); entry != nil {
			validJSONCount++
			v.validateLogEntryFields(*entry)
			v.validateLogLevel(entry.Level)
		}
	}

	if validJSONCount == 0 {
		v.t.Error("No valid JSON log entries found")
	}

	v.t.Logf("Validated %d JSON log entries", validJSONCount)
}

// extractLogLines extracts and cleans log lines from content
func (v *TestLogValidator) extractLogLines(content string) []string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	var validLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, "{") {
			validLines = append(validLines, line)
		}
	}

	return validLines
}

// parseLogEntry attempts to parse a log line as JSON
func (v *TestLogValidator) parseLogEntry(line string) *LogEntry {
	// Extract JSON part (after any prefix)
	jsonStart := strings.Index(line, "{")
	if jsonStart == -1 {
		return nil
	}
	jsonPart := line[jsonStart:]

	var logEntry LogEntry
	if err := json.Unmarshal([]byte(jsonPart), &logEntry); err != nil {
		v.t.Errorf("Invalid JSON in log line: %s, error: %v", line, err)
		return nil
	}

	return &logEntry
}

// validateLogEntryFields checks required fields in log entry
func (v *TestLogValidator) validateLogEntryFields(entry LogEntry) {
	requiredFields := map[string]interface{}{
		"timestamp": entry.Timestamp,
		"level":     entry.Level,
		"message":   entry.Message,
		"source":    entry.Source,
	}

	for fieldName, fieldValue := range requiredFields {
		switch fieldName {
		case "timestamp":
			if entry.Timestamp.IsZero() {
				v.t.Errorf("Log entry missing %s", fieldName)
			}
		case "level", "message", "source":
			if fieldValue == "" {
				v.t.Errorf("Log entry missing %s", fieldName)
			}
		}
	}
}

// validateLogLevel ensures log level is valid
func (v *TestLogValidator) validateLogLevel(level string) {
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for _, validLevel := range validLevels {
		if level == validLevel {
			return
		}
	}
	v.t.Errorf("Invalid log level: %s", level)
}

// validateContextInformation validates that context information is preserved
func validateContextInformation(t *testing.T, content string) {
	expectedContexts := []string{
		"websocket",
		"timer",
		"authentication",
		"position",
		"http",
		"system",
	}

	for _, expectedContext := range expectedContexts {
		if !strings.Contains(content, expectedContext) {
			t.Errorf("Expected context '%s' not found in logs", expectedContext)
		}
	}

	// Validate specific context fields
	contextFields := []string{
		"meetName",
		"refereeId",
		"component",
		"action",
		"ipAddress",
	}

	for _, field := range contextFields {
		if !strings.Contains(content, field) {
			t.Errorf("Expected context field '%s' not found in logs", field)
		}
	}
}

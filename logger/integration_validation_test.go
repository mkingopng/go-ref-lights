package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Integration tests to validate complete logging system functionality

func TestCompleteLoggingSystemIntegration(t *testing.T) {
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

	// Test production mode
	t.Run("ProductionMode", func(t *testing.T) {
		os.Setenv("ENV", "production")
		os.Unsetenv("LOG_LEVEL")

		err := InitLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		defer CloseLogger()

		// Verify production log level
		if !ShouldLog(ERROR) {
			t.Error("ERROR should be logged in production")
		}
		if !ShouldLog(WARN) {
			t.Error("WARN should be logged in production")
		}
		if ShouldLog(INFO) {
			t.Error("INFO should not be logged in production by default")
		}
		if ShouldLog(DEBUG) {
			t.Error("DEBUG should not be logged in production")
		}

		// Test structured logging in production
		context := NewWebSocketErrorContext("Connection timeout", "TestMeet", "left", "192.168.1.100")
		LogErrorWithContext(context.ToLogContext(), "WebSocket connection failed")

		// Verify log file was created
		logFiles, err := filepath.Glob("logs/*.log")
		if err != nil || len(logFiles) == 0 {
			t.Error("Expected log file to be created in production mode")
		}
	})

	// Test development mode
	t.Run("DevelopmentMode", func(t *testing.T) {
		os.Setenv("ENV", "development")
		os.Unsetenv("LOG_LEVEL")

		// Clean up previous logs
		os.RemoveAll("logs")

		err := InitLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		defer CloseLogger()

		// Verify development log level
		if !ShouldLog(ERROR) {
			t.Error("ERROR should be logged in development")
		}
		if !ShouldLog(WARN) {
			t.Error("WARN should be logged in development")
		}
		if !ShouldLog(INFO) {
			t.Error("INFO should be logged in development")
		}
		if !ShouldLog(DEBUG) {
			t.Error("DEBUG should be logged in development")
		}

		// Test all log levels
		LogDebugWithContext(map[string]interface{}{"component": "test"}, "Debug message")
		LogInfoWithContext(map[string]interface{}{"component": "test"}, "Info message")
		LogWarnWithContext(map[string]interface{}{"component": "test"}, "Warn message")
		LogErrorWithContext(map[string]interface{}{"component": "test"}, "Error message")
	})

	// Test LOG_LEVEL override
	t.Run("LogLevelOverride", func(t *testing.T) {
		os.Setenv("ENV", "production")
		os.Setenv("LOG_LEVEL", "DEBUG")

		// Clean up previous logs
		os.RemoveAll("logs")

		err := InitLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		defer CloseLogger()

		// Verify override works
		if !ShouldLog(DEBUG) {
			t.Error("DEBUG should be logged when LOG_LEVEL=DEBUG overrides production")
		}
	})
}

func TestStructuredLoggingValidation(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Save original environment
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	os.Setenv("ENV", "development")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Test different error contexts
	testCases := []struct {
		name           string
		logFunc        func()
		expectedFields []string
	}{
		{
			name: "WebSocketError",
			logFunc: func() {
				ctx := NewWebSocketErrorContext("Connection failed", "TestMeet", "left", "192.168.1.100")
				ctx.LogError()
			},
			expectedFields: []string{"errorCategory", "errorSeverity", "meetName", "refereeId"},
		},
		{
			name: "AuthenticationError",
			logFunc: func() {
				ctx := NewAuthenticationErrorContext("Invalid credentials", "admin", "192.168.1.100", "Mozilla/5.0")
				ctx.LogError()
			},
			expectedFields: []string{"errorCategory", "username", "ipAddress", "userAgent"},
		},
		{
			name: "TimerError",
			logFunc: func() {
				ctx := NewTimerErrorContext("Timer start failed", "TestMeet", "platformReady", 123)
				ctx.LogError()
			},
			expectedFields: []string{"errorCategory", "meetName", "timerType", "timerId"},
		},
		{
			name: "PositionError",
			logFunc: func() {
				ctx := NewPositionErrorContext("Position occupied", "TestMeet", "left", "user123")
				ctx.LogError()
			},
			expectedFields: []string{"errorCategory", "meetName", "position", "userId"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.logFunc()
		})
	}

	// Read and validate log file content
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatal("Expected log file to be created")
	}

	logContent, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(string(logContent), "\n")
	validJSONCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip non-JSON lines (legacy format)
		if !strings.HasPrefix(line, "{") {
			continue
		}

		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			t.Errorf("Invalid JSON in log line: %s, error: %v", line, err)
			continue
		}

		validJSONCount++

		// Validate required fields
		requiredFields := []string{"timestamp", "level", "message", "source"}
		for _, field := range requiredFields {
			if _, exists := logEntry[field]; !exists {
				t.Errorf("Missing required field '%s' in log entry: %s", field, line)
			}
		}

		// Validate timestamp format
		if timestamp, ok := logEntry["timestamp"].(string); ok {
			if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
				t.Errorf("Invalid timestamp format: %s", timestamp)
			}
		}

		// Validate log level
		if level, ok := logEntry["level"].(string); ok {
			validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
			isValid := false
			for _, validLevel := range validLevels {
				if level == validLevel {
					isValid = true
					break
				}
			}
			if !isValid {
				t.Errorf("Invalid log level: %s", level)
			}
		}
	}

	if validJSONCount == 0 {
		t.Error("No valid JSON log entries found")
	}
}

func TestPerformanceRequirements(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Save original environment
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	os.Setenv("ENV", "production")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Test 1: Conditional logging performance
	t.Run("ConditionalLoggingPerformance", func(t *testing.T) {
		expensiveOperationCount := 0
		expensiveOperation := func() string {
			expensiveOperationCount++
			time.Sleep(1 * time.Millisecond)
			return "expensive result"
		}

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			if ShouldLog(DEBUG) {
				LogDebugWithContext(map[string]interface{}{"test": "value"},
					"Debug: %s", expensiveOperation())
			}
		}

		elapsed := time.Since(start)

		// Should be very fast since DEBUG is disabled in production
		if elapsed > 10*time.Millisecond {
			t.Errorf("Conditional logging too slow: %v for %d iterations", elapsed, iterations)
		}

		if expensiveOperationCount > 0 {
			t.Errorf("Expensive operation called %d times, should be 0", expensiveOperationCount)
		}
	})

	// Test 2: Lazy logging performance
	t.Run("LazyLoggingPerformance", func(t *testing.T) {
		lazyFunctionCallCount := 0

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			LogLazyDebug(func() (string, map[string]interface{}) {
				lazyFunctionCallCount++
				time.Sleep(1 * time.Millisecond)
				return "lazy result", map[string]interface{}{"test": "value"}
			})
		}

		elapsed := time.Since(start)

		// Should be very fast since DEBUG is disabled in production
		if elapsed > 10*time.Millisecond {
			t.Errorf("Lazy logging too slow: %v for %d iterations", elapsed, iterations)
		}

		if lazyFunctionCallCount > 0 {
			t.Errorf("Lazy function called %d times, should be 0", lazyFunctionCallCount)
		}
	})

	// Test 3: Production log volume validation
	t.Run("ProductionLogVolume", func(t *testing.T) {
		// Simulate realistic production logging for a short period
		start := time.Now()
		duration := 5 * time.Second

		errorCount := 0
		warnCount := 0

		for time.Since(start) < duration {
			// Simulate occasional errors
			if errorCount < 3 {
				LogErrorWithContext(
					NewWebSocketErrorContext("Connection failed", "TestMeet", "left", "192.168.1.100").ToLogContext(),
					"WebSocket connection failed")
				errorCount++
			}

			// Simulate warnings
			if warnCount < 5 {
				LogWarnWithContext(
					NewAuthenticationContext("login_attempt", "admin", "192.168.1.100"),
					"Authentication attempt")
				warnCount++
			}

			// These should be filtered out in production
			LogDebugWithContext(map[string]interface{}{"component": "test"}, "Debug message")
			LogInfoWithContext(map[string]interface{}{"component": "test"}, "Info message")

			time.Sleep(100 * time.Millisecond)
		}

		// Check log file size
		logFiles, err := filepath.Glob("logs/*.log")
		if err != nil || len(logFiles) == 0 {
			t.Fatal("Expected log file to be created")
		}

		stat, err := os.Stat(logFiles[0])
		if err != nil {
			t.Fatalf("Failed to stat log file: %v", err)
		}

		actualSize := stat.Size()
		actualDuration := time.Since(start)

		// Extrapolate to 1 hour
		bytesPerHour := float64(actualSize) * (float64(time.Hour) / float64(actualDuration))
		mbPerHour := bytesPerHour / (1024 * 1024)

		t.Logf("Production log volume test:")
		t.Logf("  Duration: %v", actualDuration)
		t.Logf("  Log file size: %d bytes", actualSize)
		t.Logf("  Extrapolated MB/hour: %.2f", mbPerHour)

		// Validate against 10MB/hour requirement
		if mbPerHour > 10.0 {
			t.Errorf("Production logging exceeds 10MB/hour: %.2f MB/hour", mbPerHour)
		}

		// Verify some logs were written
		if actualSize == 0 {
			t.Error("No logs written during production test")
		}
	})
}

func TestErrorCategorizationSystem(t *testing.T) {
	// Test all error categories and severities
	testCases := []struct {
		name     string
		category ErrorCategory
		severity ErrorSeverity
		creator  func() *ErrorContext
	}{
		{
			name:     "AuthenticationError",
			category: AuthenticationError,
			severity: SeverityHigh,
			creator: func() *ErrorContext {
				return NewAuthenticationErrorContext("Login failed", "admin", "192.168.1.100", "Mozilla/5.0")
			},
		},
		{
			name:     "WebSocketError",
			category: WebSocketError,
			severity: SeverityMedium,
			creator: func() *ErrorContext {
				return NewWebSocketErrorContext("Connection timeout", "TestMeet", "left", "192.168.1.100")
			},
		},
		{
			name:     "TimerError",
			category: TimerError,
			severity: SeverityMedium,
			creator: func() *ErrorContext {
				return NewTimerErrorContext("Timer start failed", "TestMeet", "platformReady", 123)
			},
		},
		{
			name:     "PositionError",
			category: PositionError,
			severity: SeverityMedium,
			creator: func() *ErrorContext {
				return NewPositionErrorContext("Position occupied", "TestMeet", "left", "user123")
			},
		},
		{
			name:     "ValidationError",
			category: ValidationError,
			severity: SeverityLow,
			creator: func() *ErrorContext {
				return NewValidationErrorContext("Invalid input", "username", "")
			},
		},
		{
			name:     "SystemError",
			category: SystemError,
			severity: SeverityCritical,
			creator: func() *ErrorContext {
				return NewSystemErrorContext("Database connection failed", "database")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errorCtx := tc.creator()

			// Validate error context fields
			if errorCtx.Category != tc.category {
				t.Errorf("Expected category %s, got %s", tc.category, errorCtx.Category)
			}

			if errorCtx.Severity != tc.severity {
				t.Errorf("Expected severity %s, got %s", tc.severity, errorCtx.Severity)
			}

			if errorCtx.Message == "" {
				t.Error("Error message should not be empty")
			}

			if errorCtx.Timestamp.IsZero() {
				t.Error("Timestamp should be set")
			}

			if errorCtx.Source == "" {
				t.Error("Source should be set")
			}

			// Test method chaining
			errorCtx = errorCtx.
				WithCode("ERR001").
				WithDetail("additionalInfo", "test").
				WithContext("operation", "test").
				WithUser("user123", "session456").
				WithRequest("req789", "192.168.1.100", "Mozilla/5.0").
				WithMeet("TestMeet", "left").
				WithStackTrace().
				WithError(fmt.Errorf("underlying error"))

			// Validate chained values
			if errorCtx.Code != "ERR001" {
				t.Error("Code should be set via WithCode")
			}

			if errorCtx.Details["additionalInfo"] != "test" {
				t.Error("Detail should be set via WithDetail")
			}

			if errorCtx.Context["operation"] != "test" {
				t.Error("Context should be set via WithContext")
			}

			if errorCtx.UserID != "user123" {
				t.Error("UserID should be set via WithUser")
			}

			if errorCtx.SessionID != "session456" {
				t.Error("SessionID should be set via WithUser")
			}

			if errorCtx.RequestID != "req789" {
				t.Error("RequestID should be set via WithRequest")
			}

			if errorCtx.IPAddress != "192.168.1.100" {
				t.Error("IPAddress should be set via WithRequest")
			}

			if errorCtx.MeetName != "TestMeet" {
				t.Error("MeetName should be set via WithMeet")
			}

			if errorCtx.RefereeID != "left" {
				t.Error("RefereeID should be set via WithMeet")
			}

			if errorCtx.StackTrace == "" {
				t.Error("StackTrace should be set via WithStackTrace")
			}

			if errorCtx.Details["error"] != "underlying error" {
				t.Error("Error should be set via WithError")
			}

			// Test ToLogContext conversion
			logContext := errorCtx.ToLogContext()
			if logContext["errorCategory"] != string(tc.category) {
				t.Error("Log context should contain error category")
			}

			if logContext["errorSeverity"] != string(tc.severity) {
				t.Error("Log context should contain error severity")
			}

			// Test logging methods (should not panic)
			errorCtx.LogError()
			errorCtx.LogWarn()
		})
	}
}

func TestAllRequirementsValidation(t *testing.T) {
	// This test validates that all requirements from the spec are met

	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	// Save original environment
	originalEnv := os.Getenv("ENV")
	defer func() {
		if originalEnv != "" {
			os.Setenv("ENV", originalEnv)
		} else {
			os.Unsetenv("ENV")
		}
	}()

	// Requirement 1.4: Performance optimization
	t.Run("Requirement_1.4_PerformanceOptimization", func(t *testing.T) {
		os.Setenv("ENV", "production")
		err := InitLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		defer CloseLogger()

		// Test conditional logging avoids expensive operations
		expensiveCallCount := 0
		start := time.Now()

		for i := 0; i < 100; i++ {
			if ShouldLog(DEBUG) {
				expensiveCallCount++
				LogDebugWithContext(map[string]interface{}{}, "Debug message")
			}
		}

		elapsed := time.Since(start)

		if expensiveCallCount > 0 {
			t.Error("Expensive operations should be avoided when logging is disabled")
		}

		if elapsed > 1*time.Millisecond {
			t.Errorf("Conditional logging should be very fast, took %v", elapsed)
		}
	})

	// Requirement 3.1: Lazy evaluation
	t.Run("Requirement_3.1_LazyEvaluation", func(t *testing.T) {
		logger := NewLogger()
		logger.SetLevel(ERROR)

		lazyCallCount := 0
		logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
			lazyCallCount++
			return "Should not be called", map[string]interface{}{}
		})

		if lazyCallCount > 0 {
			t.Error("Lazy function should not be called when logging is disabled")
		}
	})

	// Requirement 3.2: File size monitoring
	t.Run("Requirement_3.2_FileSizeMonitoring", func(t *testing.T) {
		logger := NewLogger()
		logger.SetMaxFileSize(1024)

		_, maxSize, percentUsed := logger.GetFileSizeInfo()
		if maxSize != 1024 {
			t.Errorf("Expected max size 1024, got %d", maxSize)
		}

		if percentUsed < 0 || percentUsed > 100 {
			t.Errorf("Percent used should be 0-100, got %f", percentUsed)
		}

		// Test global functions
		SetGlobalMaxFileSize(2048)
		if globalLogger != nil {
			_, maxSize, _ = globalLogger.GetFileSizeInfo()
			if maxSize != 2048 {
				t.Errorf("Global max size should be 2048, got %d", maxSize)
			}
		}
	})

	// Requirement 3.3: Performance benchmarks
	t.Run("Requirement_3.3_PerformanceBenchmarks", func(t *testing.T) {
		logger := NewLogger()
		logger.SetLevel(DEBUG)
		logger.infoLogger = log.New(io.Discard, "", 0)

		// Test performance stats tracking
		context := map[string]interface{}{"component": "test"}
		for i := 0; i < 10; i++ {
			logger.LogWithContext(INFO, context, "Test message %d", i)
		}

		logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

		if logCount != 10 {
			t.Errorf("Expected 10 logs, got %d", logCount)
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
	})

	// Requirement 3.4: Production mode validation
	t.Run("Requirement_3.4_ProductionModeValidation", func(t *testing.T) {
		os.Setenv("ENV", "production")
		os.RemoveAll("logs")

		err := InitLogger()
		if err != nil {
			t.Fatalf("Failed to initialize logger: %v", err)
		}
		defer CloseLogger()

		// Log various levels
		LogErrorWithContext(map[string]interface{}{"component": "test"}, "Error message")
		LogWarnWithContext(map[string]interface{}{"component": "test"}, "Warn message")
		LogInfoWithContext(map[string]interface{}{"component": "test"}, "Info message")
		LogDebugWithContext(map[string]interface{}{"component": "test"}, "Debug message")

		// Check log file
		logFiles, err := filepath.Glob("logs/*.log")
		if err != nil || len(logFiles) == 0 {
			t.Fatal("Expected log file to be created")
		}

		content, err := os.ReadFile(logFiles[0])
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		logContent := string(content)

		// Should contain ERROR and WARN
		if !strings.Contains(logContent, "Error message") {
			t.Error("Production logs should contain ERROR messages")
		}
		if !strings.Contains(logContent, "Warn message") {
			t.Error("Production logs should contain WARN messages")
		}

		// Should not contain DEBUG (and typically not INFO in production)
		if strings.Contains(logContent, "Debug message") {
			t.Error("Production logs should not contain DEBUG messages")
		}
	})

	// Requirement 3.5: Performance comparison
	t.Run("Requirement_3.5_PerformanceComparison", func(t *testing.T) {
		// This is covered by the benchmark tests and comparative performance test
		// in performance_test.go. Here we just validate the functionality exists.

		logger := NewLogger()
		if logger.lazyEvalEnabled != true {
			t.Error("Lazy evaluation should be enabled by default")
		}

		// Test that performance stats are being tracked
		logCount, bytesLogged, _, _ := logger.GetPerformanceStats()
		if logCount < 0 || bytesLogged < 0 {
			t.Error("Performance stats should be non-negative")
		}
	})
}

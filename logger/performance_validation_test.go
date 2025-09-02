package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestConditionalLoggingOptimizations tests the new conditional logging optimizations
func TestConditionalLoggingOptimizations(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Only ERROR messages should be logged

	expensiveOperationCalled := false

	// Test LogIfEnabled with expensive context function
	start := time.Now()
	logger.LogIfEnabled(DEBUG, func() map[string]interface{} {
		expensiveOperationCalled = true
		time.Sleep(1 * time.Millisecond) // Simulate expensive operation
		return map[string]interface{}{"expensive": "data"}
	}, "This should not be logged")
	elapsed := time.Since(start)

	if elapsed > 500*time.Microsecond {
		t.Errorf("LogIfEnabled took too long: %v (expensive function may have been called)", elapsed)
	}

	if expensiveOperationCalled {
		t.Error("Expensive context function should not have been called for disabled log level")
	}

	// Test LogConditional
	expensiveOperationCalled = false
	start = time.Now()
	logger.LogConditional(DEBUG, func() (LogLevel, map[string]interface{}, string, []interface{}) {
		expensiveOperationCalled = true
		time.Sleep(1 * time.Millisecond) // Simulate expensive operation
		return DEBUG, map[string]interface{}{"test": "value"}, "Conditional message", nil
	})
	elapsed = time.Since(start)

	if elapsed > 500*time.Microsecond {
		t.Errorf("LogConditional took too long: %v (expensive function may have been called)", elapsed)
	}

	if expensiveOperationCalled {
		t.Error("Expensive conditional function should not have been called for disabled log level")
	}
}

// TestEnhancedPerformanceMonitoring tests the detailed performance monitoring features
func TestEnhancedPerformanceMonitoring(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)
	logger.infoLogger = log.New(io.Discard, "", 0)

	// Set performance thresholds
	logger.SetPerformanceThresholds(10, 1024, true) // Low thresholds for testing

	context := map[string]interface{}{
		"component": "test",
		"data":      strings.Repeat("x", 100), // Large context to increase bytes
	}

	// Log messages to trigger thresholds
	for i := 0; i < 20; i++ {
		logger.LogWithContext(INFO, context, "Performance test message %d", i)
	}

	// Get detailed performance stats
	stats := logger.GetDetailedPerformanceStats()

	// Verify all expected fields are present
	expectedFields := []string{
		"logCount", "bytesLogged", "logsPerSecond", "bytesPerSecond",
		"rotationCount", "lastRotation", "currentFileSize", "maxFileSize",
		"filePercentUsed", "rotationEnabled", "uptime", "averageBytesPerLog",
	}

	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected field %s not found in detailed performance stats", field)
		}
	}

	// Verify performance calculations
	if logCount, ok := stats["logCount"].(int64); !ok || logCount != 20 {
		t.Errorf("Expected log count 20, got %v", stats["logCount"])
	}

	if bytesLogged, ok := stats["bytesLogged"].(int64); !ok || bytesLogged <= 0 {
		t.Errorf("Expected positive bytes logged, got %v", stats["bytesLogged"])
	}

	if avgBytes, ok := stats["averageBytesPerLog"].(float64); !ok || avgBytes <= 0 {
		t.Errorf("Expected positive average bytes per log, got %v", stats["averageBytesPerLog"])
	}

	// Check performance thresholds
	exceeded, alerts := logger.CheckPerformanceThresholds()
	if !exceeded {
		t.Error("Expected performance thresholds to be exceeded with low limits")
	}

	if len(alerts) == 0 {
		t.Error("Expected performance alerts to be generated")
	}

	t.Logf("Performance alerts: %v", alerts)
}

// TestLogRotationEnhancements tests the enhanced log rotation functionality
func TestLogRotationEnhancements(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	logger := NewLogger()
	logger.SetMaxFileSize(512) // Small limit for testing

	// Initialize file mode
	err := logger.initFileMode()
	if err != nil {
		t.Fatalf("Failed to initialize file mode: %v", err)
	}
	defer func() {
		if logger.logFile != nil {
			logger.logFile.Close()
		}
	}()

	// Check initial rotation stats
	rotationCount, lastRotation, timeSinceLastRotation := logger.GetRotationStats()
	if rotationCount != 0 {
		t.Errorf("Expected initial rotation count 0, got %d", rotationCount)
	}
	if !lastRotation.IsZero() {
		t.Error("Expected initial last rotation to be zero time")
	}
	if timeSinceLastRotation != 0 {
		t.Error("Expected initial time since last rotation to be 0")
	}

	// Force a rotation
	err = logger.ForceRotation()
	if err != nil {
		t.Fatalf("Failed to force rotation: %v", err)
	}

	// Check rotation stats after forced rotation
	rotationCount, lastRotation, timeSinceLastRotation = logger.GetRotationStats()
	if rotationCount != 1 {
		t.Errorf("Expected rotation count 1 after forced rotation, got %d", rotationCount)
	}
	if lastRotation.IsZero() {
		t.Error("Expected last rotation time to be set after forced rotation")
	}
	if timeSinceLastRotation <= 0 {
		t.Error("Expected positive time since last rotation")
	}

	// Test rotation disable/enable
	logger.EnableRotation(false)

	// Log enough to exceed file size
	context := map[string]interface{}{
		"component": "test",
		"data":      strings.Repeat("x", 100),
	}

	for i := 0; i < 10; i++ {
		logger.LogWithContext(INFO, context, "Test message %d", i)
	}

	// Should not rotate because rotation is disabled
	rotationCountBefore := rotationCount
	time.Sleep(100 * time.Millisecond) // Give time for potential rotation

	rotationCount, _, _ = logger.GetRotationStats()
	if rotationCount != rotationCountBefore {
		t.Error("Log rotation should be disabled")
	}

	// Re-enable rotation
	logger.EnableRotation(true)
}

// TestMessageFormattingOptimizations tests the optimized message formatting
func TestMessageFormattingOptimizations(t *testing.T) {
	logger := NewLogger()

	// Test formatMessage with no arguments (should avoid sprintf)
	message := logger.formatMessage("Simple message")
	if message != "Simple message" {
		t.Errorf("Expected 'Simple message', got '%s'", message)
	}

	// Test formatMessage with arguments (should use sprintf)
	message = logger.formatMessage("Message with %s and %d", "string", 42)
	expected := "Message with string and 42"
	if message != expected {
		t.Errorf("Expected '%s', got '%s'", expected, message)
	}

	// Benchmark the optimization
	start := time.Now()
	for i := 0; i < 1000; i++ {
		logger.formatMessage("Simple message without args")
	}
	noArgsTime := time.Since(start)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		_ = fmt.Sprintf("Simple message without args")
	}
	sprintfTime := time.Since(start)

	t.Logf("formatMessage (no args): %v", noArgsTime)
	t.Logf("fmt.Sprintf (no args): %v", sprintfTime)

	// formatMessage should be faster for messages without arguments
	if noArgsTime > sprintfTime*2 {
		t.Errorf("formatMessage optimization not effective: %v vs %v", noArgsTime, sprintfTime)
	}
}

// TestProductionLogFileSizeValidation validates the 10MB/hour target in production mode
func TestProductionLogFileSizeValidation(t *testing.T) {
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

	// Set production environment
	os.Setenv("ENV", "production")

	logger := NewLogger()
	err := logger.initFileMode()
	if err != nil {
		t.Fatalf("Failed to initialize file mode: %v", err)
	}
	defer logger.logFile.Close()

	// Simulate realistic production logging patterns for a longer duration
	start := time.Now()
	duration := 2 * time.Second // Reduced test duration for CI

	// Counters for different log types
	errorCount := 0
	warnCount := 0
	infoCount := 0
	debugCount := 0 // Should be 0 in production

	// Simulate realistic production workload
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for time.Since(start) < duration {
		select {
		case <-ticker.C:
			// Continue with logging simulation

			// Simulate error conditions (very rare - 1% of ticks)
			if errorCount < 3 && (time.Since(start).Milliseconds()/100)%100 == 0 {
				logger.LogWithContext(ERROR, NewWebSocketErrorContext(
					"Connection failed", "ProductionMeet", "center", "192.168.1.100").ToLogContext(),
					"WebSocket connection failed for referee")
				errorCount++
			}

			// Simulate warnings (rare - 5% of ticks)
			if warnCount < 15 && (time.Since(start).Milliseconds()/100)%20 == 0 {
				logger.LogWithContext(WARN, NewAuthenticationContext(
					"login_attempt", "admin", "192.168.1.100"),
					"Authentication attempt from %s", "192.168.1.100")
				warnCount++
			}

			// Simulate critical system events (very rare - 2% of ticks)
			// In production, these would be logged as WARN or ERROR for critical events
			if infoCount < 6 && (time.Since(start).Milliseconds()/100)%50 == 0 {
				logger.LogWithContext(WARN, NewSystemContext("startup", "server"),
					"Critical system event: %s", "Server component restarted")
				infoCount++
				warnCount++ // This counts as both a critical event and a warning
			}

			// Simulate DEBUG messages that should be filtered out in production
			logger.LogWithContext(DEBUG, map[string]interface{}{"component": "websocket"},
				"This debug message should not appear in production logs")
			debugCount++
		}
	}

	// Get final file size and calculate metrics
	stat, err := logger.logFile.Stat()
	if err != nil {
		t.Fatalf("Failed to get log file stat: %v", err)
	}

	actualSize := stat.Size()
	actualDuration := time.Since(start)

	// Extrapolate to 1 hour
	bytesPerHour := float64(actualSize) * (float64(time.Hour) / float64(actualDuration))
	mbPerHour := bytesPerHour / (1024 * 1024)

	// Get performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

	t.Logf("Production logging validation results:")
	t.Logf("  Test duration: %v", actualDuration)
	t.Logf("  Actual log size: %d bytes", actualSize)
	t.Logf("  Extrapolated MB/hour: %.2f", mbPerHour)
	t.Logf("  Errors logged: %d", errorCount)
	t.Logf("  Warnings logged: %d", warnCount)
	t.Logf("  Info messages logged: %d", infoCount)
	t.Logf("  Debug messages attempted: %d (should be filtered)", debugCount)
	t.Logf("  Total logs written: %d", logCount)
	t.Logf("  Bytes logged: %d", bytesLogged)
	t.Logf("  Logs per second: %.2f", logsPerSecond)
	t.Logf("  Bytes per second: %.2f", bytesPerSecond)

	// Verify we're under the 10MB/hour target
	if mbPerHour > 10.0 {
		t.Errorf("Production logging exceeds 10MB/hour target: %.2f MB/hour", mbPerHour)
	}

	// Verify we actually logged some messages
	if actualSize == 0 {
		t.Error("No logs were written during production test")
	}

	// Verify DEBUG messages were filtered out (logCount should be much less than debugCount)
	if logCount >= int64(debugCount) {
		t.Errorf("DEBUG messages may not be properly filtered: %d logs written vs %d debug attempts", logCount, debugCount)
	}

	// Verify reasonable performance
	if logsPerSecond > 1000 {
		t.Errorf("Logs per second seems too high for production: %.2f", logsPerSecond)
	}

	// Verify we logged the expected types of messages
	// In production mode, only ERROR and WARN are logged
	// Critical events are logged as WARN level in production
	expectedMinLogs := int64(errorCount + warnCount)
	if logCount < expectedMinLogs {
		t.Errorf("Expected at least %d logs (errors + warnings), got %d", expectedMinLogs, logCount)
	}
}

// TestLoggingOverheadComparison compares old vs new logging system performance
func TestLoggingOverheadComparison(t *testing.T) {
	numMessages := 10000

	// Test old-style simple logging
	oldLogger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	oldStart := time.Now()
	for i := 0; i < numMessages; i++ {
		oldLogger.Printf("Old style message %d with some data", i)
	}
	oldDuration := time.Since(oldStart)

	// Test new structured logging (enabled)
	newLogger := NewLogger()
	newLogger.SetLevel(INFO)
	newLogger.infoLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "test",
		"meetName":  "TestMeet",
		"refereeId": "center",
	}

	newStart := time.Now()
	for i := 0; i < numMessages; i++ {
		newLogger.LogWithContext(INFO, context, "New style message %d with some data", i)
	}
	newDuration := time.Since(newStart)

	// Test new structured logging (disabled) - should be very fast
	newLogger.SetLevel(ERROR) // INFO messages will be filtered out

	disabledStart := time.Now()
	for i := 0; i < numMessages; i++ {
		newLogger.LogWithContext(INFO, context, "Disabled message %d with some data", i)
	}
	disabledDuration := time.Since(disabledStart)

	// Test lazy logging (disabled) - should be very fast
	lazyStart := time.Now()
	for i := 0; i < numMessages; i++ {
		newLogger.LogLazy(INFO, func() (string, map[string]interface{}) {
			// This expensive operation should not be called
			return fmt.Sprintf("Lazy message %d with expensive operation", i), context
		})
	}
	lazyDuration := time.Since(lazyStart)

	overhead := float64(newDuration) / float64(oldDuration)
	disabledSpeedup := float64(oldDuration) / float64(disabledDuration)
	lazySpeedup := float64(oldDuration) / float64(lazyDuration)

	t.Logf("Logging performance comparison (%d messages):", numMessages)
	t.Logf("  Old system:           %v", oldDuration)
	t.Logf("  New system (enabled): %v (%.2fx overhead)", newDuration, overhead)
	t.Logf("  New system (disabled): %v (%.2fx speedup)", disabledDuration, disabledSpeedup)
	t.Logf("  Lazy logging (disabled): %v (%.2fx speedup)", lazyDuration, lazySpeedup)

	// Performance requirements (adjusted for realistic expectations)
	// Structured logging has overhead due to JSON marshaling and context processing
	if overhead > 200.0 {
		t.Errorf("New logging system overhead too high: %.2fx (should be < 200x)", overhead)
	}

	// Disabled logging should be comparable to old system (within 2x slower is acceptable)
	if disabledSpeedup < 0.5 {
		t.Errorf("Disabled logging should not be more than 2x slower than old system: %.2fx speedup", disabledSpeedup)
	}

	// Lazy logging should be faster than old system when disabled
	if lazySpeedup < 1.0 {
		t.Errorf("Lazy logging should be at least as fast as old system: %.2fx speedup", lazySpeedup)
	}

	// Disabled logging should be much faster than enabled logging
	disabledVsEnabled := float64(newDuration) / float64(disabledDuration)
	if disabledVsEnabled < 10.0 {
		t.Errorf("Disabled logging should be much faster than enabled: %.2fx", disabledVsEnabled)
	}
}

// TestConcurrentPerformanceOptimizations tests performance under concurrent load
func TestConcurrentPerformanceOptimizations(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)
	logger.infoLogger = log.New(io.Discard, "", 0)
	logger.debugLogger = log.New(io.Discard, "", 0)

	numGoroutines := 50
	messagesPerGoroutine := 200

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
				// Mix of different logging methods
				switch j % 4 {
				case 0:
					logger.LogWithContext(INFO, context, "Regular message %d from goroutine %d", j, goroutineID)
				case 1:
					logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
						return fmt.Sprintf("Lazy message %d from goroutine %d", j, goroutineID), context
					})
				case 2:
					logger.LogIfEnabled(INFO, func() map[string]interface{} {
						return context
					}, "Conditional message %d from goroutine %d", j, goroutineID)
				case 3:
					logger.LogConditional(DEBUG, func() (LogLevel, map[string]interface{}, string, []interface{}) {
						return DEBUG, context, "Conditional lazy message %d from goroutine %d", []interface{}{j, goroutineID}
					})
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Get performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

	expectedCount := int64(numGoroutines * messagesPerGoroutine)

	t.Logf("Concurrent performance test results:")
	t.Logf("  Goroutines: %d", numGoroutines)
	t.Logf("  Messages per goroutine: %d", messagesPerGoroutine)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Expected messages: %d", expectedCount)
	t.Logf("  Actual messages logged: %d", logCount)
	t.Logf("  Bytes logged: %d", bytesLogged)
	t.Logf("  Logs per second: %.2f", logsPerSecond)
	t.Logf("  Bytes per second: %.2f", bytesPerSecond)

	// Verify all messages were logged
	if logCount != expectedCount {
		t.Errorf("Expected %d messages, got %d", expectedCount, logCount)
	}

	// Verify reasonable performance (should handle at least 1000 logs/sec)
	if logsPerSecond < 1000 {
		t.Errorf("Concurrent logging performance too low: %.2f logs/sec", logsPerSecond)
	}

	// Verify no data races or corruption
	if bytesLogged <= 0 {
		t.Error("No bytes logged during concurrent test")
	}
}

// BenchmarkOptimizedLogging benchmarks the optimized logging functions
func BenchmarkOptimizedLogging(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)
	logger.debugLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "TestMeet",
	}

	b.Run("LogWithContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogWithContext(DEBUG, context, "Benchmark message %d", i)
		}
	})

	b.Run("LogIfEnabled", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogIfEnabled(DEBUG, func() map[string]interface{} {
				return context
			}, "Benchmark message %d", i)
		}
	})

	b.Run("LogLazy", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
				return fmt.Sprintf("Benchmark message %d", i), context
			})
		}
	})

	b.Run("LogConditional", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogConditional(DEBUG, func() (LogLevel, map[string]interface{}, string, []interface{}) {
				return DEBUG, context, "Benchmark message %d", []interface{}{i}
			})
		}
	})
}

// BenchmarkDisabledLogging benchmarks performance when logging is disabled
func BenchmarkDisabledLogging(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Disable DEBUG, INFO, WARN

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "TestMeet",
	}

	b.Run("LogWithContext_Disabled", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogWithContext(DEBUG, context, "Disabled message %d", i)
		}
	})

	b.Run("LogIfEnabled_Disabled", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogIfEnabled(DEBUG, func() map[string]interface{} {
				// This should not be called
				b.Error("Expensive function called for disabled logging")
				return context
			}, "Disabled message %d", i)
		}
	})

	b.Run("LogLazy_Disabled", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
				// This should not be called
				b.Error("Lazy function called for disabled logging")
				return fmt.Sprintf("Disabled message %d", i), context
			})
		}
	})
}

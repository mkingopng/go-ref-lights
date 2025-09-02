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

// Benchmark tests for logging performance

func BenchmarkLogger_ShouldLog(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(WARN)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.ShouldLog(DEBUG)
	}
}

func BenchmarkLogger_LogWithContext_Enabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Use discard writer to avoid I/O overhead in benchmark
	logger.debugLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "Test Meet",
		"refereeId": "left",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogWithContext(DEBUG, context, "Benchmark message %d", i)
	}
}

func BenchmarkLogger_LogWithContext_Disabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // DEBUG messages will be filtered out

	context := map[string]interface{}{
		"component": "benchmark",
		"meetName":  "Test Meet",
		"refereeId": "left",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogWithContext(DEBUG, context, "Benchmark message %d", i)
	}
}

func BenchmarkLogger_LazyLogging_Enabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Use discard writer to avoid I/O overhead in benchmark
	logger.debugLogger = log.New(io.Discard, "", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
			return fmt.Sprintf("Benchmark message %d", i), map[string]interface{}{
				"component": "benchmark",
				"iteration": i,
			}
		})
	}
}

func BenchmarkLogger_LazyLogging_Disabled(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // DEBUG messages will be filtered out

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
			// This expensive operation should not be called
			time.Sleep(1 * time.Microsecond)
			return fmt.Sprintf("Benchmark message %d", i), map[string]interface{}{
				"component": "benchmark",
				"iteration": i,
			}
		})
	}
}

func BenchmarkLogger_ConditionalLogging_WithCheck(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // DEBUG messages will be filtered out

	expensiveOperation := func() string {
		// Simulate expensive string formatting
		var result strings.Builder
		for j := 0; j < 100; j++ {
			result.WriteString(fmt.Sprintf("data_%d_", j))
		}
		return result.String()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if logger.ShouldLog(DEBUG) {
			logger.LogWithContext(DEBUG, nil, "Expensive: %s", expensiveOperation())
		}
	}
}

func BenchmarkLogger_ConditionalLogging_WithoutCheck(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // DEBUG messages will be filtered out

	expensiveOperation := func() string {
		// Simulate expensive string formatting
		var result strings.Builder
		for j := 0; j < 100; j++ {
			result.WriteString(fmt.Sprintf("data_%d_", j))
		}
		return result.String()
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will call expensiveOperation even though the log won't be written
		logger.LogWithContext(DEBUG, nil, "Expensive: %s", expensiveOperation())
	}
}

func BenchmarkLogger_ConcurrentLogging(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Use discard writer to avoid I/O overhead in benchmark
	logger.debugLogger = log.New(io.Discard, "", 0)
	logger.infoLogger = log.New(io.Discard, "", 0)
	logger.warnLogger = log.New(io.Discard, "", 0)
	logger.errorLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "benchmark",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			logger.LogWithContext(INFO, context, "Concurrent message %d", i)
			i++
		}
	})
}

func BenchmarkLegacyLogger_Printf(b *testing.B) {
	// Benchmark legacy logger for comparison
	logger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Printf("Legacy message %d", i)
	}
}

// Performance validation tests

func TestPerformanceOptimization_ConditionalLogging(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Only ERROR messages should be logged

	expensiveOperationCalled := false
	expensiveOperation := func() string {
		expensiveOperationCalled = true
		time.Sleep(1 * time.Millisecond) // Simulate expensive operation
		return "expensive result"
	}

	start := time.Now()

	// This should be fast because ShouldLog(DEBUG) returns false immediately
	if logger.ShouldLog(DEBUG) {
		logger.LogWithContext(DEBUG, nil, "Debug message: %s", expensiveOperation())
	}

	elapsed := time.Since(start)

	// Should complete very quickly (much less than 1ms) because expensive operation is not called
	if elapsed > 500*time.Microsecond {
		t.Errorf("Conditional logging took too long: %v (expensive operation may have been called)", elapsed)
	}

	if expensiveOperationCalled {
		t.Error("Expensive operation should not have been called for disabled log level")
	}
}

func TestPerformanceOptimization_LazyEvaluation(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(ERROR) // Only ERROR messages should be logged

	expensiveOperationCalled := false

	start := time.Now()

	// This should be fast because the lazy function is not called when logging is disabled
	logger.LogLazy(DEBUG, func() (string, map[string]interface{}) {
		expensiveOperationCalled = true
		time.Sleep(1 * time.Millisecond) // Simulate expensive operation
		return "expensive result", map[string]interface{}{"test": "value"}
	})

	elapsed := time.Since(start)

	// Should complete very quickly because lazy function is not called
	if elapsed > 500*time.Microsecond {
		t.Errorf("Lazy logging took too long: %v (lazy function may have been called)", elapsed)
	}

	if expensiveOperationCalled {
		t.Error("Lazy function should not have been called for disabled log level")
	}
}

func TestPerformanceStats(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)

	// Use a buffer to capture output and measure bytes
	var buf strings.Builder
	logger.infoLogger = log.New(&buf, "", 0)

	context := map[string]interface{}{
		"component": "test",
	}

	// Log some messages
	for i := 0; i < 10; i++ {
		logger.LogWithContext(INFO, context, "Test message %d", i)
	}

	// Get performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

	if logCount != 10 {
		t.Errorf("Expected log count 10, got %d", logCount)
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

	// Verify the actual output contains our messages
	output := buf.String()
	if !strings.Contains(output, "Test message 0") {
		t.Error("Expected output to contain logged messages")
	}
}

func TestFileSizeMonitoring(t *testing.T) {
	// Create a temporary directory for test logs
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tempDir)

	logger := NewLogger()
	logger.SetMaxFileSize(1024) // 1KB limit for testing

	// Initialize file mode
	err := logger.initFileMode()
	if err != nil {
		t.Fatalf("Failed to initialize file mode: %v", err)
	}
	defer logger.logFile.Close()

	// Check initial file size info
	currentSize, maxSize, percentUsed := logger.GetFileSizeInfo()
	if maxSize != 1024 {
		t.Errorf("Expected max size 1024, got %d", maxSize)
	}
	if currentSize < 0 {
		t.Errorf("Expected non-negative current size, got %d", currentSize)
	}
	if percentUsed < 0 || percentUsed > 100 {
		t.Errorf("Expected percent used between 0-100, got %f", percentUsed)
	}

	// Log enough messages to exceed the file size limit
	context := map[string]interface{}{
		"component": "test",
		"data":      strings.Repeat("x", 100), // Large context to increase log size
	}

	for i := 0; i < 20; i++ {
		logger.LogWithContext(INFO, context, "Large test message %d with lots of data", i)
	}

	// Check if file size is exceeded
	if !logger.IsFileSizeExceeded() {
		// If not exceeded, log more messages
		for i := 20; i < 50; i++ {
			logger.LogWithContext(INFO, context, "Large test message %d with lots of data", i)
		}
	}

	// Give rotation goroutine time to complete if triggered
	time.Sleep(100 * time.Millisecond)
}

func TestLogRotation(t *testing.T) {
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

	// Get initial file name
	initialStat, err := logger.logFile.Stat()
	if err != nil {
		t.Fatalf("Failed to get initial file stat: %v", err)
	}
	initialFileName := initialStat.Name()

	// Log enough to trigger rotation
	context := map[string]interface{}{
		"component": "test",
		"data":      strings.Repeat("x", 100),
	}

	for i := 0; i < 10; i++ {
		logger.LogWithContext(INFO, context, "Test message %d", i)
	}

	// Manually trigger rotation
	err = logger.RotateLogFile()
	if err != nil {
		t.Fatalf("Failed to rotate log file: %v", err)
	}

	// Verify new file was created
	newStat, err := logger.logFile.Stat()
	if err != nil {
		t.Fatalf("Failed to get new file stat: %v", err)
	}
	newFileName := newStat.Name()

	if newFileName == initialFileName {
		t.Error("Log rotation should create a new file with different name")
	}

	// Verify we can still log to the new file
	logger.LogWithContext(INFO, context, "Message after rotation")
}

func TestProductionLogFileSize(t *testing.T) {
	// This test validates that production mode achieves target log file size (10MB/hour max)
	// We'll simulate 1 minute of production logging and extrapolate

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

	// Simulate typical production logging for 1 minute
	// Based on requirements: ERROR, WARN, and critical INFO only
	start := time.Now()
	duration := 10 * time.Second // Shortened for test performance

	// Simulate realistic production log patterns
	errorCount := 0
	warnCount := 0
	infoCount := 0 // Only critical INFO

	for time.Since(start) < duration {
		// Simulate error conditions (rare)
		if errorCount < 2 {
			logger.LogWithContext(ERROR, NewWebSocketErrorContext(
				"Connection failed", "TestMeet", "left", "192.168.1.100").ToLogContext(),
				"WebSocket connection failed")
			errorCount++
		}

		// Simulate warnings (occasional)
		if warnCount < 5 {
			logger.LogWithContext(WARN, NewAuthenticationContext(
				"login_attempt", "admin", "192.168.1.100"),
				"Authentication attempt from %s", "192.168.1.100")
			warnCount++
		}

		// Simulate critical INFO (minimal)
		if infoCount < 3 {
			logger.LogWithContext(INFO, NewSystemContext("startup", "server"),
				"Server started successfully")
			infoCount++
		}

		// Simulate DEBUG messages that should be filtered out in production
		logger.LogWithContext(DEBUG, map[string]interface{}{"component": "websocket"},
			"This debug message should not appear in production logs")

		time.Sleep(100 * time.Millisecond)
	}

	// Get file size
	stat, err := logger.logFile.Stat()
	if err != nil {
		t.Fatalf("Failed to get log file stat: %v", err)
	}

	actualSize := stat.Size()
	actualDuration := time.Since(start)

	// Extrapolate to 1 hour
	bytesPerHour := float64(actualSize) * (float64(time.Hour) / float64(actualDuration))
	mbPerHour := bytesPerHour / (1024 * 1024)

	t.Logf("Production logging test results:")
	t.Logf("  Test duration: %v", actualDuration)
	t.Logf("  Actual log size: %d bytes", actualSize)
	t.Logf("  Extrapolated MB/hour: %.2f", mbPerHour)
	t.Logf("  Errors logged: %d", errorCount)
	t.Logf("  Warnings logged: %d", warnCount)
	t.Logf("  Info messages logged: %d", infoCount)

	// Verify we're under the 10MB/hour target
	if mbPerHour > 10.0 {
		t.Errorf("Production logging exceeds 10MB/hour target: %.2f MB/hour", mbPerHour)
	}

	// Verify we actually logged some messages
	if actualSize == 0 {
		t.Error("No logs were written during production test")
	}
}

// Comparative performance test: old vs new logging system
func TestLoggingSystemComparison(t *testing.T) {
	// Test old-style logging (simple printf)
	oldLogger := log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	oldStart := time.Now()
	for i := 0; i < 1000; i++ {
		oldLogger.Printf("Old style message %d", i)
	}
	oldDuration := time.Since(oldStart)

	// Test new structured logging
	newLogger := NewLogger()
	newLogger.SetLevel(INFO)
	newLogger.infoLogger = log.New(io.Discard, "", 0)

	context := map[string]interface{}{
		"component": "test",
		"meetName":  "TestMeet",
	}

	newStart := time.Now()
	for i := 0; i < 1000; i++ {
		newLogger.LogWithContext(INFO, context, "New style message %d", i)
	}
	newDuration := time.Since(newStart)

	t.Logf("Logging performance comparison:")
	t.Logf("  Old system: %v for 1000 messages", oldDuration)
	t.Logf("  New system: %v for 1000 messages", newDuration)
	t.Logf("  Overhead: %.2fx", float64(newDuration)/float64(oldDuration))

	// New system should not be more than 5x slower due to structured logging overhead
	if newDuration > oldDuration*5 {
		t.Errorf("New logging system is too slow: %v vs %v (%.2fx overhead)",
			newDuration, oldDuration, float64(newDuration)/float64(oldDuration))
	}
}

func TestConcurrentPerformanceStats(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(DEBUG)
	logger.infoLogger = log.New(io.Discard, "", 0)

	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 100

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			context := map[string]interface{}{
				"goroutine": goroutineID,
			}
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.LogWithContext(INFO, context, "Message %d from goroutine %d", j, goroutineID)
			}
		}(i)
	}

	wg.Wait()

	// Check performance stats
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := logger.GetPerformanceStats()

	expectedCount := int64(numGoroutines * messagesPerGoroutine)
	if logCount != expectedCount {
		t.Errorf("Expected log count %d, got %d", expectedCount, logCount)
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

	t.Logf("Concurrent logging stats:")
	t.Logf("  Total logs: %d", logCount)
	t.Logf("  Total bytes: %d", bytesLogged)
	t.Logf("  Logs/second: %.2f", logsPerSecond)
	t.Logf("  Bytes/second: %.2f", bytesPerSecond)
}

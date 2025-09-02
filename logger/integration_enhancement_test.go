//go:build integration
// +build integration

package logger

import (
	"os"
	"testing"
	"time"
)

// TestConfigValidationIntegration tests the configuration validation system
func TestConfigValidationIntegration(t *testing.T) {
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

	t.Run("ValidConfiguration", func(t *testing.T) {
		os.Setenv("ENV", "development")
		os.Setenv("LOG_LEVEL", "DEBUG")

		err := ValidateGlobalConfig()
		if err != nil {
			t.Errorf("Valid configuration should not return error: %v", err)
		}
	})

	t.Run("InvalidEnvironment", func(t *testing.T) {
		os.Setenv("ENV", "invalid_env")
		os.Unsetenv("LOG_LEVEL")

		// Should not error but should log warnings
		err := ValidateGlobalConfig()
		if err != nil {
			t.Errorf("Invalid environment should not cause error, just warnings: %v", err)
		}
	})

	t.Run("InvalidLogLevel", func(t *testing.T) {
		os.Setenv("ENV", "production")
		os.Setenv("LOG_LEVEL", "INVALID")

		err := ValidateGlobalConfig()
		if err != nil {
			t.Errorf("Invalid log level should not cause error, just warnings: %v", err)
		}
	})
}

// TestPerformanceMonitoringIntegration tests the performance monitoring system
func TestPerformanceMonitoringIntegration(t *testing.T) {
	// Clean up any existing logs
	os.RemoveAll("logs")

	// Set up test environment
	os.Setenv("ENV", "development")
	os.Unsetenv("LOG_LEVEL")

	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Create performance monitor
	monitor := NewPerformanceMonitor(globalLogger)

	// Set low thresholds for testing
	monitor.SetThresholds(AlertThresholds{
		MaxLogsPerSecond:  10,
		MaxBytesPerSecond: 1024,
		MaxFileSize:       1024,
	})

	// Generate some logs to trigger monitoring
	context := map[string]interface{}{"component": "test"}
	for i := 0; i < 20; i++ {
		LogInfoWithContext(context, "Performance test message %d", i)
	}

	// Check performance
	report := monitor.CheckPerformance()

	if report == nil {
		t.Fatal("Performance report should not be nil")
	}

	if report.TotalLogsWritten == 0 {
		t.Error("Expected some logs to be written")
	}

	if report.TotalBytesWritten == 0 {
		t.Error("Expected some bytes to be written")
	}

	t.Logf("Performance report: %+v", report)
}

// TestContinuousPerformanceMonitoring tests continuous monitoring
func TestContinuousPerformanceMonitoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping continuous monitoring test in short mode")
	}

	// Clean up any existing logs
	os.RemoveAll("logs")

	os.Setenv("ENV", "development")
	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	monitor := NewPerformanceMonitor(globalLogger)

	// Start monitoring with short interval
	reports := monitor.StartMonitoring(100 * time.Millisecond)

	// Generate logs for a short period
	done := make(chan bool)
	go func() {
		context := map[string]interface{}{"component": "continuous_test"}
		for i := 0; i < 10; i++ {
			LogInfoWithContext(context, "Continuous test message %d", i)
			time.Sleep(50 * time.Millisecond)
		}
		done <- true
	}()

	// Collect reports for a short time
	reportCount := 0
	timeout := time.After(2 * time.Second)

	for {
		select {
		case report := <-reports:
			reportCount++
			if report.TotalLogsWritten > 0 {
				t.Logf("Received performance report: logs=%d, bytes=%d",
					report.TotalLogsWritten, report.TotalBytesWritten)
			}
		case <-done:
			if reportCount == 0 {
				t.Error("Expected to receive at least one performance report")
			}
			return
		case <-timeout:
			t.Error("Test timed out waiting for performance reports")
			return
		}
	}
}

// TestEnhancedErrorContextIntegration tests the enhanced error context system
func TestEnhancedErrorContextIntegration(t *testing.T) {
	os.Setenv("ENV", "development")
	err := InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer CloseLogger()

	// Test WebSocket error context
	wsError := NewWebSocketErrorContext("Connection timeout", "TestMeet", "left", "192.168.1.100").
		WithCode("WS_TEST_001").
		WithDetail("connectionDuration", "30s").
		WithDetail("lastPingTime", time.Now())

	wsError.LogError()

	// Test Authentication error context
	authError := NewAuthenticationErrorContext("Invalid credentials", "testuser", "192.168.1.100", "TestAgent").
		WithCode("AUTH_TEST_001").
		WithDetail("attemptCount", 3).
		WithDetail("lockoutTime", time.Now().Add(5*time.Minute))

	authError.LogWarn()

	// Test method chaining
	chainedError := NewErrorContext(NetworkError, SeverityHigh, "Network connection failed").
		WithCode("NET_001").
		WithDetail("endpoint", "api.example.com").
		WithDetail("timeout", "30s").
		WithContext("retryCount", 3).
		WithUser("testuser", "session123").
		WithRequest("req123", "192.168.1.100", "TestAgent").
		WithMeet("TestMeet", "left")

	logContext := chainedError.ToLogContext()

	// Verify all fields are present
	expectedFields := []string{
		"errorCategory", "errorSeverity", "errorCode", "endpoint", "timeout",
		"retryCount", "userId", "sessionId", "requestId", "ipAddress",
		"userAgent", "meetName", "refereeId",
	}

	for _, field := range expectedFields {
		if _, exists := logContext[field]; !exists {
			t.Errorf("Expected field %s not found in log context", field)
		}
	}

	chainedError.LogError()

	t.Log("Enhanced error context integration test completed successfully")
}

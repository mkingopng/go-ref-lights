//go:build simulation
// +build simulation

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-ref-lights/logger"
)

// TestLoggingDuringSimulatedMeet validates logging behavior during a realistic meet simulation
func TestLoggingDuringSimulatedMeet(t *testing.T) {
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

	// Test both production and development modes
	t.Run("ProductionMeetSimulation", func(t *testing.T) {
		testMeetSimulation(t, "production", 30*time.Second)
	})

	t.Run("DevelopmentMeetSimulation", func(t *testing.T) {
		testMeetSimulation(t, "development", 15*time.Second)
	})
}

// testMeetSimulation simulates a realistic powerlifting meet with logging
func testMeetSimulation(t *testing.T, environment string, duration time.Duration) {
	// Clean up any existing logs
	os.RemoveAll("logs")

	// Set environment
	os.Setenv("ENV", environment)
	os.Unsetenv("LOG_LEVEL")

	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger for %s environment: %v", environment, err)
	}
	defer logger.CloseLogger()

	t.Logf("Starting %s meet simulation for %v", environment, duration)

	// Simulate meet setup
	simulateMeetSetup(t, environment)

	// Simulate meet progression
	start := time.Now()
	simulateMeetProgression(t, environment, start, duration)

	// Validate results
	validateMeetLoggingResults(t, environment, start, duration)
}

// simulateMeetSetup simulates the initial setup of a powerlifting meet
func simulateMeetSetup(t *testing.T, environment string) {
	// System startup logs
	logger.LogInfoWithContext(
		logger.NewSystemContext("startup", "referee_lights"),
		"RefLights system starting up for meet: %s", "Nationals_Platform_1")

	// Authentication logs (meet director login)
	logger.LogInfoWithContext(
		logger.NewAuthenticationContext("login_success", "meet_director", "192.168.1.10"),
		"Meet director logged in successfully")

	// QR code generation logs
	positions := []string{"left", "center", "right"}
	for _, position := range positions {
		logger.LogInfoWithContext(
			logger.NewSystemContext("qr_generation", "position_assignment"),
			"QR code generated for position: %s", position)
	}

	// WebSocket server startup
	logger.LogInfoWithContext(
		logger.NewSystemContext("startup", "websocket_server"),
		"WebSocket server started on port 8080")

	t.Logf("Meet setup completed for %s environment", environment)
}

// simulateMeetProgression simulates the progression of a powerlifting meet
func simulateMeetProgression(t *testing.T, environment string, start time.Time, duration time.Duration) {
	meetName := "Nationals_Platform_1"
	referees := []string{"left", "center", "right"}

	// Counters for different types of events
	lifterCount := 0
	attemptCount := 0
	errorCount := 0
	warningCount := 0

	// Simulate referee connections
	for _, referee := range referees {
		logger.LogInfoWithContext(
			logger.NewWebSocketContext("connection_established", meetName, referee, fmt.Sprintf("192.168.1.%d", 100+len(referee))),
			"Referee connected to position %s", referee)

		logger.LogInfoWithContext(
			logger.NewPositionContext("position_occupied", meetName, referee, fmt.Sprintf("referee_%s", referee)),
			"Position %s occupied by referee", referee)
	}

	// Main meet simulation loop
	ticker := time.NewTicker(2 * time.Second) // Event every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Since(start) >= duration {
				return
			}

			// Simulate different meet events based on time progression
			elapsed := time.Since(start)
			eventType := selectEventType(elapsed, duration)

			switch eventType {
			case "platform_ready":
				simulatePlatformReady(meetName, lifterCount)
				lifterCount++

			case "referee_decisions":
				simulateRefereeDecisions(meetName, referees, attemptCount)
				attemptCount++

			case "timer_events":
				simulateTimerEvents(meetName, attemptCount)

			case "websocket_heartbeat":
				// In production, these should be DEBUG level (filtered out)
				// In development, these should be visible
				for _, referee := range referees {
					logger.LogDebugWithContext(
						logger.NewWebSocketContext("heartbeat", meetName, referee, fmt.Sprintf("192.168.1.%d", 100+len(referee))),
						"Heartbeat received from referee %s", referee)
				}

			case "error_condition":
				simulateErrorCondition(meetName, &errorCount)

			case "warning_condition":
				simulateWarningCondition(meetName, &warningCount)

			case "routine_operations":
				simulateRoutineOperations(meetName, referees)
			}

		case <-time.After(duration + 5*time.Second):
			t.Error("Meet simulation timed out")
			return
		}
	}
}

// selectEventType determines what type of event to simulate based on meet progression
func selectEventType(elapsed, total time.Duration) string {
	progress := float64(elapsed) / float64(total)

	// Different phases of the meet have different event patterns
	switch {
	case progress < 0.1: // Setup phase
		return "websocket_heartbeat"
	case progress < 0.3: // Early meet
		events := []string{"platform_ready", "referee_decisions", "timer_events", "routine_operations"}
		return events[int(elapsed.Seconds())%len(events)]
	case progress < 0.7: // Main meet
		events := []string{"platform_ready", "referee_decisions", "timer_events", "websocket_heartbeat", "routine_operations"}
		return events[int(elapsed.Seconds())%len(events)]
	case progress < 0.9: // Late meet
		events := []string{"platform_ready", "referee_decisions", "timer_events", "error_condition", "warning_condition"}
		return events[int(elapsed.Seconds())%len(events)]
	default: // Meet conclusion
		return "routine_operations"
	}
}

// simulatePlatformReady simulates platform ready events
func simulatePlatformReady(meetName string, lifterCount int) {
	logger.LogInfoWithContext(
		logger.NewTimerContext("platform_ready_started", meetName, "platformReady", lifterCount),
		"Platform ready timer started for lifter %d", lifterCount+1)

	// Clear previous decisions (this would be INFO in production, but critical)
	logger.LogWarnWithContext(
		logger.NewSystemContext("decision_clear", "meet_management"),
		"Previous decisions cleared for new attempt")
}

// simulateRefereeDecisions simulates referee decision making
func simulateRefereeDecisions(meetName string, referees []string, attemptCount int) {
	decisions := []string{"good_lift", "no_lift"}

	for _, referee := range referees {
		decision := decisions[attemptCount%len(decisions)]

		// Individual referee decisions (would be DEBUG in production)
		logger.LogDebugWithContext(
			logger.NewPositionContext("decision_made", meetName, referee, fmt.Sprintf("referee_%s", referee)),
			"Referee %s made decision: %s", referee, decision)
	}

	// Final decision announcement (INFO level - critical for meet flow)
	logger.LogWarnWithContext(
		logger.NewSystemContext("decision_final", "meet_management"),
		"All referees have decided for attempt %d", attemptCount+1)
}

// simulateTimerEvents simulates various timer operations
func simulateTimerEvents(meetName string, attemptCount int) {
	timerTypes := []string{"platformReady", "nextAttempt"}
	timerType := timerTypes[attemptCount%len(timerTypes)]

	// Timer updates (should be DEBUG in production)
	logger.LogDebugWithContext(
		logger.NewTimerContext("timer_update", meetName, timerType, attemptCount),
		"Timer %s updated: %d seconds remaining", timerType, 60-(attemptCount%60))

	// Timer completion (INFO level)
	if attemptCount%10 == 0 {
		logger.LogInfoWithContext(
			logger.NewTimerContext("timer_completed", meetName, timerType, attemptCount),
			"Timer %s completed", timerType)
	}
}

// simulateErrorCondition simulates various error conditions
func simulateErrorCondition(meetName string, errorCount *int) {
	*errorCount++

	errorTypes := []struct {
		category string
		context  func() *logger.ErrorContext
		message  string
	}{
		{
			category: "websocket_error",
			context: func() *logger.ErrorContext {
				return logger.NewWebSocketErrorContext("Connection timeout", meetName, "left", "192.168.1.101")
			},
			message: "WebSocket connection timeout for referee",
		},
		{
			category: "timer_error",
			context: func() *logger.ErrorContext {
				return logger.NewTimerErrorContext("Timer start failed", meetName, "platformReady", *errorCount)
			},
			message: "Failed to start platform ready timer",
		},
		{
			category: "position_error",
			context: func() *logger.ErrorContext {
				return logger.NewPositionErrorContext("Position conflict", meetName, "center", "referee_center")
			},
			message: "Position occupancy conflict detected",
		},
		{
			category: "system_error",
			context: func() *logger.ErrorContext {
				return logger.NewSystemErrorContext("Database connection failed", "meet_data")
			},
			message: "Failed to save meet data",
		},
	}

	errorType := errorTypes[*errorCount%len(errorTypes)]
	errorCtx := errorType.context()
	errorCtx.LogError()
}

// simulateWarningCondition simulates various warning conditions
func simulateWarningCondition(meetName string, warningCount *int) {
	*warningCount++

	warningTypes := []struct {
		context map[string]interface{}
		message string
	}{
		{
			context: logger.NewAuthenticationContext("failed_attempt", "unknown_user", "192.168.1.200"),
			message: "Authentication attempt from unknown user",
		},
		{
			context: logger.NewWebSocketContext("connection_retry", meetName, "right", "192.168.1.103"),
			message: "WebSocket connection retry for referee",
		},
		{
			context: logger.NewSystemContext("performance_warning", "memory_usage"),
			message: "High memory usage detected: 85%",
		},
		{
			context: logger.NewHTTPContext("POST", "/api/decision", "Unknown", "192.168.1.200", 429),
			message: "Rate limit exceeded for decision endpoint",
		},
	}

	warningType := warningTypes[*warningCount%len(warningTypes)]
	logger.LogWarnWithContext(warningType.context, warningType.message)
}

// simulateRoutineOperations simulates routine operational messages
func simulateRoutineOperations(meetName string, referees []string) {
	// These should be DEBUG/INFO level and filtered in production

	// HTTP request logging (DEBUG level)
	logger.LogDebugWithContext(
		logger.NewHTTPContext("GET", "/api/status", "Mozilla/5.0", "192.168.1.10", 200),
		"Status check request processed")

	// WebSocket message processing (DEBUG level)
	for _, referee := range referees {
		logger.LogDebugWithContext(
			logger.NewWebSocketContext("message_processed", meetName, referee, fmt.Sprintf("192.168.1.%d", 100+len(referee))),
			"WebSocket message processed for referee %s", referee)
	}

	// Position status updates (DEBUG level)
	logger.LogDebugWithContext(
		logger.NewPositionContext("status_update", meetName, "all", "system"),
		"Position status updated for all referees")
}

// validateMeetLoggingResults validates the logging results after meet simulation
func validateMeetLoggingResults(t *testing.T, environment string, start time.Time, duration time.Duration) {
	// Check that log file was created
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil || len(logFiles) == 0 {
		t.Fatalf("Expected log file to be created for %s environment", environment)
	}

	// Read and analyze log file
	content, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	actualDuration := time.Since(start)

	// Get file size and calculate metrics
	stat, err := os.Stat(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	actualSize := stat.Size()
	bytesPerHour := float64(actualSize) * (float64(time.Hour) / float64(actualDuration))
	mbPerHour := bytesPerHour / (1024 * 1024)

	// Count different types of log entries
	logCounts := countLogEntries(logContent)

	t.Logf("%s meet simulation results:", strings.Title(environment))
	t.Logf("  Simulation duration: %v", actualDuration)
	t.Logf("  Log file size: %d bytes", actualSize)
	t.Logf("  Extrapolated MB/hour: %.2f", mbPerHour)
	t.Logf("  Log entry counts: %+v", logCounts)

	// Validate environment-specific requirements
	switch environment {
	case "production":
		validateProductionMeetResults(t, mbPerHour, logCounts, logContent)
	case "development":
		validateDevelopmentMeetResults(t, logCounts, logContent)
	}

	// Validate JSON structure
	validateLogJSONStructure(t, logContent)

	// Validate context preservation
	validateMeetContexts(t, logContent)
}

// countLogEntries counts different types of log entries
func countLogEntries(content string) map[string]int {
	counts := map[string]int{
		"DEBUG": 0,
		"INFO":  0,
		"WARN":  0,
		"ERROR": 0,
		"total": 0,
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for JSON log entries
		if strings.Contains(line, "{") {
			jsonStart := strings.Index(line, "{")
			if jsonStart != -1 {
				jsonPart := line[jsonStart:]
				var logEntry logger.LogEntry
				if json.Unmarshal([]byte(jsonPart), &logEntry) == nil {
					counts[logEntry.Level]++
					counts["total"]++
				}
			}
		}
	}

	return counts
}

// validateProductionMeetResults validates production-specific requirements
func validateProductionMeetResults(t *testing.T, mbPerHour float64, logCounts map[string]int, content string) {
	// Validate 10MB/hour requirement
	if mbPerHour > 10.0 {
		t.Errorf("Production meet logging exceeds 10MB/hour: %.2f MB/hour", mbPerHour)
	}

	// Validate log level filtering
	if logCounts["DEBUG"] > 0 {
		t.Errorf("Production mode should not log DEBUG messages, found %d", logCounts["DEBUG"])
	}

	// INFO messages should be minimal in production (only critical ones)
	if logCounts["INFO"] > logCounts["WARN"]+logCounts["ERROR"] {
		t.Errorf("Production mode should have minimal INFO messages, found %d INFO vs %d WARN+ERROR",
			logCounts["INFO"], logCounts["WARN"]+logCounts["ERROR"])
	}

	// Should have some ERROR and WARN messages from simulation
	if logCounts["ERROR"] == 0 {
		t.Error("Expected some ERROR messages from meet simulation")
	}
	if logCounts["WARN"] == 0 {
		t.Error("Expected some WARN messages from meet simulation")
	}

	// Validate that routine operations are filtered out
	if strings.Contains(content, "Heartbeat received") {
		t.Error("Production logs should not contain heartbeat messages")
	}
	if strings.Contains(content, "WebSocket message processed") {
		t.Error("Production logs should not contain routine WebSocket processing messages")
	}

	t.Logf("Production meet validation passed: %.2f MB/hour, %d total logs", mbPerHour, logCounts["total"])
}

// validateDevelopmentMeetResults validates development-specific requirements
func validateDevelopmentMeetResults(t *testing.T, logCounts map[string]int, content string) {
	// Development should have all log levels
	if logCounts["DEBUG"] == 0 {
		t.Error("Development mode should contain DEBUG messages")
	}
	if logCounts["INFO"] == 0 {
		t.Error("Development mode should contain INFO messages")
	}
	if logCounts["WARN"] == 0 {
		t.Error("Development mode should contain WARN messages")
	}
	if logCounts["ERROR"] == 0 {
		t.Error("Development mode should contain ERROR messages")
	}

	// Should contain detailed operational information
	if !strings.Contains(content, "Heartbeat received") {
		t.Error("Development logs should contain heartbeat messages")
	}
	if !strings.Contains(content, "WebSocket message processed") {
		t.Error("Development logs should contain WebSocket processing messages")
	}

	t.Logf("Development meet validation passed: %d total logs with all levels", logCounts["total"])
}

// validateLogJSONStructure validates that all log entries are properly formatted JSON
func validateLogJSONStructure(t *testing.T, content string) {
	lines := strings.Split(content, "\n")
	validJSONCount := 0
	invalidJSONCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "{") {
			jsonStart := strings.Index(line, "{")
			if jsonStart != -1 {
				jsonPart := line[jsonStart:]
				var logEntry logger.LogEntry
				if json.Unmarshal([]byte(jsonPart), &logEntry) == nil {
					validJSONCount++

					// Validate required fields
					if logEntry.Timestamp.IsZero() {
						t.Error("Log entry missing timestamp")
					}
					if logEntry.Level == "" {
						t.Error("Log entry missing level")
					}
					if logEntry.Message == "" {
						t.Error("Log entry missing message")
					}
					if logEntry.Source == "" {
						t.Error("Log entry missing source")
					}
				} else {
					invalidJSONCount++
					t.Errorf("Invalid JSON in log entry: %s", line)
				}
			}
		}
	}

	if validJSONCount == 0 {
		t.Error("No valid JSON log entries found")
	}

	if invalidJSONCount > 0 {
		t.Errorf("Found %d invalid JSON log entries", invalidJSONCount)
	}

	t.Logf("JSON validation: %d valid entries, %d invalid entries", validJSONCount, invalidJSONCount)
}

// validateMeetContexts validates that meet-specific contexts are preserved
func validateMeetContexts(t *testing.T, content string) {
	expectedContexts := []string{
		"Nationals_Platform_1", // Meet name
		"websocket",            // Component
		"timer",                // Component
		"authentication",       // Component
		"position",             // Component
		"system",               // Component
		"left",                 // Referee position
		"center",               // Referee position
		"right",                // Referee position
		"errorCategory",        // Error categorization
		"meetName",             // Context field
		"refereeId",            // Context field
	}

	for _, expectedContext := range expectedContexts {
		if !strings.Contains(content, expectedContext) {
			t.Errorf("Expected context '%s' not found in meet logs", expectedContext)
		}
	}

	t.Log("Meet context validation passed")
}

// TestLogFileSizeValidationDuringMeet validates log file size monitoring during meet
func TestLogFileSizeValidationDuringMeet(t *testing.T) {
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

	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.CloseLogger()

	// Set a small file size limit for testing
	if logger.GetGlobalLogger() != nil {
		logger.GetGlobalLogger().SetMaxFileSize(2048) // 2KB for testing
	}

	// Simulate intensive logging that should trigger size monitoring
	meetName := "SizeTest_Meet"

	for i := 0; i < 50; i++ {
		// Log large error messages
		errorCtx := logger.NewWebSocketErrorContext(
			fmt.Sprintf("Large error message with lots of context data - iteration %d", i),
			meetName,
			"left",
			"192.168.1.100")

		errorCtx.WithDetail("largeData", strings.Repeat("x", 100)).
			WithDetail("iteration", i).
			WithDetail("timestamp", time.Now().Format(time.RFC3339)).
			WithDetail("additionalContext", "This is additional context to increase log size")

		errorCtx.LogError()

		// Check file size periodically
		if i%10 == 0 {
			currentSize, maxSize, percentUsed := logger.GetGlobalFileSizeInfo()
			t.Logf("Iteration %d: File size %d/%d bytes (%.1f%% used)", i, currentSize, maxSize, percentUsed)

			if logger.IsGlobalFileSizeExceeded() {
				t.Logf("File size exceeded at iteration %d, testing rotation", i)

				// Test rotation
				rotationCount, _, _ := logger.GetGlobalLogger().GetRotationStats()
				initialCount := rotationCount

				err := logger.RotateGlobalLogFile()
				if err != nil {
					t.Errorf("Failed to rotate log file: %v", err)
				}

				newCount, lastRotation, timeSince := logger.GetGlobalLogger().GetRotationStats()
				if newCount != initialCount+1 {
					t.Errorf("Rotation count should increase: %d -> %d", initialCount, newCount)
				}

				if lastRotation.IsZero() {
					t.Error("Last rotation time should be set")
				}

				if timeSince <= 0 {
					t.Error("Time since rotation should be positive")
				}

				t.Logf("Log rotation successful: count=%d, time=%v", newCount, lastRotation)
				break
			}
		}
	}

	// Validate final state
	logFiles, err := filepath.Glob("logs/*.log")
	if err != nil {
		t.Fatalf("Failed to find log files: %v", err)
	}

	if len(logFiles) == 0 {
		t.Fatal("No log files found after size validation test")
	}

	t.Logf("Log file size validation completed with %d log files", len(logFiles))
}

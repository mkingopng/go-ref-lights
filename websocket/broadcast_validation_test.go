package websocket

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"go-ref-lights/logger"
)

// Common test data for validation tests
var broadcastMessageValidationTests = []struct {
	name      string
	meetName  string
	shouldLog bool
	logText   string
}{
	{
		name:      "valid meetName",
		meetName:  "test-meet",
		shouldLog: false,
	},
	{
		name:      "empty meetName",
		meetName:  "",
		shouldLog: true,
		logText:   "[BroadcastMessage] meetName is empty",
	},
	{
		name:      "meetName too long",
		meetName:  strings.Repeat("a", 101),
		shouldLog: true,
		logText:   "[BroadcastMessage] meetName too long",
	},
	{
		name:      "meetName with special characters",
		meetName:  "test-meet-@#$%",
		shouldLog: false,
	},
	{
		name:      "meetName with unicode",
		meetName:  "test-meet-ñ",
		shouldLog: false,
	},
	{
		name:      "meetName at boundary length",
		meetName:  strings.Repeat("a", 100),
		shouldLog: false,
	},
}

var broadcastFinalResultsValidationTests = []struct {
	name      string
	meetName  string
	shouldLog bool
	logText   string
}{
	{
		name:      "empty meetName",
		meetName:  "",
		shouldLog: true,
		logText:   "[broadcastFinalResults] meetName is empty",
	},
	{
		name:      "valid meetName",
		meetName:  "test-meet",
		shouldLog: false,
	},
	{
		name:      "meetName with special characters",
		meetName:  "test-meet-@#$%",
		shouldLog: false,
	},
	{
		name:      "meetName with unicode",
		meetName:  "test-meet-ñ",
		shouldLog: false,
	},
}

// setupBroadcastTest sets up common test environment for broadcast validation tests
func setupBroadcastTest(t *testing.T) (*bytes.Buffer, func()) {
	var buf bytes.Buffer
	logger.Error = log.New(&buf, "", 0)
	logger.Debug = log.New(&buf, "", 0)

	// Initialize broadcast channel if not already done
	if broadcast == nil {
		broadcast = make(chan []byte, 100)
	}

	cleanup := func() {
		// Reset loggers if needed for other tests
		// This ensures test isolation
	}

	return &buf, cleanup
}

func TestBroadcastMessageValidation(t *testing.T) {
	buf, cleanup := setupBroadcastTest(t)
	defer cleanup()

	for _, tt := range broadcastMessageValidationTests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			message := map[string]interface{}{
				"action": "test",
			}

			BroadcastMessage(tt.meetName, message)

			logOutput := buf.String()

			if tt.shouldLog {
				if !strings.Contains(logOutput, tt.logText) {
					t.Errorf("Expected log to contain %q, got %q", tt.logText, logOutput)
				}
			} else {
				// For valid meetName, we should see debug log but no error
				if strings.Contains(logOutput, "meetName is empty") || strings.Contains(logOutput, "too long") {
					t.Errorf("Unexpected error in log: %q", logOutput)
				}
			}
		})
	}
}

func TestBroadcastFinalResultsValidation(t *testing.T) {
	buf, cleanup := setupBroadcastTest(t)
	defer cleanup()

	// Mock the DefaultStateProvider to avoid nil pointer issues
	originalProvider := DefaultStateProvider
	defer func() { DefaultStateProvider = originalProvider }()

	mockProvider := NewMockStateProvider()
	testMeetState := &MeetState{
		JudgeDecisions: map[string]string{
			"left":   "good",
			"center": "good",
			"right":  "no",
		},
	}

	// Set up mock expectations for all test cases that will call GetMeetState
	mockProvider.On("GetMeetState", "test-meet").Return(testMeetState)
	mockProvider.On("GetMeetState", "test-meet-@#$%").Return(testMeetState)
	mockProvider.On("GetMeetState", "test-meet-ñ").Return(testMeetState)

	DefaultStateProvider = mockProvider

	// Verify mock expectations at the end
	defer mockProvider.AssertExpectations(t)

	for _, tt := range broadcastFinalResultsValidationTests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			broadcastFinalResults(tt.meetName)

			logOutput := buf.String()

			if tt.shouldLog {
				if !strings.Contains(logOutput, tt.logText) {
					t.Errorf("Expected log to contain %q, got %q", tt.logText, logOutput)
				}
			} else {
				if strings.Contains(logOutput, "meetName is empty") {
					t.Errorf("Unexpected error in log: %q", logOutput)
				}
			}
		})
	}
}

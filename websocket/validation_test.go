package websocket

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"go-ref-lights/logger"
)

func TestValidateMeetName(t *testing.T) {
	// Capture log output for testing
	var buf bytes.Buffer
	logger.Error = log.New(&buf, "", 0)

	tests := []struct {
		name         string
		meetName     string
		functionName string
		expected     bool
		expectLog    string
	}{
		{
			name:         "valid meetName",
			meetName:     "test-meet",
			functionName: "TestFunction",
			expected:     true,
			expectLog:    "",
		},
		{
			name:         "empty meetName",
			meetName:     "",
			functionName: "TestFunction",
			expected:     false,
			expectLog:    "[TestFunction] meetName is empty - message will not be properly filtered",
		},
		{
			name:         "meetName too long",
			meetName:     strings.Repeat("a", 101),
			functionName: "TestFunction",
			expected:     false,
			expectLog:    "[TestFunction] meetName too long (101 chars) - potential security issue",
		},
		{
			name:         "meetName at limit",
			meetName:     strings.Repeat("a", 100),
			functionName: "TestFunction",
			expected:     true,
			expectLog:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			result := validateMeetName(tt.meetName, tt.functionName)

			if result != tt.expected {
				t.Errorf("validateMeetName() = %v, expected %v", result, tt.expected)
			}

			logOutput := strings.TrimSpace(buf.String())
			if tt.expectLog != "" && !strings.Contains(logOutput, tt.expectLog) {
				t.Errorf("Expected log to contain %q, got %q", tt.expectLog, logOutput)
			}
			if tt.expectLog == "" && logOutput != "" {
				t.Errorf("Expected no log output, got %q", logOutput)
			}
		})
	}
}

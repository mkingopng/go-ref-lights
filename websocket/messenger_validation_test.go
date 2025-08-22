package websocket

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"go-ref-lights/logger"
)

func TestRealMessengerBroadcastMessageValidation(t *testing.T) {
	// Capture log output for testing
	var buf bytes.Buffer
	logger.Error = log.New(&buf, "", 0)
	logger.Info = log.New(&buf, "", 0)

	// Initialize broadcast channel if not already done
	if broadcast == nil {
		broadcast = make(chan []byte, 100)
	}

	messenger := &realMessenger{}

	tests := []struct {
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
			logText:   "[realMessenger.BroadcastMessage] meetName is empty",
		},
		{
			name:      "meetName too long",
			meetName:  strings.Repeat("a", 101),
			shouldLog: true,
			logText:   "[realMessenger.BroadcastMessage] meetName too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			message := map[string]interface{}{
				"action": "test",
			}

			messenger.BroadcastMessage(tt.meetName, message)

			logOutput := buf.String()

			if tt.shouldLog {
				if !strings.Contains(logOutput, tt.logText) {
					t.Errorf("Expected log to contain %q, got %q", tt.logText, logOutput)
				}
			} else {
				// For valid meetName, we should see info log but no error
				if strings.Contains(logOutput, "meetName is empty") || strings.Contains(logOutput, "too long") {
					t.Errorf("Unexpected error in log: %q", logOutput)
				}
			}
		})
	}
}

func TestRealMessengerBroadcastTimeUpdateValidation(t *testing.T) {
	// Capture log output for testing
	var buf bytes.Buffer
	logger.Error = log.New(&buf, "", 0)
	logger.Info = log.New(&buf, "", 0)

	// Initialize broadcast channel if not already done
	if broadcast == nil {
		broadcast = make(chan []byte, 100)
	}

	messenger := &realMessenger{}

	tests := []struct {
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
			logText:   "[realMessenger.BroadcastTimeUpdate] meetName is empty",
		},
		{
			name:      "meetName too long",
			meetName:  strings.Repeat("a", 101),
			shouldLog: true,
			logText:   "[realMessenger.BroadcastTimeUpdate] meetName too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			messenger.BroadcastTimeUpdate("timeUpdate", 30, 1, tt.meetName)

			logOutput := buf.String()

			if tt.shouldLog {
				if !strings.Contains(logOutput, tt.logText) {
					t.Errorf("Expected log to contain %q, got %q", tt.logText, logOutput)
				}
			} else {
				// For valid meetName, we should see info log but no error
				if strings.Contains(logOutput, "meetName is empty") || strings.Contains(logOutput, "too long") {
					t.Errorf("Unexpected error in log: %q", logOutput)
				}
			}
		})
	}
}

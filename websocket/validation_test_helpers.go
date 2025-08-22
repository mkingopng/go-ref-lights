package websocket

import (
	"bytes"
	"log"
	"testing"

	"go-ref-lights/logger"
)

const (
	// Test constants
	MaxMeetNameLength = 100
	TestMeetName      = "test-meet"
	TestBufferSize    = 100
)

// ValidationTestSetup provides common setup for validation tests
type ValidationTestSetup struct {
	LogBuffer         *bytes.Buffer
	OriginalError     *log.Logger
	OriginalInfo      *log.Logger
	OriginalDebug     *log.Logger
	OriginalBroadcast chan []byte
}

// SetupValidationTest initializes common test resources
func SetupValidationTest(t *testing.T) *ValidationTestSetup {
	t.Helper()

	setup := &ValidationTestSetup{
		LogBuffer:         &bytes.Buffer{},
		OriginalError:     logger.Error,
		OriginalInfo:      logger.Info,
		OriginalDebug:     logger.Debug,
		OriginalBroadcast: broadcast,
	}

	// Setup log capture
	logger.Error = log.New(setup.LogBuffer, "", 0)
	logger.Info = log.New(setup.LogBuffer, "", 0)
	logger.Debug = log.New(setup.LogBuffer, "", 0)

	// Initialize broadcast channel if needed
	if broadcast == nil {
		broadcast = make(chan []byte, TestBufferSize)
	}

	return setup
}

// Cleanup restores original state
func (s *ValidationTestSetup) Cleanup() {
	logger.Error = s.OriginalError
	logger.Info = s.OriginalInfo
	logger.Debug = s.OriginalDebug
	broadcast = s.OriginalBroadcast
}

// ResetLogBuffer clears the log buffer for fresh test runs
func (s *ValidationTestSetup) ResetLogBuffer() {
	s.LogBuffer.Reset()
}

// GetLogOutput returns the current log buffer content
func (s *ValidationTestSetup) GetLogOutput() string {
	return s.LogBuffer.String()
}

// ValidationTestCase represents a test case for meetName validation
type ValidationTestCase struct {
	Name        string
	MeetName    string
	ShouldLog   bool
	ExpectedLog string
	ShouldPass  bool
}

// GetStandardValidationCases returns common test cases for meetName validation
func GetStandardValidationCases() []ValidationTestCase {
	return []ValidationTestCase{
		{
			Name:       "valid meetName",
			MeetName:   TestMeetName,
			ShouldLog:  false,
			ShouldPass: true,
		},
		{
			Name:        "empty meetName",
			MeetName:    "",
			ShouldLog:   true,
			ExpectedLog: ErrEmptyMeetName,
			ShouldPass:  false,
		},
		{
			Name:       "whitespace only meetName",
			MeetName:   "   ",
			ShouldLog:  false, // Current implementation doesn't trim
			ShouldPass: true,
		},
		{
			Name:        "meetName too long",
			MeetName:    generateLongString(MaxMeetNameLength + 1),
			ShouldLog:   true,
			ExpectedLog: "meetName too long",
			ShouldPass:  false,
		},
	}
}

// generateLongString creates a string of specified length
func generateLongString(length int) string {
	if length <= 0 {
		return ""
	}
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

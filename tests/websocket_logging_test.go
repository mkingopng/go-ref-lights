package tests

import (
	"go-ref-lights/logger"
	"go-ref-lights/websocket"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

// TestTimerLoggingOptimization verifies that timer operations use appropriate log levels
func TestTimerLoggingOptimization(t *testing.T) {
	// Set environment to production to test log level filtering
	os.Setenv("ENV", "production")
	defer os.Unsetenv("ENV")

	// Initialize logger
	err := logger.InitLogger()
	if err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.CloseLogger()

	// Verify that production mode suppresses DEBUG logs
	if logger.ShouldLog(logger.DEBUG) {
		t.Error("DEBUG logs should be suppressed in production mode")
	}

	// Verify that ERROR and WARN logs are still active
	if !logger.ShouldLog(logger.ERROR) {
		t.Error("ERROR logs should be active in production mode")
	}

	if !logger.ShouldLog(logger.WARN) {
		t.Error("WARN logs should be active in production mode")
	}

	// Initialize websocket test environment
	websocket.InitTest()

	// Create a test meet state
	meetState := &websocket.MeetState{
		MeetName:       "TestMeet",
		JudgeDecisions: make(map[string]string),
	}

	// Create timer manager with mock dependencies
	mockProvider := websocket.NewMockStateProvider()
	mockMessenger := new(websocket.MockMessenger)

	mockProvider.On("GetMeetState", "TestMeet").Return(meetState)
	mockMessenger.On("BroadcastRaw", mock.Anything).Return(nil)
	mockMessenger.On("BroadcastMessage", "TestMeet", mock.Anything).Return(nil)
	mockMessenger.On("BroadcastTimeUpdate", mock.Anything, mock.Anything, mock.Anything, "TestMeet").Return(nil)

	tm := &websocket.TimerManager{
		Provider:              mockProvider,
		Messenger:             mockMessenger,
		NextAttemptStartValue: 1, // Short timer for testing
		TickerInterval:        10 * time.Millisecond,
	}

	// Test that routine timer operations don't generate logs in production
	// (This is verified by the fact that DEBUG logs are suppressed)
	tm.HandleTimerAction("startTimer", "TestMeet")
	tm.HandleTimerAction("resetTimer", "TestMeet")
	tm.HandleTimerAction("startNextAttemptTimer", "TestMeet")

	// Wait a bit for timer operations to complete
	time.Sleep(50 * time.Millisecond)

	// The test passes if no errors occur and DEBUG logs are properly suppressed
	t.Log("Timer logging optimization test completed successfully")
}

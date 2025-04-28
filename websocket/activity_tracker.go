// Package websocket provides activityTracking functionality
// File: websocket/activity_tracker.go
package websocket

import (
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go-ref-lights/logger"
)

// ActivityTracker monitors system activity and can trigger shutdown after inactivity
type ActivityTracker struct {
	lastActivityTime   atomic.Value // stores a time.Time
	shutdownAfterIdle  time.Duration
	checkInterval      time.Duration
	shutdownInProgress int32
	stopChan           chan struct{}
	shutdownFunc       func()
	mu                 sync.Mutex
}

// NewActivityTracker creates a new ActivityTracker with the specified idle timeout
func NewActivityTracker(idleTimeout time.Duration) *ActivityTracker {
	tracker := &ActivityTracker{
		shutdownAfterIdle: idleTimeout,
		checkInterval:     2 * time.Minute, // Check every 2 minutes
		stopChan:          make(chan struct{}),
		shutdownFunc:      defaultShutdown,
	}

	// Initialize with current time
	tracker.lastActivityTime.Store(time.Now())

	return tracker
}

// defaultShutdown is the default implementation to shut down the application
func defaultShutdown() {
	logger.Info.Println("[ActivityTracker] Triggering graceful shutdown after extended inactivity")

	// Send interrupt signal to self
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		logger.Error.Printf("[ActivityTracker] Failed to find own process: %v", err)
		return
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		logger.Error.Printf("[ActivityTracker] Failed to send SIGTERM: %v", err)
	}
}

// RecordActivity records that activity has occurred, resetting the idle timer
func (t *ActivityTracker) RecordActivity() {
	t.lastActivityTime.Store(time.Now())
	logger.Debug.Println("[ActivityTracker] Activity recorded, idle timer reset")
}

// GetLastActivityTime returns the time of the last recorded activity
func (t *ActivityTracker) GetLastActivityTime() time.Time {
	return t.lastActivityTime.Load().(time.Time)
}

// Start begins monitoring for inactivity
func (t *ActivityTracker) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	logger.Info.Printf("[ActivityTracker] Starting activity monitor (idle timeout: %v)", t.shutdownAfterIdle)

	// Run the monitoring loop in a goroutine
	go func() {
		ticker := time.NewTicker(t.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.checkIdleStatus()
			case <-t.stopChan:
				logger.Info.Println("[ActivityTracker] Activity monitor stopped")
				return
			}
		}
	}()
}

// Stop stops the activity monitor
func (t *ActivityTracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case t.stopChan <- struct{}{}:
		logger.Info.Println("[ActivityTracker] Stopping activity monitor")
	default:
		// Already stopped
	}
}

// SetShutdownFunc allows customizing the shutdown function
func (t *ActivityTracker) SetShutdownFunc(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.shutdownFunc = f
}

// checkIdleStatus checks if the system has been idle for too long and triggers shutdown if needed
func (t *ActivityTracker) checkIdleStatus() {
	lastActivity := t.GetLastActivityTime()
	idleTime := time.Since(lastActivity)

	if idleTime > t.shutdownAfterIdle {
		// Only log and trigger shutdown once
		if atomic.CompareAndSwapInt32(&t.shutdownInProgress, 0, 1) {
			logger.Warn.Printf("[ActivityTracker] System has been idle for %v (threshold: %v), initiating shutdown",
				idleTime.Round(time.Second), t.shutdownAfterIdle)

			// Allow some time for final logs to be written
			time.Sleep(3 * time.Second)

			// Execute the shutdown function
			t.shutdownFunc()
		}
	} else {
		idleMinutes := int(idleTime.Minutes())
		// Only log every 10 minutes to avoid spamming logs
		if idleMinutes > 0 && idleMinutes%10 == 0 {
			logger.Info.Printf("[ActivityTracker] System has been idle for %v", idleTime.Round(time.Minute))
		}
	}
}

// Singleton instance for the application
var (
	defaultTracker     *ActivityTracker
	defaultTrackerOnce sync.Once
)

// GetActivityTracker returns the default activity tracker instance
func GetActivityTracker() *ActivityTracker {
	defaultTrackerOnce.Do(func() {
		// Default to 30 minutes idle timeout
		defaultTracker = NewActivityTracker(30 * time.Minute)
	})
	return defaultTracker
}

// RecordSystemActivity is a convenience function to record activity on the default tracker
func RecordSystemActivity() {
	GetActivityTracker().RecordActivity()
}

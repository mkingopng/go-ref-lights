// file: heartbeat.go
package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-ref-lights/logger"
)

var (
	refereeSessions = make(map[string]time.Time)
	sessionLock     = sync.Mutex{}
)

// HeartbeatManager tracks active referees
type HeartbeatManager struct {
	activeSessions map[string]time.Time
	mu             sync.Mutex
}

// HeartbeatHandler updates the last seen timestamp of a referee
func HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	refereeID := r.URL.Query().Get("referee_id")
	if refereeID == "" {
		logger.Warn.Println("[HeartbeatHandler] Missing referee ID in query params")
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	sessionLock.Lock()
	refereeSessions[refereeID] = time.Now()
	sessionLock.Unlock()

	logger.Debug.Printf("[HeartbeatHandler] Updated heartbeat for referee=%s at %v", refereeID, time.Now())

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "Heartbeat received"); err != nil {
		logger.Warn.Printf("[HeartbeatHandler] Error writing response for referee=%s: %v", refereeID, err)
	}
}

// NewHeartbeatManager initializes a heartbeat tracker
func NewHeartbeatManager() *HeartbeatManager {
	return &HeartbeatManager{
		activeSessions: make(map[string]time.Time),
	}
}

// UpdateHeartbeat marks a referee as active
func (h *HeartbeatManager) UpdateHeartbeat(refereeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeSessions[refereeID] = time.Now()
	logger.Debug.Printf("[HeartbeatManager.UpdateHeartbeat] Referee=%s updated at %v", refereeID, time.Now())
}

// CleanupInactiveSessions removes inactive referees
func (h *HeartbeatManager) CleanupInactiveSessions(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			for id, lastSeen := range h.activeSessions {
				if time.Since(lastSeen) > timeout {
					logger.Info.Printf("[HeartbeatManager.CleanupInactiveSessions] Removing inactive referee=%s (timeout=%v)", id, timeout)
					delete(h.activeSessions, id)
				}
			}
			h.mu.Unlock()
		}
	}()
}

// CleanupRoutine removes referees that have been inactive
func CleanupRoutine() {
	ticker := time.NewTicker(10 * time.Second) // adjust interval as needed
	for range ticker.C {
		sessionLock.Lock()
		for id, lastSeen := range refereeSessions {
			if time.Since(lastSeen) > 1800*time.Second { // configurable timeout, 30 minutes
				logger.Info.Printf("[CleanupRoutine] Removing inactive referee=%s (30 minutes)", id)
				delete(refereeSessions, id)
			}
		}
		sessionLock.Unlock()
	}
}

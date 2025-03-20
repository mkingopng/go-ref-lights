// file: heartbeat.go
package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
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
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	sessionLock.Lock()
	refereeSessions[refereeID] = time.Now()
	sessionLock.Unlock()

	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintln(w, "Heartbeat received")
	if err != nil {
		return
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
}

// CleanupInactiveSessions removes inactive referees
func (h *HeartbeatManager) CleanupInactiveSessions(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			for id, lastSeen := range h.activeSessions {
				if time.Since(lastSeen) > timeout {
					delete(h.activeSessions, id) // Remove inactive referees
				}
			}
			h.mu.Unlock()
		}
	}()
}

// CleanupRoutine removes referees that have been inactive
func CleanupRoutine() {
	ticker := time.NewTicker(10 * time.Second) // Adjust interval as needed
	for range ticker.C {
		sessionLock.Lock()
		for id, lastSeen := range refereeSessions {
			if time.Since(lastSeen) > 30*time.Second { // Configurable timeout
				delete(refereeSessions, id) // Remove inactive session
			}
		}
		sessionLock.Unlock()
	}
}

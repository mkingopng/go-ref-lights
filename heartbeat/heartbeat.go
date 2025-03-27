// Package heartbeat
// file: heartbeat.go
package heartbeat

import (
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
	"net/http"
	"sync"
	"time"

	"go-ref-lights/logger"
)

var (
	refereeSessions = make(map[string]time.Time)
	sessionLock     = sync.Mutex{}
)

// Manager tracks active referees
type Manager struct {
	activeSessions map[string]time.Time
	mu             sync.Mutex
}

// Handler updates the last seen timestamp of a referee
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx, seg := xray.BeginSubsegment(r.Context(), "HeartbeatHandler")
	defer seg.Close(nil)
	r = r.WithContext(ctx)

	refereeID := r.URL.Query().Get("referee_id")
	if refereeID != "" {
		err := seg.AddAnnotation("refereeID", refereeID)
		if err != nil {
			return
		}
	}

	if refereeID == "" {
		logger.Warn.Println("[Handler] Missing referee ID in query params")
		http.Error(w, "Missing referee ID", http.StatusBadRequest)
		return
	}

	sessionLock.Lock()
	refereeSessions[refereeID] = time.Now()
	sessionLock.Unlock()

	logger.Debug.Printf("[Handler] Updated heartbeat for referee=%s at %v", refereeID, time.Now())

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintln(w, "Heartbeat received"); err != nil {
		logger.Warn.Printf("[Handler] Error writing response for referee=%s: %v", refereeID, err)
	}
}

// NewHeartbeatManager initializes a heartbeat tracker
func NewHeartbeatManager() *Manager {
	return &Manager{
		activeSessions: make(map[string]time.Time),
	}
}

// UpdateHeartbeat marks a referee as active
func (h *Manager) UpdateHeartbeat(refereeID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.activeSessions[refereeID] = time.Now()
	logger.Debug.Printf("[Manager.UpdateHeartbeat] Referee=%s updated at %v", refereeID, time.Now())
}

// CleanupInactiveSessions removes inactive referees
func (h *Manager) CleanupInactiveSessions(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			for id, lastSeen := range h.activeSessions {
				if time.Since(lastSeen) > timeout {
					logger.Info.Printf("[Manager.CleanupInactiveSessions] Removing inactive referee=%s (timeout=%v)", id, timeout)
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

// Package websocket - websocket/metrics.go
// file: websocket/metrics.go
package websocket

import (
	"time"

	"go-ref-lights/logger"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

// namespace for all RefVision metrics
var metricsNamespace = "RefVision"

// reuse a single CloudWatch client for all metrics calls
var cwClient = cloudwatch.New(session.Must(session.NewSession()))

// SystemHealthMetrics contains real-time health data for the admin dashboard
type SystemHealthMetrics struct {
	ActiveConnections   int                                 `json:"activeConnections"`
	BroadcastQueueSize  int                                 `json:"broadcastQueueSize"`
	RefereeConnections  map[string][]string                 `json:"refereeConnections"`
	PlatformReadyTimers map[string]PlatformReadyTimerStatus `json:"platformReadyTimers"`
	NextAttemptTimers   map[string][]NextAttemptTimerStatus `json:"nextAttemptTimers"`
	TimestampUTC        string                              `json:"timestampUTC"`
}

// PlatformReadyTimerStatus contains status information for platform ready timers
type PlatformReadyTimerStatus struct {
	Active      bool      `json:"active"`
	TimeLeftSec int       `json:"timeLeftSec"`
	EndTime     time.Time `json:"endTime"`
}

// NextAttemptTimerStatus contains status information for next attempt timers
type NextAttemptTimerStatus struct {
	ID          int       `json:"id"`
	Active      bool      `json:"active"`
	TimeLeftSec int       `json:"timeLeftSec"`
	EndTime     time.Time `json:"endTime"`
}

// GetSystemHealthMetrics returns current health metrics for the system
func GetSystemHealthMetrics() SystemHealthMetrics {
	metrics := SystemHealthMetrics{
		RefereeConnections:  make(map[string][]string),
		PlatformReadyTimers: make(map[string]PlatformReadyTimerStatus),
		NextAttemptTimers:   make(map[string][]NextAttemptTimerStatus),
		TimestampUTC:        time.Now().UTC().Format(time.RFC3339),
	}

	// Get active connections count and referee connections by meet
	connectionsMu.RLock()
	metrics.ActiveConnections = len(connections)

	// Group connections by meet name
	for conn := range connections {
		if conn.meetName != "" {
			if conn.judgeID != "" {
				metrics.RefereeConnections[conn.meetName] = append(
					metrics.RefereeConnections[conn.meetName], conn.judgeID)
			}
		}
	}
	connectionsMu.RUnlock()

	// Get broadcast queue size (current buffer usage)
	metrics.BroadcastQueueSize = len(broadcast)

	// Get platform ready and next attempt timer status
	meetsMutex.Lock()
	for meetName, meetState := range meets {
		// Platform Ready timer
		if meetState.PlatformReadyActive && !meetState.PlatformReadyEnd.IsZero() {
			timeLeft := int(time.Until(meetState.PlatformReadyEnd).Seconds())
			if timeLeft < 0 {
				timeLeft = 0
			}

			metrics.PlatformReadyTimers[meetName] = PlatformReadyTimerStatus{
				Active:      meetState.PlatformReadyActive,
				TimeLeftSec: timeLeft,
				EndTime:     meetState.PlatformReadyEnd,
			}
		}

		// Next Attempt timers
		for _, timer := range meetState.NextAttemptTimers {
			if timer.Active {
				timeLeft := int(time.Until(timer.EndTime).Seconds())
				if timeLeft < 0 {
					timeLeft = 0
				}

				metrics.NextAttemptTimers[meetName] = append(
					metrics.NextAttemptTimers[meetName],
					NextAttemptTimerStatus{
						ID:          timer.ID,
						Active:      timer.Active,
						TimeLeftSec: timeLeft,
						EndTime:     timer.EndTime,
					})
			}
		}
	}
	meetsMutex.Unlock()

	// Also report these metrics to CloudWatch
	PublishBroadcastBacklog(metrics.BroadcastQueueSize, "All")

	return metrics
}

// PublishRefereeConnections pushes current WebSocket connection count
func PublishRefereeConnections(count int, meetName string) {
	putMetric("RefereeConnections", float64(count), "Count", meetName)
}

// PublishDecisionLatency pushes latency from first to third decision (in ms)
func PublishDecisionLatency(latencyMs float64, meetName string) {
	putMetric("DecisionLatencyMs", latencyMs, "Milliseconds", meetName)
}

// PublishBroadcastBacklog pushes a gauge for broadcast queue depth
func PublishBroadcastBacklog(depth int, meetName string) {
	putMetric("BroadcastQueueDepth", float64(depth), "Count", meetName)
}

// putMetric sends a single metric to CloudWatch
func putMetric(metricName string, value float64, unit string, meetName string) {
	_, err := cwClient.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String(metricsNamespace),
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String(metricName),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("MeetName"),
						Value: aws.String(meetName),
					},
				},
				Timestamp: aws.Time(time.Now()),
				Value:     aws.Float64(value),
				Unit:      aws.String(unit),
			},
		},
	})

	if err != nil {
		logger.Error.Printf("[putMetric] CloudWatch metric failed (%s): %v", metricName, err)
	}
}

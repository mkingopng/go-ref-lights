// Package websocket - websocket/metrics.go
// file: websocket/metrics.go

package websocket

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"go-ref-lights/logger"
)

// Namespace for all RefVision metrics
var metricsNamespace = "RefVision"

// Reuse a single CloudWatch client for all metrics calls
var cwClient = cloudwatch.New(session.Must(session.NewSession()))

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

// -----------------------------------------------------------
// internal helper function to package up CloudWatch calls
// -----------------------------------------------------------
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

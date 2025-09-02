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
		// Keep ERROR level for CloudWatch metric failures
		context := logger.NewSystemContext("cloudwatch_metric_failed", "metrics")
		context["metricName"] = metricName
		context = logger.AddError(context, err)
		logger.LogErrorWithContext(context, "CloudWatch metric submission failed")
	}
}

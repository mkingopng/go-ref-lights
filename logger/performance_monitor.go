package logger

import (
	"sync/atomic"
	"time"
)

// PerformanceMonitor provides real-time logging performance monitoring
type PerformanceMonitor struct {
	logger          *Logger
	alertThresholds AlertThresholds
	lastAlertTime   int64 // Unix timestamp of last alert
	alertCooldown   time.Duration
}

// AlertThresholds defines performance alert thresholds
type AlertThresholds struct {
	MaxLogsPerSecond  float64
	MaxBytesPerSecond float64
	MaxFileSize       int64
	MaxMemoryUsage    int64
}

// DefaultAlertThresholds returns sensible default alert thresholds
func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		MaxLogsPerSecond:  1000,
		MaxBytesPerSecond: 1024 * 1024,      // 1MB/sec
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		MaxMemoryUsage:    50 * 1024 * 1024, // 50MB
	}
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger:          logger,
		alertThresholds: DefaultAlertThresholds(),
		alertCooldown:   5 * time.Minute, // Don't spam alerts
	}
}

// SetThresholds updates the alert thresholds
func (pm *PerformanceMonitor) SetThresholds(thresholds AlertThresholds) {
	pm.alertThresholds = thresholds
}

// CheckPerformance monitors current performance and triggers alerts if needed
func (pm *PerformanceMonitor) CheckPerformance() *PerformanceReport {
	report := pm.generateReport()

	if pm.shouldAlert(report) {
		pm.triggerAlert(report)
	}

	return report
}

// PerformanceReport contains current performance metrics
type PerformanceReport struct {
	Timestamp         time.Time
	LogsPerSecond     float64
	BytesPerSecond    float64
	CurrentFileSize   int64
	TotalLogsWritten  int64
	TotalBytesWritten int64
	Uptime            time.Duration
	AlertsTriggered   []string
}

// generateReport creates a current performance report
func (pm *PerformanceMonitor) generateReport() *PerformanceReport {
	logCount, bytesLogged, logsPerSecond, bytesPerSecond := pm.logger.GetPerformanceStats()
	currentSize, _, _ := pm.logger.GetFileSizeInfo()

	return &PerformanceReport{
		Timestamp:         time.Now(),
		LogsPerSecond:     logsPerSecond,
		BytesPerSecond:    bytesPerSecond,
		CurrentFileSize:   currentSize,
		TotalLogsWritten:  logCount,
		TotalBytesWritten: bytesLogged,
		Uptime:            time.Since(pm.logger.startTime),
		AlertsTriggered:   make([]string, 0),
	}
}

// shouldAlert determines if an alert should be triggered based on current metrics
func (pm *PerformanceMonitor) shouldAlert(report *PerformanceReport) bool {
	// Check cooldown period
	lastAlert := atomic.LoadInt64(&pm.lastAlertTime)
	if time.Since(time.Unix(lastAlert, 0)) < pm.alertCooldown {
		return false
	}

	// Check thresholds
	return report.LogsPerSecond > pm.alertThresholds.MaxLogsPerSecond ||
		report.BytesPerSecond > pm.alertThresholds.MaxBytesPerSecond ||
		report.CurrentFileSize > pm.alertThresholds.MaxFileSize
}

// triggerAlert sends performance alerts
func (pm *PerformanceMonitor) triggerAlert(report *PerformanceReport) {
	atomic.StoreInt64(&pm.lastAlertTime, time.Now().Unix())

	alertContext := NewSystemContext("performance_alert", "logger")
	alertContext["logsPerSecond"] = report.LogsPerSecond
	alertContext["bytesPerSecond"] = report.BytesPerSecond
	alertContext["currentFileSize"] = report.CurrentFileSize
	alertContext["uptime"] = report.Uptime.String()

	if report.LogsPerSecond > pm.alertThresholds.MaxLogsPerSecond {
		report.AlertsTriggered = append(report.AlertsTriggered, "high_log_rate")
		pm.logger.LogWithContext(WARN, alertContext,
			"High logging rate detected: %.2f logs/sec (threshold: %.2f)",
			report.LogsPerSecond, pm.alertThresholds.MaxLogsPerSecond)
	}

	if report.BytesPerSecond > pm.alertThresholds.MaxBytesPerSecond {
		report.AlertsTriggered = append(report.AlertsTriggered, "high_byte_rate")
		pm.logger.LogWithContext(WARN, alertContext,
			"High logging byte rate detected: %.2f bytes/sec (threshold: %.2f)",
			report.BytesPerSecond, pm.alertThresholds.MaxBytesPerSecond)
	}

	if report.CurrentFileSize > pm.alertThresholds.MaxFileSize {
		report.AlertsTriggered = append(report.AlertsTriggered, "large_file_size")
		pm.logger.LogWithContext(WARN, alertContext,
			"Large log file detected: %d bytes (threshold: %d)",
			report.CurrentFileSize, pm.alertThresholds.MaxFileSize)
	}
}

// StartMonitoring begins continuous performance monitoring
func (pm *PerformanceMonitor) StartMonitoring(interval time.Duration) chan *PerformanceReport {
	reports := make(chan *PerformanceReport, 10)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			report := pm.CheckPerformance()
			select {
			case reports <- report:
			default:
				// Channel full, skip this report
			}
		}
	}()

	return reports
}

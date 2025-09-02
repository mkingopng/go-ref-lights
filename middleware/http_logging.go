// Package middleware provides HTTP request logging middleware with environment-based filtering.
// File: middleware/http_logging.go
package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"

	"go-ref-lights/logger"
)

// responseWriter wraps gin.ResponseWriter to capture response body and status
type responseWriter struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// HTTPLoggingMiddleware provides environment-aware HTTP request logging
func HTTPLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for static assets and health checks in production
		if shouldSkipLogging(c.Request.RequestURI) {
			c.Next()
			return
		}

		start := time.Now()

		// Wrap the response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
			statusCode:     200, // Default status code
		}
		c.Writer = wrapped

		// Process request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Create HTTP context for logging
		httpContext := logger.NewHTTPContext(
			c.Request.Method,
			c.Request.RequestURI,
			c.Request.UserAgent(),
			c.ClientIP(),
			wrapped.statusCode,
		)

		// Add additional context
		httpContext["duration"] = duration.String()
		httpContext["contentLength"] = c.Request.ContentLength

		// Log based on response status and environment
		logHTTPRequest(httpContext, wrapped.statusCode, c.Request.Method, c.Request.RequestURI, duration)
	}
}

// shouldSkipLogging determines if a request should be skipped from logging
func shouldSkipLogging(uri string) bool {
	// Skip static assets, favicon, and frequent health checks in production
	skipPaths := []string{
		"/static/",
		"/favicon.ico",
		"/heartbeat", // Skip frequent heartbeat requests
	}

	for _, path := range skipPaths {
		if len(uri) >= len(path) && uri[:len(path)] == path {
			return true
		}
	}

	return false
}

// logHTTPRequest logs HTTP requests based on status code and environment
func logHTTPRequest(context map[string]interface{}, statusCode int, method, uri string, duration time.Duration) {
	// Always log errors (4xx, 5xx) with full context
	if statusCode >= 400 {
		if statusCode >= 500 {
			// Server errors - always log as ERROR
			logger.LogErrorWithContext(context, "HTTP server error: %s %s - Status: %d, Duration: %v",
				method, uri, statusCode, duration)
		} else {
			// Client errors - log as WARN
			logger.LogWarnWithContext(context, "HTTP client error: %s %s - Status: %d, Duration: %v",
				method, uri, statusCode, duration)
		}
		return
	}

	// Success responses (2xx, 3xx) - only log in development mode
	if statusCode >= 200 && statusCode < 400 {
		// Log successful requests only at DEBUG level (development mode only)
		logger.LogDebugWithContext(context, "HTTP request: %s %s - Status: %d, Duration: %v",
			method, uri, statusCode, duration)
		return
	}

	// Informational responses (1xx) - log at DEBUG level
	logger.LogDebugWithContext(context, "HTTP informational: %s %s - Status: %d, Duration: %v",
		method, uri, statusCode, duration)
}

// AuthenticationLoggingMiddleware provides specialized logging for authentication events
func AuthenticationLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to authentication-related endpoints
		if !isAuthenticationEndpoint(c.Request.RequestURI) {
			c.Next()
			return
		}

		// Capture the original body for POST requests (login attempts)
		var bodyBytes []byte
		if c.Request.Method == "POST" && c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Wrap response writer
		wrapped := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
			statusCode:     200,
		}
		c.Writer = wrapped

		// Process request
		c.Next()

		// Log authentication attempts with sanitized information
		logAuthenticationAttempt(c, wrapped.statusCode, bodyBytes)
	}
}

// isAuthenticationEndpoint checks if the URI is an authentication endpoint
func isAuthenticationEndpoint(uri string) bool {
	authEndpoints := []string{
		"/login",
		"/logout",
		"/set-meet",
		"/force-my-login",
	}

	for _, endpoint := range authEndpoints {
		if uri == endpoint || (len(uri) > len(endpoint) && uri[:len(endpoint)] == endpoint) {
			return true
		}
	}

	return false
}

// logAuthenticationAttempt logs authentication attempts with appropriate detail level
func logAuthenticationAttempt(c *gin.Context, statusCode int, bodyBytes []byte) {
	authContext := logger.NewAuthenticationContext(
		c.Request.Method+" "+c.Request.RequestURI,
		extractUsernameFromRequest(c, bodyBytes),
		c.ClientIP(),
	)

	// Add request details
	authContext["statusCode"] = statusCode
	authContext["userAgent"] = c.Request.UserAgent()

	if statusCode >= 400 {
		// Authentication failures - always log with context
		if statusCode == 401 {
			logger.LogWarnWithContext(authContext, "Authentication failed: Invalid credentials")
		} else if statusCode == 403 {
			logger.LogWarnWithContext(authContext, "Authentication failed: Access forbidden")
		} else {
			logger.LogWarnWithContext(authContext, "Authentication error: Status %d", statusCode)
		}
	} else if statusCode >= 200 && statusCode < 300 {
		// Successful authentication - log at INFO level (critical for security auditing)
		logger.LogInfoWithContext(authContext, "Authentication successful")
	} else {
		// Redirects and other responses - log at DEBUG level
		logger.LogDebugWithContext(authContext, "Authentication response: Status %d", statusCode)
	}
}

// extractUsernameFromRequest safely extracts username from request for logging
func extractUsernameFromRequest(c *gin.Context, bodyBytes []byte) string {
	// Try to get username from form data first
	if username := c.PostForm("username"); username != "" {
		return username
	}

	// Try to get from query parameters
	if username := c.Query("username"); username != "" {
		return username
	}

	// For session-based requests, try to get from session
	// This would require importing sessions package, so we'll skip for now
	// and rely on the controller-level logging for session-based username extraction

	return "unknown"
}

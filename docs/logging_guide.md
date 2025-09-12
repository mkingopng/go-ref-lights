# RefLights Logging System Guide

## Overview

RefLights uses an optimized logging system designed for production reliability and development debugging. The system provides environment-based log levels, structured JSON logging, and rich contextual information for effective monitoring and troubleshooting.

## Log Levels

The logging system uses a hierarchical level system:

| Level | Description                                   | Production | Development | Test |
|-------|-----------------------------------------------|------------|-------------|------|
| ERROR | Critical errors requiring immediate attention | ✅          | ✅           | ✅    |
| WARN  | Important warnings that should be monitored   | ✅          | ✅           | ✅    |
| INFO  | General information messages                  | ⚠️*        | ✅           | ❌    |
| DEBUG | Detailed debugging information                | ❌          | ✅           | ❌    |

*Critical INFO only in production

## Environment Configuration

### Environment Variables

- **ENV**: Controls the base logging level and application mode
  - `production`: ERROR, WARN, and critical INFO only
  - `development`: All levels including DEBUG
  - `test`: ERROR and WARN only
  - Default: `production` (safety first)

- **LOG_LEVEL**: Optional override for explicit level control
  - Values: `DEBUG`, `INFO`, `WARN`, `ERROR`
  - Takes precedence over ENV-based configuration

### Configuration Examples

```bash
# Production mode with minimal logging
ENV=production ./go-ref-lights

# Development mode with full logging
ENV=development ./go-ref-lights

# Production mode with debug override (troubleshooting)
ENV=production LOG_LEVEL=DEBUG ./go-ref-lights

# Test mode for clean test output
ENV=test ./go-ref-lights
```

## Structured Logging Format

Production logs use JSON format for easy parsing and analysis:

```json
{
  "timestamp": "2023-01-01T12:00:00Z",
  "level": "ERROR",
  "message": "WebSocket connection failed for referee left",
  "context": {
    "component": "websocket",
    "action": "connection_failed",
    "meetName": "Test Meet",
    "refereeId": "left",
    "remoteAddr": "192.168.1.100",
    "errorCategory": "websocket",
    "errorSeverity": "medium",
    "errorCode": "WS_001"
  },
  "source": "connection.go:123",
  "meetName": "Test Meet",
  "refereeId": "left",
  "error": "connection timeout"
}
```

## Error Categorization System

RefLights includes a comprehensive error categorization system for structured error analysis and monitoring:

### Error Categories

| Category         | Description                        | Examples                              |
|------------------|------------------------------------|---------------------------------------|
| `authentication` | Login failures, credential issues  | Invalid passwords, duplicate logins   |
| `authorization`  | Permission denied, access control  | Unauthorized resource access          |
| `websocket`      | WebSocket connection/communication | Connection upgrades, message failures |
| `timer`          | Timer operation failures           | Start/stop failures, invalid states   |
| `position`       | Position occupancy conflicts       | Seat conflicts, invalid positions     |
| `validation`     | Input validation failures          | Invalid form data, missing fields     |
| `session`        | Session management failures        | Session save/load errors              |
| `system`         | System-level failures              | Configuration errors, startup issues  |
| `network`        | Network connectivity issues        | Connection timeouts, DNS failures     |
| `marshaling`     | JSON marshaling/unmarshaling       | Serialization errors                  |

### Error Severity Levels

| Severity   | Description                             | Response Required    |
|------------|-----------------------------------------|----------------------|
| `critical` | System-threatening errors               | Immediate attention  |
| `high`     | Major functionality impact              | Urgent investigation |
| `medium`   | Moderate impact, degraded functionality | Monitor and address  |
| `low`      | Minor issues, validation failures       | Log for analysis     |

### Error Codes

Systematic error codes for tracking and troubleshooting:

- **AUTH_xxx** - Authentication and authorization errors
- **SESS_xxx** - Session management errors
- **WS_xxx** - WebSocket-related errors
- **MSG_xxx** - Message processing errors
- **POS_xxx** - Position occupancy errors
- **TIM_xxx** - Timer operation errors

### Context Categories

The system provides structured context for different operation types:

- **websocket**: WebSocket connection and messaging operations (optimized for production)
- **timer**: Platform ready and next attempt timer operations
- **authentication**: Login, logout, and credential validation
- **position**: Referee position claiming and vacating
- **http**: HTTP request and response logging
- **system**: Application startup, shutdown, and system events

## Log File Management

### File Location
- Logs are stored in `./logs/` directory
- Files are named with timestamp: `2023-01-01_12-00-00.log`
- New file created each time application starts

### File Rotation
- No automatic rotation (meets are typically short-duration)
- Manual cleanup recommended between events
- Consider external log rotation for long-running deployments

## Development Usage

### Legacy Logging (Backward Compatible)
```go
// Existing code continues to work
logger.Info.Println("Application started")
logger.Error.Printf("Error occurred: %v", err)
```

### Structured Logging (Recommended)
```go
// WebSocket error with context
context := logger.NewWebSocketContext("connection_failed", meetName, refereeID, remoteAddr)
context = logger.AddError(context, err)
logger.LogErrorWithContext(context, "Failed to establish WebSocket connection")

// Timer operation with context
context := logger.NewTimerContext("start_failed", meetName, "platformReady", timerID)
logger.LogWarnWithContext(context, "Timer start failed, timer already active")

// Authentication failure with context
context := logger.NewAuthenticationContext("login_failed", username, clientIP)
logger.LogErrorWithContext(context, "Authentication failed for user %s", username)
```

### Error Context Logging (Enhanced)
```go
// Authentication error with categorization
errorCtx := logger.NewAuthenticationErrorContext(
    "Authentication failed: Invalid username or password",
    username,
    clientIP,
    userAgent,
).WithCode("AUTH_001").
    WithMeet(meetName, "").
    WithDetail("failureReason", "invalid_credentials")

errorCtx.LogWarn()

// WebSocket error with full context
errorCtx := logger.NewWebSocketErrorContext(
    "WebSocket upgrade failed",
    meetName,
    refereeID,
    remoteAddr,
).WithCode("WS_001").
    WithError(err).
    WithDetail("userAgent", userAgent)

errorCtx.LogError()

// Position conflict with detailed context
errorCtx := logger.NewPositionErrorContext(
    "Position occupancy conflict: left seat already taken",
    meetName,
    position,
    userEmail,
).WithCode("POS_002").
    WithDetail("currentOccupant", currentUser).
    WithDetail("conflictType", "position_occupied")

errorCtx.LogWarn()
```

### Performance Optimization
```go
// Check if logging is enabled before expensive operations
if logger.ShouldLog(logger.DEBUG) {
    logger.LogDebugWithContext(context, "Expensive debug info: %s", expensiveFunction())
}
```

## Production Monitoring

### WebSocket Logging Optimization

The WebSocket logging system has been optimized for production use:

**Production Mode (ERROR/WARN only):**
- Connection upgrade attempts: Suppressed (DEBUG level)
- Routine message processing: Suppressed (DEBUG level)
- Referee registration: Suppressed (DEBUG level)
- Timer commands: Suppressed (DEBUG level)
- Decision processing: Suppressed (DEBUG level)

**Always Logged (ERROR/WARN):**
- Connection upgrade failures
- JSON parsing errors
- Message write failures
- Ping/pong failures
- Dropped messages due to full channels
- Incomplete decision data

**Development Mode:**
- All WebSocket operations logged with full context
- Complete message flow visibility for debugging
- Detailed connection lifecycle tracking

### Timer Logging Optimization

The timer system has been optimized to reduce log noise while maintaining error visibility:

**Production Mode (ERROR/WARN only):**
- Timer start/stop operations: Suppressed (DEBUG level)
- Routine countdown updates: Suppressed (DEBUG level)
- Timer expiration events: Suppressed (DEBUG level)
- Timer state changes: Suppressed (DEBUG level)

**Always Logged (ERROR/WARN):**
- Missing meet state errors
- JSON marshaling failures
- Timer reset attempts on inactive timers
- Timer operation failures with full context

**Development Mode:**
- Complete timer lifecycle logging
- Timer ID tracking for debugging
- Full context including duration and end times
- Detailed timer state transitions

### Log Analysis

**Search for errors:**
```bash
grep "ERROR" logs/*.log
```

**Parse JSON logs:**
```bash
cat logs/*.log | jq '.level, .message, .context'
```

**Filter by component:**
```bash
cat logs/*.log | jq 'select(.context.component == "websocket")'
```

**Filter by error category:**
```bash
cat logs/*.log | jq 'select(.context.errorCategory == "authentication")'
```

**Filter by error severity:**
```bash
cat logs/*.log | jq 'select(.context.errorSeverity == "critical" or .context.errorSeverity == "high")'
```

**Find specific error codes:**
```bash
cat logs/*.log | jq 'select(.context.errorCode | startswith("AUTH_"))'
```

**WebSocket issues only (production):**
```bash
cat logs/*.log | jq 'select(.context.component == "websocket" and (.level == "ERROR" or .level == "WARN"))'
```

**Connection problems:**
```bash
cat logs/*.log | jq 'select(.context.action | contains("connection"))'
```

**Find authentication issues:**
```bash
cat logs/*.log | jq 'select(.context.component == "authentication" and .level == "ERROR")'
```

**Timer issues only (production):**
```bash
cat logs/*.log | jq 'select(.context.component == "timer" and (.level == "ERROR" or .level == "WARN"))'
```

**Track timer operations:**
```bash
cat logs/*.log | jq 'select(.context.component == "timer" and .context.timerType != "")'
```

### Common Issues and Log Patterns

**Authentication Failures:**
```json
{
  "level": "WARN",
  "message": "Authentication failed: Invalid username or password",
  "context": {
    "component": "authentication",
    "action": "login_failed",
    "username": "referee1",
    "ipAddress": "192.168.1.100",
    "errorCategory": "authentication",
    "errorSeverity": "high",
    "errorCode": "AUTH_001",
    "failureReason": "invalid_credentials"
  }
}
```

**WebSocket Connection Problems:**
```json
{
  "level": "ERROR",
  "message": "WebSocket upgrade failed",
  "context": {
    "component": "websocket",
    "action": "connection_upgrade_failed",
    "meetName": "Test Meet",
    "refereeId": "left",
    "remoteAddr": "192.168.1.100",
    "errorCategory": "websocket",
    "errorSeverity": "medium",
    "errorCode": "WS_001"
  },
  "error": "connection timeout"
}
```

**WebSocket Message Delivery Issues:**
```json
{
  "level": "WARN",
  "context": {
    "component": "websocket",
    "action": "message_write_failed",
    "meetName": "Test Meet",
    "refereeId": "center",
    "remoteAddr": "192.168.1.101"
  },
  "error": "broken pipe"
}
```

**Timer Operation Failures:**
```json
{
  "level": "ERROR",
  "context": {
    "component": "timer",
    "action": "meet_state_missing",
    "meetName": "Test Meet",
    "timerType": "startTimer",
    "timerId": ""
  },
  "message": "Meet state not found for timer action"
}
```

**Timer Reset Issues:**
```json
{
  "level": "WARN",
  "context": {
    "component": "timer",
    "action": "reset_inactive_timer",
    "meetName": "Test Meet",
    "timerType": "platform_ready",
    "timerId": "123"
  },
  "message": "No active platform ready timer to reset"
}
```

**Authentication Failures:**
```json
{
  "level": "ERROR",
  "context": {
    "component": "authentication",
    "action": "login_failed",
    "username": "referee1",
    "ipAddress": "192.168.1.100"
  }
}
```

**Position Conflicts:**
```json
{
  "level": "WARN",
  "message": "Position occupancy conflict: left seat already taken",
  "context": {
    "component": "position",
    "action": "set_position",
    "position": "left",
    "meetName": "Test Meet",
    "errorCategory": "position",
    "errorSeverity": "medium",
    "errorCode": "POS_002",
    "currentOccupant": "referee1@example.com",
    "conflictType": "position_occupied"
  }
}
```

## CloudWatch Integration

For AWS deployments, logs are automatically sent to CloudWatch:

### Log Groups
- Application logs: `/aws/ecs/referee-lights`
- Container logs include both application and system logs

### Useful CloudWatch Queries
```
fields @timestamp, level, message, context.component, context.errorCategory
| filter level = "ERROR"
| sort @timestamp desc
```

```
fields @timestamp, message, context.meetName, context.refereeId, context.errorCode
| filter context.errorCategory = "websocket"
| filter level = "ERROR" or level = "WARN"
```

```
fields @timestamp, message, context.errorCode, context.errorSeverity
| filter context.errorSeverity = "critical" or context.errorSeverity = "high"
| sort @timestamp desc
```

```
fields @timestamp, message, context.errorCode, context.failureReason
| filter context.errorCategory = "authentication"
| stats count() by context.errorCode
```

```
fields @timestamp, message, context.meetName, context.position, context.conflictType
| filter context.errorCategory = "position"
| filter level = "ERROR" or level = "WARN"
| sort @timestamp desc
```

## Troubleshooting

### Common Issues

**Logs not appearing:**
- Check ENV and LOG_LEVEL settings
- Production mode suppresses DEBUG and INFO by default
- Use `LOG_LEVEL=DEBUG` to override

**Performance concerns:**
- Use conditional logging for expensive operations
- Check `logger.ShouldLog()` before costly string operations
- Disabled levels have minimal overhead

**JSON parsing errors:**
- System automatically falls back to simple logging
- Original message is preserved
- Marshaling error is logged for debugging

### Debug Configuration
```go
// Check current logger configuration
globalLogger := logger.GetGlobalLogger()
if globalLogger != nil {
    fmt.Printf("Current level: %s\n", globalLogger.GetLevel().String())
    fmt.Printf("Should log DEBUG: %v\n", logger.ShouldLog(logger.DEBUG))
}
```

## Testing

### Running Logger Tests

The logging system includes comprehensive unit and integration tests:

```bash
# Run unit tests
go test -v ./logger/

# Run integration tests (comprehensive environment testing)
go test -v -tags=integration ./logger/

# Run all tests with coverage
go test -coverprofile=coverage.out ./logger/
go tool cover -func=coverage.out
```

### Integration Test Coverage

The integration tests validate:

- **Environment-based configuration**: All ENV and LOG_LEVEL combinations
- **File logging behavior**: Production vs test mode file creation
- **Log level filtering**: Proper message filtering based on configuration
- **Thread safety**: Concurrent access to logging configuration
- **Edge cases**: Case sensitivity, whitespace handling, invalid values
- **Performance**: Log file size and overhead measurement
- **Complete workflow**: End-to-end logging system operation

### Test Environments

Integration tests cover these scenarios:

| Environment | LOG_LEVEL | Expected Behavior           |
|-------------|-----------|-----------------------------|
| production  | (none)    | WARN level, file logging    |
| development | (none)    | DEBUG level, file logging   |
| test        | (none)    | WARN level, no file logging |
| production  | DEBUG     | DEBUG level override        |
| invalid     | (none)    | Fallback to WARN level      |

## Best Practices

### For Developers
1. Use structured logging for new code
2. Include relevant context (meet name, referee ID, etc.)
3. Use appropriate log levels (ERROR for failures, WARN for issues, INFO for important events)
4. Check `ShouldLog()` before expensive operations
5. Provide meaningful error messages with context
6. Run integration tests when modifying logging behavior

### For Operations
1. Use production mode in live environments
2. Monitor ERROR and WARN level logs
3. Set up CloudWatch alarms for error rates
4. Use structured log parsing for analysis
5. Clean up old log files between events
6. Validate logging configuration in staging environments

### For Debugging
1. Use development mode for comprehensive logging
2. Enable DEBUG level for detailed troubleshooting
3. Use LOG_LEVEL override for production debugging
4. Search logs by component and action
5. Correlate logs using meet name and referee ID
6. Use integration tests to verify logging behavior

## Migration Guide

### From Legacy Logging
Existing code using `logger.Info.Printf()` continues to work without changes. For new code, prefer structured logging:

```go
// Old way (still works)
logger.Error.Printf("WebSocket error for %s: %v", refereeID, err)

// New way (recommended)
context := logger.NewWebSocketContext("error", meetName, refereeID, remoteAddr)
context = logger.AddError(context, err)
logger.LogErrorWithContext(context, "WebSocket connection error")
```

### Performance Migration
Replace expensive logging operations:

```go
// Before
logger.Debug.Printf("State: %s", expensiveStateFunction())

// After
if logger.ShouldLog(logger.DEBUG) {
    logger.LogDebugWithContext(context, "State: %s", expensiveStateFunction())
}
```

This guide provides comprehensive information for using the RefLights logging system effectively in both development and production environments.

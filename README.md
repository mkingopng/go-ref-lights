# RefLights

## Overview
RefLights is a referee lighting system designed for powerlifting competitions. It provides a real-time, synchronized referee light system that enables fair and efficient judging for lifters and meet directors. The application is now **deployed to AWS** and is currently in **production**.

## Features
- **Multi-meet functionality**: Supports multiple competitions running in parallel.
- **Single login enforcement**: Prevents users from logging in from multiple devices simultaneously.
- **Real-time referee decisions**: Judges can submit lift decisions, which are instantly reflected on the lighting system.
- **WebSocket communication**: Ensures seamless real-time updates for referee actions.
- **Dynamic meet and position assignment**: Referees can claim and vacate positions easily.
- **Platform ready & next attempt timers**: Countdown timers for lifter readiness and next attempts.
- **Secure authentication**: Password-based login with bcrypt hashing.
- **AWS deployment**: Hosted using **AWS Fargate, ECS, ALB, and CloudWatch** for monitoring.
- **Optimized logging system**: Environment-based log levels with structured JSON logging for production monitoring and debugging.

## Installation (Local Development)
### Prerequisites
- Go (>= 1.20)
- Python (>= 3.10) with Poetry
- AWS CDK (for deployment)
- Docker (for containerized deployment)
- Node.js & NPM (for frontend dependencies, if needed)

### Build and Run
1. Clone the repository:
   ```bash
   git clone https://github.com/yourrepo/referee-lights.git
   cd referee-lights
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   poetry install
   ```
3. Build the application:
   ```bash
   go build ./...
   ```
4. Run locally:
   ```bash
   go run main.go
   ```

## Running Tests
To execute all tests, run:
```bash
go test -v ./...
```

## Deployment to AWS
1. Ensure AWS CLI is configured.
2. Deploy using AWS CDK:
   ```bash
   cdk deploy
   ```
3. The service is accessible at:
   ```
   https://referee-lights.michaelkingston.com.au
   ```

## Usage
### Logging in
1. Select a meet from the list.
2. Enter provided referee credentials.
3. Claim a referee position (Left, Center, Right).
4. Use the interface to submit lift decisions.

### Referee Lights Interface
- **White Button**: Signals a good lift.
- **Red Button**: Signals a failed lift.
- **Platform Ready Timer**: Initiated for lifter readiness.
- **Vacate Position**: Frees up a referee slot.

## Logging System

RefLights features an optimized logging system designed for production reliability and development debugging:

### Environment-Based Log Levels
- **Production** (`ENV=production`): ERROR, WARN, and critical INFO messages only - optimized for performance
- **Development** (`ENV=development`): All log levels including DEBUG for comprehensive debugging
- **Test** (`ENV=test`): ERROR and WARN messages only for clean test output

### Structured Logging
Production logs use JSON format with rich context for easy parsing and analysis:
```json
{
  "timestamp": "2023-01-01T12:00:00Z",
  "level": "ERROR",
  "message": "WebSocket connection upgrade failed",
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
  "source": "connection.go:123",
  "error": "connection timeout"
}
```

### Error Categorization
The logging system includes comprehensive error categorization for better monitoring and troubleshooting:
- **Categories**: Authentication, WebSocket, Timer, Position, Validation, System, and more
- **Severity Levels**: Critical, High, Medium, Low for proper alerting
- **Error Codes**: Systematic codes (AUTH_001, WS_001, POS_002) for tracking specific issues
- **Rich Context**: IP addresses, user agents, meet details, and failure reasons

### Performance Optimizations
- **WebSocket logging optimized**: Routine operations (heartbeats, message processing) suppressed in production
- **Timer logging optimized**: Routine countdown updates suppressed in production
- **HTTP request logging**: Successful requests suppressed, errors preserved with context
- **Conditional logging**: Expensive operations only executed when logs will be written
- **Structured context**: Rich debugging information without performance overhead

### Configuration
Set logging level via environment variables:
```bash
ENV=production ./go-ref-lights          # Production logging (WARN level)
ENV=development ./go-ref-lights         # Verbose logging (DEBUG level)
ENV=test ./go-ref-lights                # Test logging (WARN level, no files)
LOG_LEVEL=DEBUG ./go-ref-lights         # Override specific level
```

### Testing
The logging system includes comprehensive tests:
```bash
go test -v ./logger/                    # Unit tests
go test -v -tags=integration ./logger/  # Integration tests
```

## Future Enhancements
- Integration with OpenLifter for automated lift decisions.

## Contributing
1. Fork the repository.
2. Create a feature branch.
3. Make your changes and test thoroughly.
4. Submit a pull request.

## Contact
For any issues or inquiries, please contact **michael.kenneth.kingston@gmail.com**.

```bash
go test -v -tags=unit ./...
```

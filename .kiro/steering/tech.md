# Technology Stack

## Backend
- **Go 1.23+** - Primary backend language
- **Gin** - Web framework for HTTP routing and middleware
- **Gorilla WebSocket** - Real-time communication
- **bcrypt** - Password hashing
- **Gin Sessions** - Session management with cookie store

## Frontend
- **HTML templates** - Server-side rendering
- **Static assets** - CSS, JavaScript, images served from `/static`
- **WebSocket client** - Real-time updates

## Infrastructure & Deployment
- **AWS Fargate/ECS** - Container orchestration
- **Application Load Balancer** - HTTPS termination and routing
- **CloudWatch** - Logging and monitoring
- **AWS CDK (Python)** - Infrastructure as Code
- **Docker** - Containerization

## Development Tools
- **Poetry** - Python dependency management (for CDK)
- **golangci-lint** - Go linting
- **gofmt/goimports** - Code formatting
- **pytest** - Python testing (CDK tests)
- **pre-commit** - Git hooks

## Common Commands

### Development
```bash
# Install dependencies
go mod tidy
poetry install

# Run locally
go run cmd/referee-lights/main.go

# Format code
gofmt -l -w .
goimports -l -w .

# Lint
golangci-lint run --timeout 2m

# Run all checks
./check.sh
```

### Testing
```bash
# Run all Go tests
go test -v ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run unit tests only
go test -v -tags=unit ./...

# Run Python tests (CDK)
pytest
```

### Build & Deploy
```bash
# Build binary
go build -o go-ref-lights ./cmd/referee-lights

# Build Docker image
docker build -t referee-lights .

# Deploy to AWS
cdk deploy
```

## Key Technical Requirements
- **Concurrency**: Go's goroutines handle multiple meets simultaneously
- **Real-time Communication**: WebSocket connections with heartbeat monitoring
- **Ephemeral State**: Decisions cleared on platform ready, no persistent storage
- **Session Management**: Position occupancy with exclusive access control
- **Timer Management**: Dual timer system (platform ready + multiple next attempt timers)
- **Mobile Optimization**: Responsive design for referee mobile interfaces

## Environment Variables
- `ENV` - Environment mode (production/development/test) - Controls logging levels and application behavior
- `LOG_LEVEL` - Optional logging level override (DEBUG/INFO/WARN/ERROR)
- `APP_HOST` - Server host (default: localhost/0.0.0.0)
- `APP_PORT` - Server port (default: 8080)
- `MEETS_PATH` - Path to meets configuration
- `MEET_CREDS_PATH` - Path to credentials file
- `APPLICATION_URL` - Base URL for QR code generation
- `WEBSOCKET_URL` - WebSocket endpoint for real-time communication

### Logging Configuration
- **Production** (`ENV=production`): ERROR, WARN, and critical INFO messages only
- **Development** (`ENV=development`): All log levels including DEBUG for comprehensive debugging
- **Test** (`ENV=test`): ERROR and WARN messages only for clean test output
- Use `LOG_LEVEL` to override environment defaults for troubleshooting

## Performance Considerations
- Built in Go for speed and concurrent meet handling
- WebSocket heartbeat prevents connection drops during phone multitasking
- Ephemeral decision storage prevents memory accumulation
- Optimized for mobile referee interfaces with minimal latency

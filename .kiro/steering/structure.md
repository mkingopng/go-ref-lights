# Project Structure

## Go Application Layout

### Core Directories
- `cmd/referee-lights/` - Main application entry point
- `controllers/` - HTTP request handlers and business logic
- `middleware/` - Authentication, authorization, and request processing
- `models/` - Data structures and domain models
- `services/` - Business logic and external integrations
- `websocket/` - Real-time communication handlers
- `logger/` - Centralized logging utilities

### Frontend Assets
- `templates/` - HTML templates for server-side rendering
- `static/` - CSS, JavaScript, images, and other static assets
  - `static/css/` - Stylesheets
  - `static/js/` - Client-side JavaScript
  - `static/images/` - Images, icons, logos

### Configuration & Data
- `config/` - JSON configuration files for meets and credentials
- `logs/` - Application and meet-specific log files organized by event

### Infrastructure
- `referee_lights_cdk/` - AWS CDK Python stack definitions
- `scripts/` - Build and deployment scripts
- `tests/` - Integration and end-to-end tests

### Analysis & Utilities
- `log_analysis/` - Python scripts for log processing and analysis
- `docs/` - Documentation and user manuals

## File Naming Conventions

### Go Files
- Controllers: `*_controller.go` with corresponding `*_controller_test.go`
- Services: `*_service.go` with corresponding `*_service_test.go`
- Models: Descriptive names like `meet.go`, `meet_test.go`
- Tests: Always suffix with `_test.go`

### Templates
- HTML templates use descriptive names: `admin.html`, `login.html`, `lights.html`
- Shared components: `header.html` for common elements

### Configuration
- JSON files for runtime configuration: `meets.json`, `meet_creds.json`
- YAML for application config: `config.yaml`

## Architecture Patterns

### MVC-Style Organization
- **Controllers** handle HTTP requests and responses
- **Services** contain business logic and external integrations
- **Models** define data structures
- **Middleware** handles cross-cutting concerns (auth, logging)

### Dependency Injection
- Services are injected into controllers via constructors
- Promotes testability and loose coupling

### WebSocket Management
- Centralized WebSocket handling in `websocket/` package
- Heartbeat monitoring to maintain referee connections
- Real-time decision broadcasting to all connected clients
- Position-based connection management

### State Management
- **Ephemeral Decisions**: Cleared on platform ready button press
- **Position Occupancy**: Persistent until referee vacates or disconnects
- **Timer State**: Platform ready clears decisions, next attempt timers persist
- **Session Management**: Cookie-based with heartbeat monitoring

### Concurrency Patterns
- Goroutines for handling multiple concurrent meets
- Mutex locks for position occupancy management
- Channel-based communication for timer coordination
- Thread-safe decision collection and clearing

### Logging Strategy
- **Environment-based log levels**: Production (ERROR/WARN), Development (all levels), Test (ERROR/WARN)
- **Structured JSON logging**: Rich context with meet name, referee ID, component, and action details
- **Performance optimized**: Conditional logging prevents expensive operations when disabled
- **Contextual categories**: WebSocket, Timer, Authentication, Position, HTTP, and System operations
- **Meet-specific log separation**: Logs include meet context for debugging specific competitions
- **Centralized logger initialization**: Global logger with backward compatibility for existing code

## Testing Organization
- Unit tests alongside source files (`*_test.go`)
- Integration tests in `tests/` directory
- Test helpers in `*_test_helpers.go` files
- Mock services for testing isolation

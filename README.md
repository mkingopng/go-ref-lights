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
The project includes several types of tests:

### Unit Tests
For testing individual components:
```bash
go test -v ./...
```

### Integration Tests
For testing the interaction between components:
```bash
go test -v -tags=integration ./tests
```

### Remote Tests
For testing against the production AWS environment:
```bash
# Update credentials in tests/remote_simulation_test.go first
go test -v -tags=remote -run=TestFullMeetSimulation ./tests
```

These remote tests include:

1. **Full Meet Simulation (TestFullMeetSimulation)**
   - Simulates a complete powerlifting competition with 100 lifters
   - Each lifter performs 9 attempts (3 squats, 3 bench, 3 deadlift)
   - ~20 seconds between attempts with random variation
   - Periodically simulates network disconnections and reconnections
   - Includes referee position changes and admin actions

2. **Network Resilience Testing (TestRefereeNetworkIssues)**
   - Rapid disconnection and reconnection of referees
   - Verifies system maintains state and functionality

3. **Load Testing (TestHighLoad)**
   - Creates multiple simultaneous connections
   - Tests broadcast performance
   - Verifies system handles many concurrent users

Use these tests with caution as they interact with the production environment.
The constants in `remote_simulation_test.go` should be configured appropriately
before running.

### Test Structure
- `tests/` directory contains both unit and integration tests
- Integration tests use the `integration` build tag
- Tests can be run against the test_mule meet configuration for consistent results
- The test infrastructure includes helpers for:
  - WebSocket testing with message capture
  - Authentication bypass for testing
  - Simulating referee connections and interactions

For running specific tests:
```bash
# Run a specific test
go test -v -tags=integration -run=TestRefereeFlow ./tests

# Run tests with race detection
go test -v -tags=integration -race ./tests
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

## Future Enhancements
- Integration with OpenLifter for automated lift decisions.

## Contributing
1. Fork the repository.
2. Create a feature branch.
3. Make your changes and test thoroughly.
4. Submit a pull request.

## Contact
For any issues or inquiries, please contact **michael.kenneth.kingston@gmail.com**.

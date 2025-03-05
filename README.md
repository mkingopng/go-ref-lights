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

## Future Enhancements
- Improved CI/CD pipeline.
- Enhanced UI/UX for better referee experience.
- Extended analytics and performance tracking.
- Integration with OpenLifter for automated lift decisions.

## Contributing
1. Fork the repository.
2. Create a feature branch.
3. Make your changes and test thoroughly.
4. Submit a pull request.

## Contact
For any issues or inquiries, please contact **michael.kenneth.kingston@gmail.com**.

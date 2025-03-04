compile
```bash
go build ./...
```

run app
```bash
go run main.go
```

run all tests
```bash
go test -v ./...
```

---
# Best Practices for Maintaining Unit Tests**
Since we’re writing **a large number of tests**, here are **best practices** to 
ensure long-term maintainability:

### Follow the Given-When-Then Structure
Write tests in a **clear, structured way**:
✅ **Keeps tests readable** and **maintains consistency**.

### Use Mocks for External Dependencies
- **Use mock services** instead of real implementations.
- **Minimize reliance on actual databases, API calls, or network connections**.
- Speeds up tests and reduces flakiness

### Separate Unit vs. Integration vs. Smoke Tests
- **Unit Tests** → Test individual functions in isolation.
- **Integration Tests** → Test how multiple components work together.
- **Smoke Tests** → Run a minimal test to check if the system starts up without errors.
- test under load to ensure that the system performs well under high traffic
- **Avoid confusion** between different test types.

### Run Tests in CI/CD Pipelines**
- **Ensure all tests run automatically** before merging code.
- **Use GitHub Actions, GitLab CI, or Jenkins** to automate testing.
- **Catches issues before they reach production**.

### Keep Tests Fast
- **Optimize tests** to avoid slow performance.
- **Mock external services** instead of making real calls.
- **Use table-driven tests** to avoid redundant code.
- Encourages running tests frequently

### Write Self-Contained Tests
- **Each test should be independent**.
- **Tests should NOT rely on global state** (e.g., shared session data, database entries).
- **Prevents flaky test failures**.

### Use Meaningful Test Names

### Ensure Clean Test Data
- **Reset mock services after each test**.
- **Use `defer` to clean up test artifacts**.
- **Prevents one test from interfering with another**.

### Prioritize High-Coverage Areas
- **Start with testing critical business logic**.
- **Cover edge cases (invalid input, errors, permissions, etc.).**
- **Focus on high-risk areas first**.


# tasks

1. Unit tests
    - broadcast_test.go (Websocket broadcast testing)
    - connection_test.go (Real-time connection handling)
    - meet_state_test.go (State transitions in meets)
    - timer_test.go (Timekeeping accuracy)
    - main_test.go (Bootstrap and config validation)

2. Integration tests
	- Login + Session Management → Ensure login persists a session.
    - Meet Creation + Role Assignment → Verify a user can create a meet and 
	  assign roles.
    - Position Claiming + Websocket Broadcast → User claims a position → UI 
	  updates via broadcast.
    - Referee Actions + State Updates → A referee gives a lift decision → 
	  The state updates correctly.

3. Smoke tests
4. Load tests
5. update precommit hooks
	- golangci-lint (code quality)
    - gofmt (formatting)
    - govet (detect common issues)
    - prettier for frontend parts (if applicable)

6. CI/CD: Automate tests and deploys via GitHub Actions.
	- Run unit tests.
    - Run integration tests.
    - Run smoke tests.
    - Deploy if all pass.

7. improved formatting

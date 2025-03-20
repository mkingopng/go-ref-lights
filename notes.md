compile
```bash
go build ./...
```

run app locally
```bash
ENV=development go run main.go
```

or run from docker:
```bash
docker run -e ENV=development -p 8080:8080 referee-lights
```

run all tests
```bash
go test -v ./...
```

run unit tests
```bash
go test -v -tags=unit ./...
```

run unit tests in a specific directory
```bash
go test -v -tags=unit ./websocket
```

test coverage
```bash
go test -v -tags=unit -coverprofile=cover.out ./controllers
```
or
```bash
go test -v -tags=unit -cover ./controllers
```

run precommit hooks:
```bash
pre-commit run --all-files
```

before committing, run:
```bash
poetry run pre-commit run --all-files
go test -v -tags=unit ./...
```

run integration tests
```bash
go test -v -tags=integration ./...
```

run integration tests in a specific directory
```bash
go test -v -tags=integration ./websocket
```


go's race detector:
```bash
go test -race ./...
```

---
# Best Practices for Maintaining Unit Tests**
Since we’re writing **a large number of tests**, here are **best practices** to
ensure long-term maintainability:

### Follow the Given-When-Then Structure
Write tests in a **clear, structured way**:
- **Keeps tests readable** and **maintains consistency**.

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

### Run Tests in CI/CD Pipelines
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
- **Tests should NOT rely on global state** (e.g. shared session data, database entries).
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
1. Integration tests
    - Login + Session Management → Ensure login persists a session.
    - Meet Creation + Role Assignment → Verify a user can create a meet and
      assign roles.
    - Position Claiming + Websocket Broadcast → User claims a position → UI
      updates via broadcast.
    - Referee Actions + State Updates → A referee gives a lift decision →
      The state updates correctly.
    - broadcast_integration_test.go
    - auth_integration_test.go
    - page_integration_test.go
    - position_integration_tst.go
    - api_integration_test.go

#  Test Upgrades
1. Smoke tests
2. Load tests (K6)
3. update precommit hooks
    - golangci-lint (code quality)
    - gofmt (formatting)
    - govet (detect common issues)
    - prettier for frontend parts (if applicable)
4. CI/CD: Automate tests and deploys via GitHub Actions.
    - Run unit tests.
    - Run integration tests.
    - Run smoke tests.
    - Deploy if all pass.
5. improved formatting
6. admin page
7. logout from anywhere
8. reset meet state
9. sudo page

---

# Improvements and bugs
1. When a referee is in position, if they look at another page or app on
   their phone, it can cause the health check to fail. Ideally, the
   healthy connection should be maintained regardless of what the user does
   on their phone, as long as the browser window is open. However, if, for
   whatever reason, the connection becomes unhealthy, the user should be
   able to refresh the page and rejoin the meet without any issues. This is
   not always happening as it should. In many cases, when I hit refresh, I get
   a 404 error. Refer to the screenshots attached
2. There are many mechanisms for the referee to vacate their position,
   or log out however only the admin panel is working correctly. The other
   mechanisms don't work correctly.
   - On the referee screen, there is a button called vacate position. see
     attached image. When this button is pressed it should vacate the position
     and take the user back to /index. However, it does not do this. The
     user gets a 404 error. Refer to the screenshots attached. This is not
     the correct behaviour. We need to correct this. What should happen is a
     redirect to /index.
   - The referee screen has a button called logout. When this button is
     pressed, the user should be logged out and taken back to the login
     page. This doesn't happen. The user gets a 404 error. Refer to the
     screenshot. we need to fix this. What should happen is the user is
     logged out from that when the button is pressed, the referee is taken
     back to /index where they can take on a new position.
   - the referee page has a button called Home. we should remove this button
     as it is not needed.
   - When the referee logs out the meet persists
   - When the admin logs out the meet resets
3. There needs to be a super-user or sudo role who can log into
   any meet and take control as a fall-back position. This is not yet built in
   to the functionality. Not sure how to implement this yet.
4. Need to implement dynamic logo. Most meets currently use the APL logo,
   however more and more meets will use specific logos. In anticipation of
   this i have included logo in the meet.go data structure but it is not
   used anywhere yet. Need to implement this.
5. Review the CDK code and optimise. Consider how to scale to zero

-------

# CDK upgrades
- scheduling
- scale to zero

```python

```

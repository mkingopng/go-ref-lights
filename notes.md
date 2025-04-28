run app locally
```bash
go run cmd/referee-lights/main.go
```

or run from docker:
```bash
docker run -e ENV=development -p 8080:8080 referee-lights
```
-------------------------------

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

```bash
go test -v -tags=unit ./controllers
```

------------------------
# test coverage

test coverage of unit packages
```bash
go test -v -tags=unit -cover ./...
```

test coverage of a specific package
```bash
go test -v -tags=unit -coverprofile=cover.out ./controllers
```
or
```bash
go test -v -tags=unit -cover ./controllers
```

-----------------------
# precommit hooks

run precommit hooks:
```bash
pre-commit run --all-files
```

before committing, run:
```bash
poetry run pre-commit run --all-files
go test -v -tags=unit ./...
```

-----------------------
# integration tests

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

load testing & save logs to JSON
```bash
k6 run --out json=test/k6/results.json tests/k6/script.js
```

---------------------------------------
# Best Practices for Maintaining Unit Tests
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

-----------------
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

-------

# CDK upgrades
- scheduling
- scale to zero

# some simple real tests

Below is a concise list of scenarios you can methodically run through in
the remaining time, covering each major flow. If all (or most) of these
work as expected, you’ll be in a good place for your presentation.

1. Single-User (Referee) Flows

   Login & Seat Claim
   Log in as a referee for “Meet A.”
   From the “Positions” page, choose a seat (e.g., Center).
   Verify you see “Connected.”

   Refresh & Re-Claim
   Hit Refresh in the browser (or close and reopen the tab if feasible).
   Confirm you are still recognized as occupant of that seat (no “Seat Taken” message).

   Vacate from the Referee Screen
   Click the “Vacate Position” button.
   Confirm you are redirected to /index (not a 404), and the seat is freed.
   (Optional) Re-claim the seat to verify it works.

   Logout from the Referee Screen
   Click the “Logout” button.
   Confirm you’re redirected to /index (or /login, depending on your choice) and the seat is freed.

   Phone Sleep / App Switch
   On your phone, open another app or let the phone sleep ~15 seconds.
   Come back to the browser, confirm you are still recognized (or reloaded) in that seat.
   If the connection was dropped, try refreshing. You should reclaim your seat.

2. Admin (Meet Director) Flows

   **Admin Panel**
   Log in as an Admin, open /admin?meet=YourMeetName.
   Verify that you see occupancy states for left, center, right (including your phone’s occupant, if claimed).

   **Force Vacate**
   With your phone in a seat, click “Force Vacate” in the Admin panel.
   Verify the phone sees it was disconnected (or occupant is forcibly removed), and the seat is free in the Admin panel.

   **Reset Instance**
   “Reset Instance” from the Admin panel to clear everything.
   Confirm the seats become vacant and the phone occupant is kicked out.

   **Switch from Admin to Phone**
   If you have two devices, remain Admin on one, and phone as referee on the other, watch occupancy updates in real time.

3. Multiple-User Collision
   User A claims “Center.”
   User B logs in to the same meet and tries to claim “Center.”
   Expected result: “Seat is already taken” for B.
   User A refreshes, confirm they keep “Center.”
   User B tries a different seat, or wait for Admin to “Force Vacate” A’s seat, then claim “Center.”

4. QR Code Flow (If relevant)
   Generate a QR Code from the admin or the UI (center, left, right).
   Scan with your phone.
   If you have the route that automatically assigns “AnonymousReferee,” test that as well.
   Confirm it joins the seat, and you see healthy connectivity.
   Phone Sleep again, re-check connectivity upon returning.

5. Verify Logs
   CloudWatch Logs: watch the container logs to ensure no big errors appear.
   Confirm you see the typical “SetPosition” or “VacatePosition” messages.
   If something 404s, you’ll notice it in the logs.

## Additional Tips
- Use More Than One Browser/Device if possible. Having your phone and one
  desktop browser helps you see real concurrency.
- Test Each Flow Once in normal usage, then break it (like phone sleeps).
  This ensures you’ve tried each path from start to finish.
- Keep your Admin session open in a separate tab to see real-time occupant
  changes as you test from the phone.
---------------------------

## 4. General housekeeping

   **A. Possibly unify your environment checks**
   You have checks for `env == "production"`, `env == "test"`, etc., scattered in a few places. Usually that’s fine, but if you see repeated code for “set gin mode to release if production, else test,” you can put that in a single function.

   **B. Timestamps and logs**
   You’re storing a `LastUpdated time.Time` in each `Occupancy`, but not always using it. If you truly need it for debugging or for cleaning up old meets, that’s fine—but if it’s never read, you might remove it.

   **C. Sudo/superuser code**
   You do have basic routes for `SudoController`, “force vacate any meet,” “force logout meet director,” etc. That’s helpful, but if you’re going to rely on this path in production, it’s worth adding better error handling (for example, verifying that the meet exists before you reset it). Right now, some of that code does minimal checks—maybe that’s enough, maybe not.

   **D. Testing**
   You’ve mentioned you want a more comprehensive test suite. Right now, the code has good structure for testing (especially with all those injected functions like `loadMeetCredsFunc`), but it’s easy to forget that you can remove some of the dead or placeholder code once you finalize the approach.

---

### Summary of key “next steps”
1. Make sure all docstrings and comments **reflect the actual code** after you unify the logic for meeting selection, seat vacancy, etc.
2. sudo code & functionality
3. testing suite
4. CDK code improvements
5. fix all remaining warnings, TODO and FIX_ME
6. fix samsung issues
7. prune unused code

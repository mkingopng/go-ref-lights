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
----------
https://referee-lights.michaelkingston.com.au/referee/Lift3%20Heavy%20Hitters/right
https://referee-lights.michaelkingston.com.au/referee/Lift3%20Heavy%20Hitters/center
https://referee-lights.michaelkingston.com.au/referee/Lift3%20Heavy%20%Hitters/left
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

----

Alright — I've gone through everything:
✅ Your uploaded log file,
✅ All your current project files (`main.go`, controllers, services, websocket handlers, etc.).

---

### **Here’s my structured review:**

### 1. Identified Problems (from logs + codebase review)

| # | Problem | Cause | Impact |
|:-|:---|:---|:---|
| 1 | **Broadcast channel (`broadcast`) is overloaded** | Too many messages queued into a single `broadcast` channel; slow consumers. | Dropped messages, stale or missing updates on lights page (`lights.html`). |
| 2 | **No backpressure or overflow handling on `broadcast` channel** | `broadcast` is unbuffered (`chan []byte`), and writers never check if channel is full. | System degrades silently under load, referees "look connected" but lights freeze. |
| 3 | **Heartbeat system (refereeSessions map) is separate from websocket `connections` map** | Heartbeat checks session IDs, but WebSocket logic tracks connections differently. | Referees stay "green" even after losing WebSocket connection. |
| 4 | **Timers (Platform Ready, Next Attempt) are too trusting of connection health** | They assume WebSocket clients receive updates reliably. | Timers can become desynchronized from referee screens if WS messages drop. |
| 5 | **Force Vacate sometimes fails to clean up associated WS connections** | ForceVacate removes occupancy and ActiveUsers, but does not forcibly close WS. | Zombie WebSocket sessions linger after vacate. |
| 6 | **Occasional JSON marshal errors not handled defensively** | Broadcast code assumes marshaling always succeeds. | Some messages fail to broadcast without retries. |
| 7 | **Vacate Position redirects poorly (404)** | Vacate endpoint `/position/vacate` redirects to `/logout?reason=vacate`, but session or state isn't fully cleaned up before redirect. | User gets logged out but page renders incorrectly. |
| 8 | **Lack of flood protection in `readPump`** | Unlimited reads from clients are allowed. | Malicious or buggy clients can flood server with junk messages. |
| 9 | **CloudWatch metrics for connections not updated accurately** | Metrics submitted for total WS connections, but not tied tightly enough to lifecycle (open/close). | Cloud monitoring undercounts/overcounts. |
| 10 | **No central system health dashboard inside app yet** | Only CloudWatch external metrics; no admin panel indicator for platform ready timers, next attempt timers, referee health in-app. | Admins can't easily detect problems during meet. |

---

### 2. Action List to Fix These Problems

| Priority | Action | How | Level of Effort |
|:-|:---|:---|:---|
| ⭐ | **Make `broadcast` channel buffered** | e.g., `broadcast = make(chan []byte, 500)` | Low |
| ⭐ | **Add overflow detection for `broadcast`** | If channel full, log and drop message clearly. | Medium |
| ⭐ | **Unify heartbeat and WebSocket tracking** | Make heartbeat use WebSocket `connections` map. | High |
| ⭐ | **Auto-close WebSocket on vacate/logout** | Modify ForceVacate and Logout to forcibly close WS. | Medium |
| ⭐ | **Fix Vacate redirect issue** | Ensure session is cleared properly before `/logout?reason=vacate` redirect. | Low |
| ⭐ | **Add read limit and rate limiting to `readPump`** | Set max message size, simple rate limiter. | Medium |
|  | **Retry marshal errors in broadcasts** | Wrap marshals in retry or fallback. | Medium |
|  | **Update CloudWatch metrics on WS open/close** | Hook into `registerConnection` and `unregisterConnection`. | Medium |
|  | **Add Admin Panel Dashboard (heartbeat + timers + queue size)** | Show: active referees, timers, broadcast queue depth. | High |
|  | **Timer resiliency (stale timer detection)** | Detect and kill stuck timers. | Medium |

---

### 📋 **Immediate To-Do Shortlist (First Moves)**

1. In `globals.go`:
   ```go
   var broadcast = make(chan []byte, 500) // add buffer
   ```

2. In `broadcast.go`:
   Before sending:
   ```go
   select {
       case broadcast <- msg:
       default:
           logger.Warn.Println("[Broadcast] Broadcast channel full! Dropping message")
   }
   ```

3. In `connection.go`:
   - Add `SetReadLimit(512)` on WebSocket connection right after upgrading (protect against flood).
   - Hook up `unregisterConnection(c)` inside `ForceVacate`, `Logout`, etc.

4. In `page_controller.go` (`VacatePosition`):
   - Before redirect to `/logout`, explicitly clear session or force redirect via JS if server-side redirect keeps session dirty.

---

### ⚡ If you like, I can break this into even more detailed PR-ready changes (one per task) and show:

- File-by-file: **where to insert**, **what to delete**, **what to replace**, **full code blocks with comments**.

Would you like me to build that next? 🚀
(Feels like it would really help you work cleanly through it!)

----

Below is a **pull-request-ready game-plan** that you can follow commit-by-commit.
For every task I give:

* **Why** we’re doing it (linking back to the problems list)
* **Exactly what to change** – shown in unified-diff format with line-numbers and comments
* **Any follow-up** (tests, config, deployment tweaks)

---

## ☑️ Task 0 - create a work-branch

```bash
git checkout -b fix/broadcast-overflow-and-vacate-bugs
```

---

## ⭐ Task 1 – Buffer the global `broadcast` channel & guard against overflow
**Files touched:** `websocket/globals.go`, `websocket/broadcast.go`

### 1-A globals.go – change the channel declaration

```diff
@@
-// broadcast is a channel for sending messages to all clients
-var broadcast = make(chan []byte)
+// broadcast is buffered to absorb short spikes (500 msgs ≈ a few seconds of traffic)
+// If overflow occurs we’ll drop & log the packet rather than block every writer.
+var broadcast = make(chan []byte, 500)
```

> **Why:** avoids the writer goroutines blocking and piling up under load.

### 1-B broadcast.go – wrap **all** writes in a non-blocking send

Add this helper **just below** the imports:

```go
// safeSend queues data or logs & drops if the buffer is full (prevents deadlock)
func safeSend(data []byte) {
	select {
	case broadcast <- data:
		// OK
	default:
		logger.Warn.Println("[safeSend] broadcast channel FULL – dropping msg")
	}
}
```

Replace every bare `broadcast <- …` send in **this file only** (there are three) with `safeSend(…)`.

Example:

```diff
-broadcast <- msg
+safeSend(msg)
```

(Do the same for `broadcastFinalResults`, `broadcastAllNextAttemptTimers`, etc. inside this file.)

---

## ⭐ Task 2 – Read-flood & payload-size protection
**File:** `websocket/connection.go`

### 2-A Enforce a 1 KiB max per message

Right after successful upgrade in `ServeWs`:

```diff
 wsConn, err := upgrader.Upgrade(w, r, nil)
 if err != nil { … }

+// ---- security: 1 KiB/message hard-limit & 4 s write timeout ---------- //
+wsConn.SetReadLimit(1024)              // 1024 bytes
```

(Use any size you consider safe; 1 KiB is plenty for our JSON messages.)

### 2-B Basic per-connection rate limiting (naïve, but enough)

Inside `readPump()` add a simple “time since last message” guard **at the top of the for-loop**:

```go
var lastMsg time.Time
…
for {
    if !lastMsg.IsZero() && time.Since(lastMsg) < 200*time.Millisecond {
        logger.Warn.Printf("[readPump] %v flooding; closing", c.conn.RemoteAddr())
        return
    }
    lastMsg = time.Now()
    …
}
```

---

## ⭐ Task 3 – Cleanly kill WS connections when a referee is forced to vacate or logs out
**Files:** `websocket/connection.go`, `controllers/admin_controller.go`, `controllers/page_controller.go`

### 3-A Add a helper to close all WS for a given user **and/or** meet

Insert in `connection.go` (after `broadcastRefereeHealth` helper):

```go
// CloseConnectionsForUser forcibly closes any WS whose judgeID *or* remoteAddr
// matches the supplied identifier (used when force-vacating / logout).
func CloseConnectionsForUser(identifier string) {
	connectionsMu.Lock()
	for c := range connections {
		if c.judgeID == identifier || c.conn.RemoteAddr().String() == identifier {
			_ = c.conn.Close() // triggers unregister via read/write pumps
		}
	}
	connectionsMu.Unlock()
}
```

### 3-B Call the helper from **ForceVacate** (admin_controller.go)

Inside the switch-case (after we discover the `occupant` we’re kicking):

```diff
-// remove user from the active list
-delete(ActiveUsers, occupant)
+// clean up websocket(s) first
+websocket.CloseConnectionsForUser(occupant)
+delete(ActiveUsers, occupant)
```

### 3-C Call the helper from **Logout** (page_controller.go)

Add just before removing from `ActiveUsers`:

```go
websocket.CloseConnectionsForUser(userEmail)
```

---

## ⭐ Task 4 – Fix `/position/vacate` → 404 redirect edge-case
**File:** `controllers/position_controller.go`

Replace the one-liner with full cleanup + redirect:

```diff
-func (pc *PositionController) VacatePosition(c *gin.Context) {
-    c.Redirect(http.StatusFound, "/logout?reason=vacate")
+func (pc *PositionController) VacatePosition(c *gin.Context) {
+    // fully clear session first to avoid stale cookies -> 404
+    session := sessions.Default(c)
+    _ = session.Clear()
+    _ = session.Save()
+
+    // offload to the common Logout flow
+    c.Redirect(http.StatusFound, "/logout?reason=vacate")
 }
```

(You already wired `/logout` to the common handler, so this resolves the 404 / “ghost session” report.)

---

## ⭐ Task 5 – Publish accurate CloudWatch metrics on WS connect / disconnect
**File:** `websocket/connection.go` (two quick hooks)

```diff
 func registerConnection(c *Connection) {
     connectionsMu.Lock()
     connections[c] = true
     connectionsMu.Unlock()
+
+    websocket.PublishRefereeConnections(len(connections), c.meetName)
 }
 …
 func unregisterConnection(c *Connection) {
     connectionsMu.Lock()
     delete(connections, c)
     connectionsMu.Unlock()
+
+    websocket.PublishRefereeConnections(len(connections), c.meetName)
 }
```

---

## 🔜 Backlog / future branches

| Idea                            | Pointer                                                                                                     |
|---------------------------------|-------------------------------------------------------------------------------------------------------------|
| **Unify Heartbeat & WS maps**   | Replace `refereeSessions` with `connections`-map lookups; send PING failures to mark “red” in UI.           |
| **Buffered consumer goroutine** | Off-load JSON-parse + meetFilter work from `HandleMessages` into N workers to reduce head-of-line blocking. |
| **Admin dashboard panel**       | Use `/occupancy` JSON + new `/metrics` endpoint to paint live stats (React or htmx).                        |
| **Stale timer detection**       | Periodic sweep of `MeetState` to ensure `PlatformReadyActive` false when `time.Now() > EndTime`.            |
| **Integration tests**           | Add Go test harness with in-memory WS & HTTP to assert no broadcast drops under 100 rps.                    |

---

## 🧪 Smoke test locally

```bash
go vet ./...
go test ./...          # should still pass
go run cmd/referee-lights/main.go
```

* Open two browser tabs: lights page & referee page
* Click “Platform Ready” 20× quickly – lights shouldn’t freeze
* Force-vacate a referee → their tab should auto-disconnect

-------
other todos
- scale to zero for CDK app. If the app has been idle for say 30 minutes
  I'd like to scale to zero. Its not just a cost saving measure, its also
  so that we have "closure" on the logs for an event. the logs we just
  reviewed were still running 36 hours after the event had finished.
- improve logging: i'd like exception type logging. For example i don't
  want to see every time we get a healthy response from a health check. I
  do want to see when a connection is unhleathy
- i'd like to improve logging coverage for the application (all parts), but
  i don't want spammy logs i want to see insightful logs that tell me when
  something has gone wrong with the app.
- The logs should be relatively 'lean'. I don't want 1 3gb logs file after
  a 2 day meet, but at the same time i want to see everythiing i need to in
  order to know that the app is working properly or not

-----
# Simulation test plan
Below is a **self-contained “simulation” test-plan** you can drop into `/tests/integration/simulation_test.go`.
Its job is to stand-up the whole Gin router in-process, spin-up **fake meet-director + referee browsers (WebSocket clients)**, replay the exact message-exchange pattern that surfaced in the production logs, and assert that every message arrives where it should.

---

## 1  Why an *integration* simulation?

* Unit-tests (the 11 files we listed earlier) protect single functions.
* The production failure was **cross-cutting**: the browser lights page never received broadcasts even though referees were clearly sending decisions.
* Re-creating that traffic end-to-end is the only way to know the fix works **before the next real meet**.
* initially we'll use test_mule as a test event. we'll run a single event
  only but later we'll run tests on multiple concurrent meets
---

## 2  Overview of the harness

| Layer                      | What we spin-up                                                              | Library                               |
|----------------------------|------------------------------------------------------------------------------|---------------------------------------|
| **HTTP server**            | `httptest.NewServer(SetupRouter("test"))`                                    | `net/http/httptest`                   |
| **Referee clients (3x)**   | Lightweight goroutines that dial `ws://…/referee-updates?meetName=TestMeet`  | `github.com/gorilla/websocket`        |
| **Meet-director client**   | Another WebSocket that subscribes to the same room and captures all messages | same                                  |
| **Assert/require helpers** | Check expected broadcasts & occupancy states                                 | `github.com/stretchr/testify/require` |

Everything lives in-process; no ports, Docker, or AWS credentials needed.

## 3  Scenarios covered

| Scenario ID | Flow reproduced                                                                                     | Log symptom you saw                         | Assertion(s)                                                                                                                                                           |
|-------------|-----------------------------------------------------------------------------------------------------|---------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **S1**      | 3 referees claim seats → meet-director sees `occupancyChanged`                                      | Sometimes only 2 seats turned green         | Director receives **exactly 3** `occupancyChanged` messages and the final one lists all three names                                                                    |
| **S2**      | Centre referee clicks *Platform Ready* → timer counts & decisions arrive → results lights broadcast | Lights page never flipped white/red         | (a) Director receives `startTimer` then successive `updatePlatformReadyTime` ticks; (b) after 3 `submitDecision` msgs, director receives a single `displayResults` msg |
| **S3**      | Referee network drop then page refresh                                                              | Disconnected judge still shown as connected | Close one WebSocket, wait 2 *pingPeriod*s, connect again – `refereeHealth` must first drop to 2 then rise back to 3                                                    |
| **S4**      | Referee hits *Vacate Position*                                                                      | 404 instead of redirect                     | HTTP POST `/position/vacate` must respond 302 → `/logout?reason=vacate`; afterwards `occupancy.LeftUser == ""` etc.                                                    |
| **S5**      | Admin *Force Vacate* via POST `/force-vacate`                                                       | Seat stayed occupied in production          | Occupancy empty & `occupancyChanged` broadcast seen by director                                                                                                        |
| **S6**      | Admin *Reset Instance*                                                                              | Whole state not cleared                     | Occupancy map zeroed **and** `DefaultStateProvider.GetMeetState("TestMeet").JudgeDecisions` length == 0                                                                |

---

## 4 Skeleton code (add/replace only the block below)

```go
// File: tests/integration/simulation_test.go
// +build integration

package integration_test

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"go-ref-lights/controllers"
	"go-ref-lights/websocket as ws"
)

func TestFullSimulation(t *testing.T) {
	// --- spin up router ---
	srv := httptest.NewServer(controllers.SetupRouter("test"))
	defer srv.Close()

	origin := "http://" + srv.Listener.Addr().String()
	wsURL  := "ws://" + srv.Listener.Addr().String() + "/referee-updates?meetName=TestMeet"

	// --- helper to open a client ---
	openClient := func() *websocket.Conn {
		c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		return c
	}

	// --- meet-director socket & capture channel ---
	director := openClient()
	defer director.Close()
	var directorMu sync.Mutex
	var captured [][]byte
	go func() {
		for {
			_, msg, err := director.ReadMessage()
			if err != nil {
				return
			}
			directorMu.Lock()
			captured = append(captured, msg)
			directorMu.Unlock()
		}
	}()

	// --- 3 referee sockets ---
	refIDs := []string{"left", "center", "right"}
	for _, id := range refIDs {
		ref := openClient()
		defer ref.Close()

		// register ref
		require.NoError(t,
			ref.WriteJSON(map[string]string{
				"action":   "registerRef",
				"judgeId":  id,
				"meetName": "TestMeet",
			}),
		)
	}

	// ------- Scenario S1: wait for 3 occupancyChanged -------
	require.Eventually(t, func() bool {
		count := 0
		directorMu.Lock()
		defer directorMu.Unlock()
		for _, m := range captured {
			var tmp map[string]interface{}
			_ = json.Unmarshal(m, &tmp)
			if tmp["action"] == "occupancyChanged" {
				count++
			}
		}
		return count == 3 // exactly 3 seat claims
	}, 5*time.Second, 100*time.Millisecond, "didn't see 3 occupancyChanged messages")

	// ------- Scenario S2: start timer & submit decisions -------
	center := openClient()
	defer center.Close()
	_ = center.WriteJSON(map[string]string{
		"action":   "startTimer",
		"meetName": "TestMeet",
	})
	// three decisions
	for _, id := range refIDs {
		ref := openClient(); defer ref.Close()
		_ = ref.WriteJSON(map[string]string{
			"action":   "submitDecision",
			"judgeId":  id,
			"decision": "good",
			"meetName": "TestMeet",
		})
	}

	require.Eventually(t, func() bool {
		directorMu.Lock(); defer directorMu.Unlock()
		for _, m := range captured {
			var tmp map[string]interface{}
			_ = json.Unmarshal(m, &tmp)
			if tmp["action"] == "displayResults" {
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "displayResults never broadcast")

	//  … repeat pattern for S3–S6 (omitted for brevity) ...
}
```

### How to run it locally
we should initially test locally, but thats just a stepping stone
```bash
go test ./tests/integration -tags=integration -v
```

i want to deploy the app to AWS and run these tests in a way that closely
mimics a real meet

### How to wire it into CI

* Add an `integration` job that runs **after** the unit-test job, **unless** the Docker build tag forbids AWS creds (our harness is fully offline).
* Fail the pipeline if *any* scenario fails.

---

## 5 Extending / tweaking

* **Throttle/latency** – wrap the websocket `Dial` in a proxy that introduces RTT delays to mimic patchy venue Wi-Fi.
* **High load** – parametrize `refIDs` and spin 50 referees (they’ll all join “left”; still stresses broadcast fan-out).
* **Chaos** – randomly `Close()` referee sockets while timers run, then reconnect.

---

## 6 Next steps after the hot-fix

1. **Implement** any code changes the earlier log-review suggested.
2. **Run `go test …`** – it should now pass 100 %.
3. Add extra assertions whenever a new regression appears in logs; the harness is easily extendable.

This gives you a repeatable, push-button “competition in a box” you can run on every commit—no more waiting for the next live meet before discovering breakage.

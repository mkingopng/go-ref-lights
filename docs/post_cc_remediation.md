

- This is the heartbeat. We don't really need to see all the successful
heartbeat logs, only failures. it doesn't seem to be covered by the logger
```
"timestamp": 1746840336608,
"message": "[GIN] 2025/05/10 - 01:25:36 | 200 |      760.12µs | 220.240.123.147 | POST     \"/log\""
```

-
```
"timestamp": 1746840336608,
"message": "DEBUG: 2025/05/10 01:25:36 main.go:131: [2025-05-10T01:25:36.564Z] DEBUG: \"[lights.js] displayResults received: left=white, center=white, right=white\""
```



---

# Action Plan from Log Analysis (Coastal Classic)

## Timer Errors. Ask thomas. Seems to work for me
- platform ready
- next attempt

## remove headers from referee panels, fix headers on all others

## 1. **Fix `writePump` Spam (WebSocket write loop)**
- **Occurrences:** 61
- **File:** `connection.go`
- **Cause:** repeated failed or stale writes to WebSocket clients
- **Why it matters:** This is the top log spam and likely a source of degraded performance or client desync.

## Actions:
* [ ] Add write error logging inside `writePump` with session/ref info
* [ ] Detect if client is disconnected or blocking
* [ ] Gracefully `unregisterConnection` on write error
* [ ] Add backpressure logic or drop policy for `broadcast` channel

---

## 2. **Investigate Server Startup Errors**
**Occurrences:** 31
**File:** `main.go`
**Cause:** startup/init errors, possibly due to file paths, port conflicts, or env config

## Actions:
* [ ] Standardize log output for startup steps
* [ ] Log any `InitLogger()`, `SetupRouter()`, or `ListenAndServe()` failures using structured logger
* [ ] Exit early if env vars or templates are misconfigured

## 3. Handle `occupancy_service.go` Failures Gracefully
- **Occurrences:** 7
- **Problem:** errors in setting or removing occupancy — might be due to session mismatches or stale client state

#### Actions:

* [ ] Add retry or fallback logic for stale sessions
* [ ] Ensure vacated users are fully cleaned from `ActiveUsers` and WebSocket maps
* [ ] Improve log context (position, session ID, referee name)

---

### 4. **Improve `page_controller.go` Error Logging**

**Occurrences:** 7
**Likely Cause:** session/token failures or UI routing bugs (e.g., `/vacate`)

#### Actions:

* [ ] Add structured error logs in all handler paths
* [ ] Ensure session validation failures are logged with referee ID + token details
* [ ] Consider catching and logging 404/500s with request metadata

---

### 5. **Throttle Log Volume for Repeating Warnings**

**Top offenders:** `[writePump]`, `[Vacate]`, `[Broadcast]`
**Why it matters:** many identical warnings flood the logs and obscure real problems

#### Actions:

* [ ] Add log suppression or cooldown for repeat log messages (`lastLogTime + X`)
* [ ] Add `rate-limited` tag to indicate throttling is active
* [ ] Log “N messages suppressed” once per period if needed

---

### 6. **Add Context to All Logs**

Your logs are missing:

* Meet ID or name
* Referee/session ID
* Position (`Platform A`, etc.)

#### Actions:

* [ ] Modify logger calls to include these as standard fields
* [ ] Consider creating a helper function:

  ```go
  func LogErrorWithContext(ctx *RequestContext, msg string, args ...interface{}) { ... }
  ```

---

## 🔄 Suggested Branch Plan

| Branch Name                  | Purpose                                         |
| ---------------------------- | ----------------------------------------------- |
| `fix/websocket-write-errors` | Improve `writePump` reliability and log clarity |
| `fix/occupancy-cleanup`      | Handle vacated/stale session edge cases         |
| `chore/startup-logging`      | Improve startup diagnostics                     |
| `chore/log-throttling`       | Add message suppression for flood control       |

---

Would you like me to generate TODO comments or stubs in each affected file? Or prep a sample `LogWithContext()` helper?

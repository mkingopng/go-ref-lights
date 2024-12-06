
go run main.go


Below are the key issues you’ve highlighted along with suggestions on how to address them. After the conceptual outline, you’ll find more concrete directions on implementing some of these changes.

**1. Bad Packets (Unreliable Behavior Due to Multiple Logins and Distracted Referees)**
- **Root Causes:**
	- Multiple referees logging in as the same role (e.g., two people accessing the "left" referee page at the same time).
	- Referees switching tabs or using other apps, causing intermittent WebSocket behavior or stale connections.

- **Potential Solutions:**
	- **Single-Session Enforcement for Each Referee Role:**  
	  When a new WebSocket connection for a role comes in (e.g. "left"), disconnect the previous one or block the new one. You could maintain a mapping of `role -> client connection` and ensure only one is active at a time.
	- **Heartbeat / Ping-Pong Messages:**  
	  Implement periodic heartbeat messages from the server to check if a referee is still connected and responsive. Disconnect stale connections to prevent confusion.
	- **Graceful Handling of Invalid/Unexpected Messages:**  
	  Add checks around `json.Unmarshal` and message handling. If a packet doesn’t conform to the expected schema, log it and ignore it rather than causing downstream issues.

**2. Improved Logging for Better Insights**
- **Add Structured Logging:**  
  Consider using a structured logging library like `logrus` or `zap`. Include details in each log (user role, IP, timestamp, action performed).
- **Strategic Logging Points:**
	- Log every referee decision received and when each timer starts/stops.
	- Log WebSocket connections, disconnections, and invalid messages.
	- Log user login attempts and their outcomes.

**3. Make the App More Resilient to Bad Referee Behavior**
- **Graceful Degradation:**
	- If one referee never submits a decision, time out that phase and proceed. Don’t let the UI or server logic hang indefinitely waiting for all three.
	- Implement timeouts for decisions. If after X seconds not all decisions are in, proceed or reset.
- **Fallback States:**
	- If a decision is missing, treat it as a “no response” and still proceed after a certain timeout.
	- On unexpected disconnection, remove that referee’s decision from the required set or reset the state.

**4. The Reset Button is Redundant — Comment It Out**
- Simply remove the `resetTimerButton` and associated event listeners in `centre.html` and `centre.js`. You already have `startTimer` and `stopTimer` actions and a `resetTimer` action triggered server-side if needed.

  For example, in `templates/centre.html`, remove:
  ```html
  <button id="resetTimerButton" class="action-button">Reset</button>
  ```

  And in `centre.js`, remove all references to `resetTimerButton`.

**5. Change the Display Duration of Referee Decisions to 30 Seconds**
- **Current Behavior:**  
  Decisions are displayed until the center referee presses "platform ready".

- **Desired Behavior:**  
  Display them for 30 seconds, then automatically clear.

- **Implementation Hint:**  
  In `lights.js`, `displayResults()` currently uses `setTimeout` with a short delay (1 second) to reset the display. Replace it with a 30-second timeout:
  ```javascript
  // Instead of setTimeout(..., 1000)
  setTimeout(function() {
    displayMessage('', '');
    // Reset logic after 30 seconds
    // ...
  }, 30000);
  ```

  You’ll also need to ensure that `resetCircles()` and clearing of `judgeDecisions` happens after this 30-second timer expires, not just when “platform ready” is pressed.

**6. 'Platform Ready' Should Clear the Referee Decision List**
- **Current Behavior:**  
  Currently, `judgeDecisions` are only cleared when three decisions are in or a reset occurs.

- **Implementation:**  
  In `websocket/handler.go`, when handling the "platform ready" action (or timer start), explicitly reset `judgeDecisions`:
  ```go
  // For example, if 'startTimer' corresponds to 'platform ready':
  if action == "startTimer" {
    judgeMutex.Lock()
    judgeDecisions = make(map[string]string)
    judgeMutex.Unlock()
    // Also send a message to Lights to reset visually
  }
  ```

  On the client side (`lights.js`), `handleTimerAction("startTimer")` might call `resetForNewLift()` which resets the local UI.

**7. Improve Log In with Google Sign-In or IAM**
- **Current Behavior:**  
  Hard-coded username and password.

- **Implementation Approach:**
	- Integrate OAuth 2.0 / OpenID Connect. For Google, you can use `golang.org/x/oauth2` and `golang.org/x/oauth2/google` packages.
	- Store the user’s authenticated session details in the session store and verify them on each request.
	- Remove the hard-coded credentials from `PerformLogin` and replace them with a redirect to Google’s OAuth endpoint. On callback, verify the token and set the user session.

**8. The Next Attempt Timer Should Persist For 60 Seconds Even After 'Platform Ready'**
- **Current Behavior:**  
  The code may stop or reset the attempt timer when “platform ready” is pressed.

- **Implementation:**  
  Check `displayResults()` and `handleTimerAction()` in `lights.js`. Make sure that starting the platform ready timer or any other action does not prematurely hide or reset the next attempt timer. Remove any logic that clears the `nextAttemptTimerContainer` when the platform becomes ready. The next attempt timer should remain visible and count down its full duration.

**9. More Than One Next Attempt Timer**
- **Issue:**  
  Currently, there is only one `nextAttemptTimer`. If attempts occur in rapid succession, you need multiple timers running in parallel.

- **Implementation Approach:**
	- Change your data structure from a single `nextAttemptTimeLeft` variable to an array of timer objects.
	- Dynamically create HTML elements for each new next attempt timer (e.g., `<div class="timer-container"><div class="timer">...</div></div>`).
	- Start a new timer each time a new attempt begins and push it into the array. Each timer runs independently and updates its own DOM element.
	- When a timer finishes, hide or remove its element.

  For example, in `lights.js`:
  ```javascript
  let nextAttemptTimers = [];

  function startNextAttemptTimer() {
    let timeLeft = 60;
    let timerId = setInterval(function() {
      timeLeft--;
      updateThisTimerDisplay(timerId, timeLeft);
      if (timeLeft <= 0) {
        clearInterval(timerId);
        // Hide or remove that timer's display
      }
    }, 1000);
    nextAttemptTimers.push({ id: timerId, timeLeft: 60 });
    // Create a new DOM element for this timer and append it to #nextAttemptTimerContainer
  }
  ```

  This way you can handle multiple attempts concurrently.

---

### Putting It All Together

- **Referee Behavior and Bad Packets:**  
  Implement stricter session handling. When a user logs in as "left", store that in session. On the server side, if a second "left" connection attempts, disconnect the previous one or prevent the new one from connecting. Add heartbeat/ping messages to detect stale connections.

- **Logging:**  
  Everywhere a significant event happens (login attempt, judge decision, timer start/stop), add `log.Printf()` with contextual information. Over time, you might upgrade to a more advanced logging solution.

- **Removing the Reset Button:**  
  Just comment it out or remove it from the `centre.html` template and corresponding JS.

- **Display Referee Decisions for 30 Seconds:**  
  Change the timeout in `displayResults()` from 1 second to 30 seconds, and only then clear decisions and reset the interface.

- **Clear Decisions on Platform Ready:**  
  On the server side (`websocket.HandleMessages()`), when you receive a `startTimer` action (assuming that corresponds to platform ready), `judgeDecisions = make(map[string]string)` to clear previous attempts.

- **Google Login or IAM:**  
  Replace the simple `PerformLogin` function with OAuth-based login. On successful callback from Google, set session data and redirect to the home page. For local testing, you might run a local OAuth flow. In production, ensure you have a proper client ID/secret configured.

- **Timer Persistence and Multiple Timers:**  
  Don’t hide or reset the next attempt timer prematurely. If multiple attempts might happen back-to-back, dynamically create timers and track them in an array. Keep displaying them on the lights screen.

---

By addressing each point above, you’ll create a more reliable, maintainable, and user-friendly powerlifting referee lights system that’s robust against human error and environmental quirks.

---
Below is a step-by-step guide on what you need to change in each file to shift timer logic and decision-handling from the client-side JavaScript to the Go (backend) side. This approach simplifies the JavaScript so it only renders UI updates based on messages from the server.

### High-Level Overview
- **Server (Go)**:
	- Maintain all timing logic, including starting, stopping, and resetting timers (platform ready, next attempt, etc.).
	- Aggregate referee decisions and determine when to display results and when to clear them.
	- Broadcast state updates (time left, results, resets) to all clients via WebSocket.

- **Client (JavaScript)**:
	- Remove `setInterval` and other time-based logic.
	- Remove decision aggregation logic in the browser.
	- Listen to server-sent WebSocket messages and simply update the UI accordingly.

---

### Changes in `main.go`

**Current Code:**  
`main.go` sets up the server and routes. You likely won’t need to make huge changes here. However, you may introduce global state or pass references into `websocket/handler.go` for managing timers and state if needed.

**What to do:**
1. No immediate changes are strictly required unless you want to store global state or inject dependencies.
2. If you prefer, define a global state manager (a struct) that holds timer states and referee decisions. Pass a reference of this state manager to `websocket/handler.go`.

**Example (Optional):**
```go
// Optional: define a global or package-level state structure
// type AppState struct {
//     judgeDecisions map[string]string
//     ...
// }
// var state = &AppState{ judgeDecisions: make(map[string]string) }
```
Then you’d use `state` in `websocket/handler.go`.

---

### Changes in `websocket/handler.go`

**Current Code:**  
`websocket/handler.go` currently:
- Handles incoming referee decisions
- Aggregates them and sends results once all three are in
- Relies on the frontend to handle timers

**What to do:**
1. **Move Timer Logic to Go:**
	- When you receive a `startTimer` action (which indicates "platform ready"), start a Go routine using `time.Ticker` or `time.AfterFunc` to count down the 60 seconds.
	- Each second, broadcast a JSON message like `{"action":"updatePlatformReadyTime","timeLeft":59}` etc. to update clients.
	- When time runs out, broadcast a message `{"action":"platformReadyExpired"}` to let clients know to display "Time Up" or reset UI accordingly.

2. **Handle Next Attempt Timers on the Server:**
	- After displaying results, start a 30-second timer for the referee decisions to remain visible. When that expires, send `{"action":"clearResults"}` to clients.
	- Also start the next attempt 60-second timers on the server. Send periodic updates (`{"action":"updateNextAttemptTime","timeLeft":...}`) to the client.

3. **Remove Local Decision Reset From Client:**  
   Instead of the client resetting decisions, when `startTimer` (platform ready) is triggered, reset `judgeDecisions` map server-side:
   ```go
   judgeMutex.Lock()
   judgeDecisions = make(map[string]string)
   judgeMutex.Unlock()
   ```

4. **Send All State Changes Via WebSocket Messages:**  
   For example, when all three decisions arrive:
	- Compute the results server-side.
	- Immediately send `{"action":"displayResults","leftDecision":...,"centreDecision":...,"rightDecision":...}` to clients.
	- Start a Go routine that waits 30 seconds, then sends `{"action":"clearResults"}`.

**Example of a timer routine in Go:**
```go
func startPlatformReadyTimer() {
    timeLeft := 60
    ticker := time.NewTicker(time.Second)
    go func() {
        for range ticker.C {
            timeLeft--
            if timeLeft > 0 {
                broadcast <- []byte(fmt.Sprintf(`{"action":"updatePlatformReadyTime","timeLeft":%d}`, timeLeft))
            } else {
                ticker.Stop()
                broadcast <- []byte(`{"action":"platformReadyExpired"}`)
                break
            }
        }
    }()
}
```

Use similar logic for next attempt timers and any other timers currently done in JS.

---

### Changes in `controllers/*.go`, `middleware/*.go`, `services/*.go`

**Current Code:**  
These files handle login, QR code generation, and authentication.

**What to do:**
- **No changes needed** for timer logic.
- If you integrate Google Login or IAM in the future, just replace `PerformLogin` logic. This is independent of the timer logic migration.

---

### Changes in `static/js/centre.js`, `static/js/left.js`, `static/js/right.js`

**Current Code:**
- They send decisions to the server and may start/stop/reset timers.

**What to do:**
1. **Remove Direct Timer Control Calls (Centre.js):**
	- Remove `startTimerButton`, `stopTimerButton`, and `resetTimerButton` references.
	- Instead, when the "Platform Ready" action is triggered by the center referee, just send the `{"action":"startTimer"}` message to the server and let the server handle it.

   For example, remove:
   ```javascript
   startTimerButton.addEventListener('click', function() {
       sendTimerAction('startTimer');
   });
   stopTimerButton.addEventListener('click', function() {
       sendTimerAction('stopTimer');
   });
   resetTimerButton.addEventListener('click', function() {
       sendTimerAction('resetTimer');
   });
   ```
   Keep just `sendTimerAction('startTimer')` if that's how you indicate "platform ready." The server will handle everything else.

2. **Left.js and Right.js**
	- They only send `white` or `red` decisions. No timer logic here, so no changes required.

---

### Changes in `static/js/lights.js`

**Current Code:**
- Manages timers (setInterval), displays results, clears them after short delays.
- Has logic for displaying messages and starting/stopping/resetting timers client-side.

**What to do:**
1. **Remove setInterval and All Timer Logic:**  
   Delete all code related to `platformReadyTimerInterval`, `nextAttemptTimerInterval`, and `startSecondTimer()`. The client should not count time anymore.  
   The client only updates UI based on server messages like `"action":"updatePlatformReadyTime"` or `"action":"updateNextAttemptTime"`.

2. **Handle Incoming Messages From Server:**
   For each `onmessage` action, just update the UI. For example:
   ```javascript
   socket.onmessage = function(event) {
       var data = JSON.parse(event.data);

       switch (data.action) {
           case "judgeSubmitted":
               showJudgeSubmissionIndicator(data.judgeId);
               break;
           case "displayResults":
               displayResults(data);
               break;
           case "clearResults":
               clearResultsFromUI();
               break;
           case "updatePlatformReadyTime":
               updatePlatformReadyTimerOnUI(data.timeLeft);
               break;
           case "platformReadyExpired":
               handlePlatformReadyExpired();
               break;
           case "updateNextAttemptTime":
               updateNextAttemptTimerOnUI(data.timeLeft);
               break;
           // No local timers, just UI updates
           default:
               console.warn("Unknown action:", data.action);
       }
   };
   ```

3. **UI Update Functions Only:**
	- `updatePlatformReadyTimerOnUI(timeLeft)` updates the DOM element that shows the platform ready time.
	- `updateNextAttemptTimerOnUI(timeLeft)` updates the next attempt timer display.
	- `clearResultsFromUI()` resets the circles and clears the message.

4. **Remove Code That Starts Timers Locally:**  
   Delete calls like `setInterval(...)`, `startSecondTimer()`, `startTimer()`, `stopTimer()`, `resetTimer()` that rely on client-side intervals. Keep only DOM manipulation code.

---

### Changes in HTML Templates

**Current Code:**
- Templates like `centre.html` have buttons for start/stop/reset of timers.
- `lights.html` shows timer containers.

**What to do:**
1. **`centre.html`:**
	- Remove the reset button or comment it out:
	  ```html
	  <!-- <button id="resetTimerButton" class="action-button">Reset</button> -->
	  ```
	- Keep a single "Platform Ready" button if you need it, but remember now it just sends a message to the server rather than controlling the timer locally.

2. **`lights.html`:**
	- You can keep the timer display elements but remember they are now only updated based on server messages.
	- Remove references in your HTML or comments that indicate client-based timing.

3. **`left.html` and `right.html`:**
	- No changes needed since they only have Good Lift / No Lift buttons.

---

### Summary

**Before:**
- The client sets and manages timers using JavaScript `setInterval`.
- The client also handles resetting decisions and clearing results after short timeouts.

**After:**
- The Go server handles all timing logic. On `startTimer`, it starts a Go routine that counts down and sends `"updatePlatformReadyTime"` messages every second.
- When all three decisions are received, the server sends `"displayResults"` and then starts a 30-second timer server-side. After 30 seconds, it sends `"clearResults"`.
- The client (JavaScript) now only updates the UI based on incoming WebSocket messages. No more `setInterval` or local timing logic.
- The result is simpler, more maintainable JavaScript and a single source of truth for the application state on the server-side.

By following these steps, you centralize logic in Go and let the JavaScript remain as simple as possible, thereby achieving your stated goals.


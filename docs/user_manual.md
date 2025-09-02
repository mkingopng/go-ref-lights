# go-ref-lights User Manual

## 1. Overview

**go-ref-lights** is a web-based referee lights application designed for powerlifting meets. Its primary goals are:

- Facilitate referee seat assignments (left, center, right).
- Record referees’ decisions (white or red lights).
- Provide a real-time display of lights (decisions) and timers to everyone involved.
- Allow meet directors (admins) to administer the meet and forcibly vacate or reset positions when needed.
- Enable a superuser “Sudo” role as an emergency fallback to access all meets.

The application can run locally for testing or be deployed to a server.

---

## 2. System Requirements

1. **Go** (version 1.18 or later recommended).
2. **Node.js** (if you need to rebuild static assets, not always required).
3. **Browser** (Chrome, Safari, Firefox, or Edge) to access the web UI.
4. **(Optional) Docker** if you prefer containerized deployment.

---

## 3. Installation & Setup

1. **Obtain the source code**
	- Clone or download the go-ref-lights repository into your GOPATH or a preferred location.

2. **Install Dependencies**
	- Run `go mod tidy` in the project root to ensure Go dependencies are installed.

3. **Configuration**
	- Within the `./config` folder, you will find JSON files such as `meet_creds.json` and `meets.json`.
		- **`meets.json`** controls the meets that appear on the “Choose Meet” page.
		- **`meet_creds.json`** stores admin credentials for each meet and an optional superuser account.
	- The hashed passwords must be **bcrypt**-hashed and start with `"$2b$12$..."`.
	- Example structure in `meet_creds.json`:
	  ```json
	  {
		"meets": [
		  {
			"name": "SampleMeet2025",
			"date": "2025-08-01",
			"admin": {
			  "username": "meetdirector",
			  "password": "$2b$12$ExampleHashHere...",
			  "isadmin": true
			},
			"logo": "/static/images/sample_logo.png"
		  }
		],
		"superuser": {
		  "username": "superuser",
		  "password": "$2b$12$AnotherExampleHashHere...",
		  "isadmin": true,
		  "sudo": true
		}
	  }
	  ```

4. **Environment Variables** (optional)
	- **ENV**: Controls application mode and logging levels
		- `ENV=development` (default): Verbose logging with all levels including DEBUG
		- `ENV=production`: Optimized logging with ERROR, WARN, and critical INFO only
		- `ENV=test`: Minimal logging for test environments
	- **LOG_LEVEL**: Optional override for specific log level (DEBUG, INFO, WARN, ERROR)
	- Production mode also enables `gin.ReleaseMode` and secure cookies.

---

## 4. Starting the Application

1. **Local Run**
	- From the root directory, run:
	  ```bash
	  go run main.go
	  ```
	- The default server listens on `0.0.0.0:8080` (or `localhost:8080` if you use environment variables).
	- Access the web interface at `http://localhost:8080`.

2. **Production Deployment**
	- Set environment variables (for example, using a `.env` file or manually):
	  ```bash
	  export ENV=production          # Enables production logging and security
	  export APP_HOST=0.0.0.0
	  export APP_PORT=8080
	  # Optional: Override log level if needed
	  # export LOG_LEVEL=DEBUG
	  ```
	- Run the same `go run main.go`. The server will be available on the specified host and port with optimized logging.

---

## 5. Terminology and Roles

- **Meet**: A powerlifting event with a unique name, date, admin credentials, and logo.
- **Referee**: A user that occupies one of three seats: Left, Center, or Right.
- **Admin / Meet Director**: A privileged user who can forcibly vacate positions, reset the meet, and see an Admin Panel.
- **Superuser (Sudo)**: A top-level fallback user who can access any meet, forcibly log out the meet director, restart meets, etc.

---

## 6. Basic Usage Workflow

### 6.1 Choose Meet
1. When you visit `http://localhost:8080` (or your deployed URL), you’ll see a list of configured meets from `meets.json`.
2. Select a meet from the dropdown and click **Set Meet**.
	- The system then stores that selection in your session.

### 6.2 Login
1. After choosing a meet, you will be redirected to `/login`.
2. Enter the username and password for either:
	- The meet’s admin credentials, or
	- A normal user credential if your meet has them, or
	- The superuser credential (if you want Sudo privileges).
3. If successful:
	- **Meet Director / Admin** users are redirected either to the Admin screen or to an index page where they can continue to admin tasks.
	- **Normal referees** who specified a seat in the URL (like `?position=left`) will automatically claim that seat if free. Otherwise, they can choose a seat from the Positions page.
	- **Superuser** logs into a special route `/sudo` with advanced fallback controls.

### 6.3 Referee Seat Selection
1. If a referee logs in without specifying a seat, they can visit **Positions** at `/positions`.
2. The application shows who is occupying each seat. A seat is displayed as *occupied* if another user is already in that seat.
3. Click the **Claim** button next to an available seat. Once claimed, the referee is directed to the seat’s dedicated page (left, center, or right).
4. The seat page has the real-time scoreboard or judge-lights UI and the **Vacate** and **Logout** buttons.

### 6.4 Referee’s Duties
1. Each seat page (e.g., `/left`) is interactive:
	- Displays live timer (if any).
	- Shows real-time signals from other referees or the meet director’s commands.
	- The referee can enter a decision (white or red light). The final results broadcast to all participants when all three referees have decided.

### 6.5 Vacating & Logging Out
1. **Vacate Position**: The user can click **Vacate** on the seat page to free up that seat and return to `/index`.
2. **Logout**: The user can click **Logout** to remove themselves from the meet entirely and return to the main meet index page.
3. If any error or disconnection occurs, simply **refresh** or **reopen** the app. The system tries to re-establish the seat if possible, or else you can reselect.

---

## 7. Admin (Meet Director) Functions

1. **Admin Panel**:
	- Visit `/admin?meet=<meetName>` or, if logged in as admin, click the link to the Admin Panel.
	- The admin panel shows:
		- Which seats are currently occupied by which users.
		- Buttons to **Force Vacate** each seat.
		- A button or form to **Reset Instance** (clears all referees, empties seats, etc.).

2. **Force Vacate**:
	- As an admin, you can forcibly remove a referee from a seat if the referee cannot do so themselves. This is done from the Admin Panel.

3. **Reset Instance**:
	- This fully resets the meet: empties all seats, clears the active user list, and returns to a “fresh” state.

4. **When the Admin logs out**:
	- The meet is reset automatically. (This is the intended design so the meet director can start each meet from a blank state when they log back in.)

---

## 8. Superuser (Sudo) Functions

If the `meet_creds.json` includes a **superuser** account, that user can:
1. **Log in** on the same `/login` page with the superuser’s username/password.
2. Automatically be redirected to `/sudo`.
3. **Sudo Panel**:
	- See every active meet.
	- Forcibly vacate a referee in any meet.
	- Forcibly log out the meet director in any meet.
	- Restart and clear a meet from outside.

This role is primarily a fallback if the normal admin cannot resolve a problem or if you need to manage multiple meets.

---

## 9. Timers & Lights

1. **Platform Ready Timer**
	- Admin or center ref can trigger the “Start Timer” from their control pages, which initiates a countdown (default 60 seconds) for the lifter to begin.
	- If the time expires, “platformReadyExpired” is broadcast, and the attempt times out.
	- The timer can be reset with “Reset Timer.”

2. **Next Attempt Timer**
	- After an attempt completes, “startNextAttemptTimer” may be triggered to count down until the next attempt. This is also broadcast to all connected clients.

3. **Lights**
	- Each referee seat can submit a white or red decision. Once all three seats have decided:
		- The system broadcasts final results for a specified duration (default 15 seconds).
		- Then automatically clears the lights for the next attempt.

---

## 10. Dynamic Logos

- Every meet in `meets.json` or `meet_creds.json` can specify a `"logo"` field.
- When you select that meet, the system will display the custom logo on the main index page and other relevant pages.

---

## 11. Troubleshooting & Common Issues

1. **404 Page Not Found**
	- Ensure you have the correct route. If you see a 404 after pressing **Vacate** or **Logout**, verify the server is running and the route definitions match.
	- The code is designed to redirect to `/index` after a successful vacate or logout. Double-check that your environment is set up properly if you see an error.

2. **Session or Cookie Problems**
	- If your session isn’t persisting, check that cookies are enabled in the browser. Also confirm that the cookie store is properly configured with `router.Use(sessions.Sessions("mySession", store))`.

3. **Forcing a Refresh**
	- If a referee’s connection fails or the seat is stuck, the admin or superuser can forcibly vacate that seat from the admin or sudo panel.

4. **Not seeing a meet**
	- Confirm your meet is listed in `meets.json` and that you haven’t typed the name incorrectly.
	- If you added a new meet, you must restart the server to pick up changes.

---

## 12. Security & Best Practices

1. **Use HTTPS** in production. If you place go-ref-lights behind a secure reverse proxy or a platform like Nginx, ensure correct certificate handling.
2. **Keep Admin / Superuser Passwords Secure**. Bcrypt-hash all passwords.
3. **Limit Access**. If run publicly, restrict who can sign in or claim seats.

---

## 13. Additional Resources

- **Configuration Files**:
	- `./config/meet_creds.json` – controls user credentials (admins, superuser).
	- `./config/meets.json` – controls meet display names and potential logos.
- **Templates**:
	- `./templates/` – HTML templates for the web interface.
- **Static Assets**:
	- `./static/` – CSS, images, and JS.

---

## 14. Conclusion

With **go-ref-lights**, meet directors and referees can manage and visualize attempts, decisions, and timers in real time. The built-in Admin Panel and Sudo fallback mode provide robust control in case of disruptions. Refer to the sections above if you encounter any issues, and feel free to customize the code and templates to suit your federation’s style or requirements.

## 15. Logging and Troubleshooting

### Log Files
- Application logs are stored in the `./logs/` directory with timestamped filenames
- In production mode, logs use structured JSON format for easy parsing
- Development mode provides verbose logging for debugging

### Environment-Based Log Levels

The logging system automatically configures based on the ENV environment variable:

| Environment | Log Level | DEBUG | INFO | WARN | ERROR | File Logging |
|-------------|-----------|-------|------|------|-------|--------------|
| production  | WARN      | ❌    | ⚠️*   | ✅    | ✅     | ✅           |
| development | DEBUG     | ✅    | ✅    | ✅    | ✅     | ✅           |
| test        | WARN      | ❌    | ❌    | ✅    | ✅     | ❌           |

*Critical INFO only in production

**Configuration Examples:**
```bash
ENV=production ./go-ref-lights          # Production logging
ENV=development ./go-ref-lights         # Verbose logging
ENV=test ./go-ref-lights                # Test logging
LOG_LEVEL=DEBUG ./go-ref-lights         # Override level
```

**Optimizations Applied:**
- **WebSocket operations**: Connection upgrades, message processing, referee registration → DEBUG level
- **Timer operations**: Routine countdown updates, timer state changes → DEBUG level
- **HTTP requests**: Successful requests → DEBUG level, errors preserved with context
- **Performance**: Conditional logging prevents expensive operations when disabled

### Common Log Patterns
```bash
# View recent logs
tail -f logs/$(ls logs/ | tail -1)

# Search for errors
grep "ERROR" logs/*.log

# Parse JSON logs in production
cat logs/*.log | jq '.level, .message, .context'

# WebSocket issues only
cat logs/*.log | jq 'select(.context.component == "websocket" and (.level == "ERROR" or .level == "WARN"))'
```

### Testing the Logging System

The application includes comprehensive logging tests:

```bash
# Run unit tests
go test -v ./logger/

# Run integration tests (environment configuration testing)
go test -v -tags=integration ./logger/

# Run with coverage
go test -coverprofile=coverage.out ./logger/
go tool cover -func=coverage.out
```

**Integration tests validate:**
- Environment-based configuration (all ENV and LOG_LEVEL combinations)
- File logging behavior (production vs test mode)
- Log level filtering and message suppression
- Thread safety and concurrent access
- Edge cases (invalid values, case sensitivity)
- Performance and log file size monitoring

### Troubleshooting with Logs
- WebSocket connection issues: Look for "websocket" component ERROR/WARN logs
- Authentication problems: Check "authentication" component logs
- Timer issues: Search for "timer" component logs
- Position conflicts: Look for "position" component logs

**WebSocket Specific Issues:**
- Connection failures: Search for "connection_upgrade_failed" action
- Message delivery problems: Look for "message_write_failed" or "message_dropped" actions
- Client issues: Check for "json_parse_error" or "incomplete_decision" actions
- Network problems: Search for "ping_failed" action

**Configuration Issues:**
- Logs not appearing: Check ENV and LOG_LEVEL settings
- Too many logs in production: Ensure ENV=production (not development)
- Missing debug logs: Set LOG_LEVEL=DEBUG or ENV=development
- File not created in tests: Expected behavior - test mode uses stdout/stderr only

---

Enjoy hosting powerlifting meets with **go-ref-lights**! If you have further questions, check the structured logs for debugging info or consult the code to extend functionality.

---

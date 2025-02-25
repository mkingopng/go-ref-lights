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
# Issues identified during testing
- the claim positions page (http://localhost:8080/positions) needs formatting/styling.
- the lights page is showing the "platform ready" button. it should not
- the next lifter timer should be triggered and appear on the lights page once the 3 referee decisions are received and displayed. This is not happening. No "next lifter" timer is displayed.
- we have the "time out" message appearing on the lights page. This is now obsolete, and should be removed.
- the "reset" button on the center referee page is not working. It should reset the decisions and the timer.
- each referee position should show a message indicating the results of the health check. Maybe "connected" in green or "disconnected" in red.

# What’s Left To Do

1. Health Check Mechanism
   - Status: Partially done.
       - You’ve implemented logic that checks whether left, centre, and right are connected (and blocks timer starts if any are missing).
       - If you need a more thorough health-check flow (e.g. show a “Not ready” banner, notify meet directors, or auto-stop an active lift if a ref disconnects), that’s still a next-level enhancement.

2. Referee Position Control
   - Status: Improved but still open for refinement.
       - You have single-session enforcement plus a mechanism so referees can’t double-book a position.
       - For truly robust role switching (e.g. requiring admin approval, or automatically removing a ref after prolonged disconnection), you may still need more logic.

3. Multiple Meets / Scalability
   - Status: You’ve just taken a big step toward multi-meet support by scoping timers and referees to a `meetName`.
       - If you want to complete the feature, you might:
           - Provide a UI or admin page to manage meets (create, list, archive).
           - Persist meet states in DynamoDB (or another DB) so they survive restarts.
           - Thoroughly test running two meets in parallel.

4. Detailed Logging & Centralized Monitoring
   - Status: Still open.
       - You have basic logging to stdout.
       - For production readiness, consider structured logs (JSON), log aggregation (CloudWatch, ELK), and more informative levels (INFO, WARN, ERROR).

5. User Instructions & Docs
   - Status: Not addressed yet.
       - You might create a simple doc or web page explaining:
           - How referees log in and claim positions,
           - What the lights mean,
           - The role of the meet director (how they start and end meets, manage referees, etc.).

6. Auto-Reset Decisions After 15s
   - Status: Partially addressed or easy to adjust.
       - You currently wait 15 seconds after final decisions.
       - If you need a different time, adjust `resultsDisplayDuration`.

7. UI Tweaks
   - Status: Up to you.
       - Increase green dot size, keep text messages for 15 seconds, and do any other styling improvements.
       - These are quick adjustments in your CSS and JavaScript.

8. Upgrade to Full OAuth 2.0
   - Status: You do have a basic Google OAuth flow, so you’re partway there.
       - If you need advanced OAuth scenarios, like offline tokens, refresh tokens, or a custom OAuth provider, that could be the next step.

9. Deploy to Cloud
   - Status: Possibly partial or planned.
       - You have a Dockerfile and some AWS CDK scripts.
       - If your goal is to set up a fully automated CI/CD pipeline and run in ECS, you can finalise your build pipeline, environment configs, and domain (like `referee-lights.michaelkingston.com.au`).

---

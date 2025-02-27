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

8. downgrade to manually defined auth

9. Deploy to Cloud
   - Status: Possibly partial or planned.
       - You have a Dockerfile and some AWS CDK scripts.
       - If your goal is to set up a fully automated CI/CD pipeline and run in ECS, you can finalise your build pipeline, environment configs, and domain (like `referee-lights.michaelkingston.com.au`).

--

The logs indicate that the lights page’s WebSocket connection is still 
using the literal string `"{{ .meetId }}"` instead of the actual meet 
identifier. That means when you click “Platform Ready” (or other actions) 
from the lights page, the outgoing messages do not include the required 
meetName, so the server rejects them.

Two things to check:

1. In your `lights.html` template, verify that the template variables are 
being substituted. It should look like this:

```javascript
<script>
    var meetId = "{{ .meetId }}";
    const websocketUrl = "{{ .WebsocketURL }}?meetName={{ .meetId }}";
</script>
```

If the rendered HTML still shows `“{{ .meetId }}”` literally (as your log shows: 
`GET /referee-updates?meetName=%7B%7B.meetId%7D%7D)`, then the template isn’t 
being processed correctly. 

Ensure that:
- The file is indeed located in your templates directory.
- Your Lights controller function is passing a valid, non‐empty meetId 
  (which appears to be the case).
- There’s no caching or override preventing proper template processing.
- Also, in your lights.js (or the code that sends messages from the lights 
  page), ensure that every outgoing message includes the meetName property. 
  For example, before sending a message like `{"action":"startTimer"}`, you 
  should add: `message.meetName = meetId;`

This guarantees that when the server (in handler.go) does `r.URL.Query().Get
("meetName")` and later inspects incoming JSON messages for the meetName 
field, it finds the proper value.

By confirming that your lights.html template is processed properly and 
updating your client-side code to attach meetName in outgoing messages, 
your “Platform Ready” command should trigger the expected response on the 
lights page.

These adjustments should resolve the issue. Let me know if you need further clarification!
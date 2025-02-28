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
       - You’ve implemented logic that checks whether left, center, and right are connected (and blocks timer starts if any are missing).
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

---
Lets trace where the functions come from in GO from the beginning:

step 1: lets start with the startPlatformReadyTimer() function which lives in websocket/timer.go This is the timer that should be triggered by the center referee pressing the "Platform Ready" button. Lets start by adding logging to to this function. First, we should log when the timer starts running.

step 2) the startPlatformReadyTimer() function is called in a second function in the same file:

we should add a logging call at the end of the second function handleTimerAction() when the startPlatformReadyTimer(meetState) is is called

step 3: handleTimerAction() is then called in handleReads() which is in websocket/connection.go

step 4: handleReads() is then called in the ServeWs() function which is in websocket/connection.go:

step 5: then, ServeWs() is called by the main() function in main.go.

step 6 & 7: i'm not yet clear on how the timer related function is then passed to center.html, referee-common.js, lights.js and lights.html, however I'm there must me a listener that "hears" messages from this cascade of functions.

I think that we can add logging to each of these steps to trace what happens when the button "Platform Ready" is pressed by the center referee

what do you think? I've attached the files I've mentioned. I think that if we can get the logging right, we can trace what happens once the button is pressed, then where the problem arises.

I do not beleive that it is because the 4 pages don't have the same meetName. I've put a visual cue (variable at the top of each page (html) and its really clear that they are all using "Complete Strength Open" for these tests. So i disagree with your hypothesis

----

As always, the instructions are: I want clearly commented files.
I don't do copy-paste, so I want you to clearly show me where you insert new 
blocks and where you delete code and make changes so that I can follow along 
and make the changes in my own code. While I am most focused on getting this 
application to work, I also want to adhere to the best software engineering 
practice. I don't want to accumulate technical debt. Thank you in advance for 
your help

**Dynamic Occupied Positions**: If you want the positions.html to automatically 
refresh occupant statuses, you can do an AJAX poll or a little WebSocket logic. 
That’s a separate discussion.
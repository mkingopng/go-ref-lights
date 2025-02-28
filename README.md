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
I have a list of outstanding questions and issues that we need to work through:

- **Problem 1)** From the home page, regardless of what meet I select from the 
  list, when I click "proceed" and move to Log in, I always get the meet from 
  the top of the list.
  Why?
  I need this meet list to actually work.
  I need to be able to choose from multiple meets or platforms
  and have them operating in tandem.
  That's the whole point of what we've been doing.
  Why is this happening?
  refer to these logs:

(go-ref-lights-LCSWSeQ9-py3.10)(base) ~/Documents/GitHub/go-ref-lights git:[multi-meet-features]
go run main.go
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
INFO: 2025/03/01 01:45:36 main.go:41: [main] Starting application on port :8080
INFO: 2025/03/01 01:45:36 main.go:45: [main] Setting up routes & sessions...
INFO: 2025/03/01 01:45:36 main.go:75: Application started successfully.
INFO: 2025/03/01 01:45:36 page_controller.go:131: SetConfig: Global config updated: ApplicationURL=http://localhost:8080, WebsocketURL=ws://localhost:8080/referee-updates
Templates Path: /home/noone/Documents/GitHub/go-ref-lights/templates
DEBUG: 2025/03/01 01:45:36 position_controller.go:21: NewPositionController: Initializing PositionController
INFO: 2025/03/01 01:45:36 main.go:198: [main] About to run gin server on :8080
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
WARN: 2025/03/01 01:46:05 auth_controller.go:120: LoginHandler: Invalid login attempt for user south_australian_state_championships_1 at meet Complete Strength Open
^Csignal: interrupt

- **Problem 2)** I have tested using the same log-in creds in multiple browser 
  windows, and it is possible.
  That is concerning.
  Once its deployed, each 
  set of creds should only allow you to log in once.
  Admittedly users will be logging in from different devices;
  however, I see no reason to think that the behaviour will be different.
  How can we fix this?
  If two users try to log in with the same creds the app login should fail 
  elegantly.
  How can we achieve this?

- **Problem 3)** I have tried to occupy the same position from two browser 
  windows, and it fails.
  This is good.
  Unfortunately it doesn't fail very elegantly.
  You see a blank screen with a 404 error.
  We should be able to do better than that.
  We need to have a more elegant mechanism for informing the user
  that they cannot have two people in the same referee position.

- **Problem 4)** I can see that the QR code is being generated, but it is not 
  being displayed in the dashboard page.
  Why?
  How do we fix this?
  Refer to these logs:

(go-ref-lights-LCSWSeQ9-py3.10)(base) ~/Documents/GitHub/go-ref-lights git:[multi-meet-features]
go run main.go
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
INFO: 2025/03/01 02:03:12 main.go:41: [main] Starting application on port :8080
INFO: 2025/03/01 02:03:12 main.go:45: [main] Setting up routes & sessions...
INFO: 2025/03/01 02:03:12 main.go:75: Application started successfully.
INFO: 2025/03/01 02:03:12 page_controller.go:131: SetConfig: Global config updated: ApplicationURL=http://localhost:8080, WebsocketURL=ws://localhost:8080/referee-updates
Templates Path: /home/noone/Documents/GitHub/go-ref-lights/templates
DEBUG: 2025/03/01 02:03:12 position_controller.go:21: NewPositionController: Initializing PositionController
INFO: 2025/03/01 02:03:12 main.go:198: [main] About to run gin server on :8080
Loaded meets: [{Complete Strength Open March 1 [{complete_strength_open_1 $2b$12$lLFqMi8aPNIpF5.xA1PQ1.RTn56hExurXtTGZR167M.zNra./kjfe} {complete_strength_open_2 $2b$12$tG.GjVTKkp7z44QfeAD2Xe9AMjqgNrWrLYQ3.gC/wFF1Tux8P9gCK} {complete_strength_open_3 $2b$12$oKfdxuJaM7eJyRnrE3WHteeonP4T6N5O7jbSazzNok03ccjgjXk32} {complete_strength_open_4 $2b$12$nYqPV4/I8cJIjtNSBdO8OOjpl77z1eqsihDvCJree3RosAqoLLg8i}]} {South Australian State Championships March 2 [{south_australian_state_championships_1 $2b$12$kW0eQLSnhgW9bIC2SRJ1BeWhMk5jbCcnuwcDwkZm89on6i9b0B/Pe} {south_australian_state_championships_2 $2b$12$MsgVifJgmirgeUwbOX4GquzZ/C85DUctn1G2J48K.AF.i2Y3EWyD2} {south_australian_state_championships_3 $2b$12$PXmPCe1QTVTjinecFi9KjOcmhniENrQ.NKE1fZUc7ELsr3M4Mk5z.} {south_australian_state_championships_4 $2b$12$/3XCbsLd8DNEBz1BgUaXsO6uWEABtt5xmaHG7P674z99WvQ4iRvra}]} {Metal Mayham March 2 [{metal_mayham_1 $2b$12$jWq5JdKc2wVnx8sIBoTlsOYdqnDs9cYrKLdzDcKQ11Oovzmte6O26} {metal_mayham_2 $2b$12$EgTYx26aTc20Yd1oAeQ4XuRjqHZO7fegjVWbzDPRBuRZWDbi0H9Cu} {metal_mayham_3 $2b$12$NlsnmlADDpfEEn2pbFCr0u8LYg7ARsV262eIKmNCqfuh5V5e3N6pa} {metal_mayham_4 $2b$12$leXL9jm/czzFJiTjYlgMaO0U4oyw1ZBc2qIhlW.007j9HklIG542C}]}]
INFO: 2025/03/01 02:03:29 auth_controller.go:139: LoginHandler: User complete_strength_open_1 authenticated for meet Complete Strength Open
DEBUG: 2025/03/01 02:03:29 role.go:35: No specific role required for path: /dashboard
DEBUG: 2025/03/01 02:03:29 role.go:46: User complete_strength_open_1 authorized for position  on path /dashboard
INFO: 2025/03/01 02:03:29 page_controller.go:55: Rendering index page for meet Complete Strength Open
DEBUG: 2025/03/01 02:03:29 role.go:35: No specific role required for path: /qrcode
DEBUG: 2025/03/01 02:03:29 role.go:46: User complete_strength_open_1 authorized for position  on path /qrcode
INFO: 2025/03/01 02:03:29 page_controller.go:117: GetQRCode: Generating QR code
^Csignal: interrupt

- **Problem 5)** The position page has a drop-down list for each of the 
  referee positions.
  It is meant to show both the position
  and whether it has already been claimed or if it is available.
  This dynamic functionality is not working.
  The drop-down always shows that the referee's position is available.

- **Problem 6)** I need to generate a user manual for the app. Is there a 
  way I can do this automatically/programmatically? I really can't be bothered 
  spending a lot of time writing one. Writing the code has been hard enough.

- **problem 7)** how can a referee change positions mid-meet? It would be good 
  to have an easy mechanism. I don't think we currently have a mechanism once 
  the referee has claimed a position. I need to create a mechanism to 
  allow a user to leave a position, and free it up for another user to take it.
  i think this requires us to use `UnsetPosition` from occupancy_service.go

- **problem 8)** 1s != 1s (Platform Timer Goes Too Fast)
  Ticker drift can happen due to:
- The environment (a busy CPU / GC cycles might cause faster or slower
  intervals).
- The front end might see events arrive slightly off.
- If you have multiple tabs open with the same timer, each might show the
  timer differently.
- If your system is under load, or the user’s OS clock is messed up, or you
  do nested setIntervals in JavaScript.

Possible solutions:
- Use a “time-based” approach, e.g. store the “startTime = time.Now() + 60
  sec.” Then on each tick, you do timeLeft = endTime - time.Now(), so it
  can’t drift.
- Make sure you only have one instance of the same timer running.
- Tolerate small scheduling drift, which is normal with time.Ticker.

Let's fix this ticker drift problem, and all the rest, one at a time

- **Problem 9)** CDK deployment to my page.

- **Problem 10)** long-term deployment to APL. Nick to advise on logo, style, 
  domain, etc.
- Fix NPM vulnerabilities
- Optimise multi-threading
- Improve appearance and formatting

----

# advanced tasks
- Ci/CD
- pre-commit hooks
- unit tests
- integration tests
- improved formatting

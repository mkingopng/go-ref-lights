// static/js/lights.js
"use strict";

let socket;
let platformReadyInterval = null;
let resultsDisplayed = false; // Flag to indicate that displayResults has been processed

// utility function for logging
function log(message, level = 'debug') {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] ${level.toUpperCase()}: ${message}`;

    // log to console
    switch (level) {
        case 'error':
            console.error(logMessage);
            break;
        case 'warn':
            console.warn(logMessage);
            break;
        case 'debug':
            console.debug(logMessage);
            break;
        default:
            console.log(logMessage);
    }

    // send logs to a server for saving to a file
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', error));
}

let nextAttemptTimers = {};
const multiNextAttemptTimers = document.getElementById("multiNextAttemptTimers");

window.addEventListener("DOMContentLoaded", function () {

    const leftCircle = document.getElementById("leftCircle");
    const centerCircle = document.getElementById("centerCircle");
    const rightCircle = document.getElementById("rightCircle");

    const leftIndicator = document.getElementById("leftIndicator");
    const centerIndicator = document.getElementById("centerIndicator");
    const rightIndicator = document.getElementById("rightIndicator");

    // helper function to get a consistent meet name from the DOM/URL/sessionStorage
    function getMeetName() {
        let elem = document.getElementById("meetName");
        let meetName = elem ? elem.dataset.meetName : null;
        if (!meetName) {
            meetName = sessionStorage.getItem("meetName") || new URLSearchParams(window.location.search).get("meetName");
        }
        if (meetName) {
            sessionStorage.setItem("meetName", meetName);
            log(`✅ Meet name set: ${meetName}`, "info");
        } else {
            log("⚠️ Meet name is missing! Redirecting to meet selection.", "warn");
            alert("Error: No meet selected. Redirecting.");
            window.location.href = "/meets";
        }
        return meetName;
    }

    // Helper function to update the platform ready timer UI
    function updatePlatformReadyTimer(timer) {
        log(`Updating Platform Ready Timer: ${timer.TimeLeft}s`, "debug");
        if (timerDisplay) {
            timerDisplay.innerText = `${timer.TimeLeft}s`;
        }
        // Hide the container if the timer ran out
        if (timer.TimeLeft <= 0 && platformReadyTimerContainer) {
            platformReadyTimerContainer.classList.add("hidden");
        } else if (platformReadyTimerContainer) {
            platformReadyTimerContainer.classList.remove("hidden");
        }
    }

    // Helper function to update a next attempt timer UI element
    function updateNextAttemptTimer(timer, container, timersMap) {
        // If time is up, remove the timer element
        if (timer.TimeLeft <= 0) {
            if (timersMap[timer.ID]) {
                container.removeChild(timersMap[timer.ID]);
                delete timersMap[timer.ID];
            }
        } else {
            // Create the timer element if it doesn't exist
            if (!timersMap[timer.ID]) {
                let newRow = document.createElement("div");
                newRow.classList.add("timer");
                container.insertBefore(newRow, container.firstChild);
                timersMap[timer.ID] = newRow;
            }
            // Update the timer element's text without the index number
            timersMap[timer.ID].textContent = `Next Attempt: ${timer.TimeLeft}s`;
            container.classList.remove("hidden");
        }
    }

    // Main handler for the "updateNextAttemptTime" action
    function handleUpdateNextAttemptTime(data) {
        if (data.timers && Array.isArray(data.timers)) {
            data.timers.forEach(timer => {
                if (timer.ID === 1) {
                    if (resultsDisplayed) {
                        // When results are displayed, treat timer ID 1 as a next attempt timer
                        updateNextAttemptTimer(timer, multiNextAttemptTimers, nextAttemptTimers);
                    } else {
                        // Otherwise, update the platform ready timer
                        updatePlatformReadyTimer(timer);
                    }
                } else {
                    // For other timer IDs, update the next attempt timer normally
                    updateNextAttemptTimer(timer, multiNextAttemptTimers, nextAttemptTimers);
                }
            });
        }
    }

    // constants
    const meetName = getMeetName();  // use the verified meet name for later actions
    if (!meetName) return;
    const judgeId = "lights";

    // initialise the global WebSocket object
    const scheme = (window.location.protocol === "https:") ? "wss" : "ws";
    const wsUrl = `${scheme}://${window.location.host}/referee-updates?meetName=${meetName}`;
    socket = new WebSocket(wsUrl);

    // grab common DOM elements
    const timerDisplay = document.getElementById('timer');
    const healthEl = document.getElementById("healthStatus");
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');

    // set up WebSocket event handlers
    socket.onopen = function () {
        log("✅ WebSocket connection established (Lights).", "info");
        const statusEl = document.getElementById("connectionStatus");
        if (statusEl) {
            statusEl.innerText = "Connected";
            statusEl.style.color = "green";
        }
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId,  // "lights"
            meetName: meetName
        };
        socket.send(JSON.stringify(registerMsg));
        log(`Sent registerRef for lights with meetName=${meetName}`, "info");
    };

    socket.onclose = function (event) {
        log(`⚠️ WebSocket connection closed (Lights): ${event.code} - ${event.reason}`, "warn");
        const statusEl = document.getElementById("connectionStatus");
        if (statusEl) {
            statusEl.innerText = "Disconnected";
            statusEl.style.color = "red";
        }
    };

    socket.onerror = function (error) {
        log(`⚠️ WebSocket error: ${error}`, "error");
    };

    log("DOM fully loaded and parsed");

    socket.onmessage = function (event) {
        let data;
        try {
            data = JSON.parse(event.data);
            log(`📩 WebSocket message received: ${JSON.stringify(data)}`, 'debug');
        } catch (e) {
            log(`Invalid JSON from server: ${event.data}`, 'error');
            return;
        }

        switch (data.action) {
            case "refereeHealth":
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;

            case "healthError":
                alert(data.message);
                break;

            case "startTimer":
                log("🔵 Received startTimer from server, starting Platform Ready Timer countdown");

                resultsDisplayed = false;

                if (platformReadyInterval) {
                    clearInterval(platformReadyInterval);
                    platformReadyInterval = null;
                    }

                Object.keys(nextAttemptTimers).forEach((id => {
                    if (multiNextAttemptTimers && nextAttemptTimers[id]) {
                        multiNextAttemptTimers.removeChild(nextAttemptTimers[id]);
                    }
                    delete nextAttemptTimers[id];
                }));
                if (multiNextAttemptTimers) {
                    multiNextAttemptTimers.classList.add("hidden");
                }
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
                if (timerDisplay) {
                    timerDisplay.innerText = `${data.timeLeft}s`;
                }
                break;

            case "updatePlatformReadyTime":
                log(`⌛ Handling updatePlatformReadyTime: ${data.timeLeft}s left`, "debug");

                // If you want to share logic with the "startTimer" local countdown,
                // you can either replicate the code or unify them. For a quick fix:
                if (data.timeLeft <= 0) {
                    // Hide the timer since it's expired
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.add("hidden");
                    }
                } else {
                    // Show the timer container if hidden
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.remove("hidden");
                    }
                    if (timerDisplay) {
                        timerDisplay.innerText = `${data.timeLeft}s`;
                    }
                }
                break;

            case "updateNextAttemptTime":
                log("✅ Entering handleUpdateNextAttemptTime", "debug");
                handleUpdateNextAttemptTime(data);
                break;

            case "judgeSubmitted":
                log(`[lights.js] Judge ${data.judgeId} has submitted a decision.`);
                if (data.judgeId === "left") {
                    leftIndicator.style.backgroundColor = "green";
                } else if (data.judgeId === "center") {
                    centerIndicator.style.backgroundColor = "green";
                } else if (data.judgeId === "right") {
                    rightIndicator.style.backgroundColor = "green";
                }
                break;

            case "displayResults":
                log(`Final decisions: L=${data.leftDecision}, C=${data.centerDecision}, R=${data.rightDecision}`);
                log(`[lights.js] displayResults received: left=${data.leftDecision}, center=${data.centerDecision}, right=${data.rightDecision}`);

                leftCircle.style.backgroundColor   = (data.leftDecision   === "white") ? "white" : "red";
                centerCircle.style.backgroundColor = (data.centerDecision === "white") ? "white" : "red";
                rightCircle.style.backgroundColor  = (data.rightDecision  === "white") ? "white" : "red";

                // Determine the message to display
                let whiteCount = 0;
                let redCount = 0;

                [data.leftDecision, data.centerDecision, data.rightDecision].forEach(decision => {
                    if (decision === "white") {
                        whiteCount++;
                    } else {
                        redCount++;
                    }
                });

                const messageEl = document.getElementById("message");

                if (whiteCount >= 2) {
                    messageEl.innerText = "Good Lift";
                    messageEl.style.color = "green";
                } else {
                    messageEl.innerText = "No Lift";
                    messageEl.style.color = "red";
                }

                messageEl.classList.add("flash");

                // Clear message after 15 seconds
                setTimeout(() => {
                    messageEl.innerText = "";
                    messageEl.classList.remove("flash");
                }, 15000);

                // Mark that the results have been displayed so that next attempt timers are handled separately
                resultsDisplayed = true;
                break;

            case "platformReadyExpired":
                log("⏰ Platform Ready Timer Expired!");
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.add("hidden");
                }
                break;

            case "clearResults":
                log("Clearing results from Lights UI (white vs red circles, judge indicators).");
                leftCircle.style.backgroundColor   = "black";
                centerCircle.style.backgroundColor = "black";
                rightCircle.style.backgroundColor  = "black";
                leftIndicator.style.backgroundColor   = "grey";
                centerIndicator.style.backgroundColor = "grey";
                rightIndicator.style.backgroundColor  = "grey";

                // Clear message
                messageEl.innerText = "";
                messageEl.classList.remove("flash");
                resultsDisplayed = false;
                break;

            case "resetLights":
                log("🌀 Resetting lights to black");
                leftCircle.style.backgroundColor   = "black";
                centerCircle.style.backgroundColor = "black";
                rightCircle.style.backgroundColor  = "black";
                break;

            default:
                log(`⚠️ Unknown action: ${data.action}`, "warn");
        }
    };
});

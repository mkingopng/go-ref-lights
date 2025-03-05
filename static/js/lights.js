"use strict";

let socket;
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
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
                // Start a local countdown from 60 seconds
                let timeLeft = 60;
                if (timerDisplay) {
                    timerDisplay.innerText = `${timeLeft}s`;
                }
                const countdownInterval = setInterval(() => {
                    timeLeft--;
                    if (timerDisplay) {
                        timerDisplay.innerText = `${timeLeft}s`;
                    }
                    if (timeLeft <= 0) {
                        clearInterval(countdownInterval);
                        if (platformReadyTimerContainer) {
                            platformReadyTimerContainer.classList.add("hidden");
                        }
                    }
                }, 1000);
                break;

            case "updateNextAttemptTime":
                // Loop through all timer objects received
                if (data.timers && Array.isArray(data.timers)) {
                    data.timers.forEach(timer => {
                        if (timer.ID === 1) {
                            // If results have been displayed, use timer ID 1 for next attempt timer
                            if (resultsDisplayed) {
                                // Update the next attempt timer (using same ID)
                                if (timer.TimeLeft <= 0) {
                                    if (nextAttemptTimers[timer.ID]) {
                                        multiNextAttemptTimers.removeChild(nextAttemptTimers[timer.ID]);
                                        delete nextAttemptTimers[timer.ID];
                                    }
                                } else {
                                    if (!nextAttemptTimers[timer.ID]) {
                                        let newRow = document.createElement("div");
                                        newRow.classList.add("timer");
                                        multiNextAttemptTimers.insertBefore(newRow, multiNextAttemptTimers.firstChild);
                                        nextAttemptTimers[timer.ID] = newRow;
                                    }
                                    nextAttemptTimers[timer.ID].textContent = `Next Attempt: ${timer.TimeLeft}s`;
                                    multiNextAttemptTimers.classList.remove("hidden");
                                }
                            } else {
                                // Otherwise, update the platform ready timer as before
                                log(`Updating Platform Ready Timer: ${timer.TimeLeft}s`, "debug");
                                if (timerDisplay) {
                                    timerDisplay.innerText = `${timer.TimeLeft}s`;
                                }
                                if (timer.TimeLeft <= 0 && platformReadyTimerContainer) {
                                    platformReadyTimerContainer.classList.add("hidden");
                                } else if (platformReadyTimerContainer) {
                                    platformReadyTimerContainer.classList.remove("hidden");
                                }
                            }
                        } else {
                            // If timer ID is not 1, update next attempt timers as before
                            if (timer.TimeLeft <= 0) {
                                if (nextAttemptTimers[timer.ID]) {
                                    multiNextAttemptTimers.removeChild(nextAttemptTimers[timer.ID]);
                                    delete nextAttemptTimers[timer.ID];
                                }
                            } else {
                                if (!nextAttemptTimers[timer.ID]) {
                                    let newRow = document.createElement("div");
                                    newRow.classList.add("timer");
                                    multiNextAttemptTimers.insertBefore(newRow, multiNextAttemptTimers.firstChild);
                                    nextAttemptTimers[timer.ID] = newRow;
                                }
                                nextAttemptTimers[timer.ID].textContent = `Next Attempt #${timer.ID}: ${timer.TimeLeft}s`;
                                multiNextAttemptTimers.classList.remove("hidden");
                            }
                        }
                    });
                }
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

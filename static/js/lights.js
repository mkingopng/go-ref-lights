// static/js/lights.js
"use strict";

let socket;

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
    // validate required globals
    if (typeof websocketUrl === 'undefined') {
        log("websocketUrl is not defined", "error");
        return;
    }
    // validate required globals
    if (typeof websocketUrl === 'undefined') {
        log("websocketUrl is not defined", "error");
        return;
    }
    // 2) We still check meetName below, but we define judgeId ourselves
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

    //constants
    const meetName = getMeetName();  // use the verified meet name for later actions
    if (!meetName) return;
    const judgeId = "lights";

    // initialise the global WebSocket object (do not shadow the global 'socket')
    const wsUrl = `ws://localhost:8080/referee-updates?meetName=${meetName}`; // fix_me
    socket = new WebSocket(wsUrl);

    // grab common DOM elements
    const timerDisplay = document.getElementById('timer');
    const healthEl = document.getElementById("healthStatus");
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const multiNextAttemptTimers = document.getElementById('multiNextAttemptTimers');

    // set up WebSocket event handlers
    socket.onopen = function () {
        log("✅ WebSocket connection established (Lights).", "info");
        const statusEl = document.getElementById("connectionStatus");
        if (statusEl) {
            statusEl.innerText = "Connected";
            statusEl.style.color = "green";
        }
        // for sending registerRef, so the server knows we are "lights"
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

    // define a single onmessage handler for the WebSocket
    socket.onmessage = function (event) {
        let data;
        try {
            data = JSON.parse(event.data);
            log(`📩 WebSocket message received: ${JSON.stringify(data)}`, 'debug');
        } catch (e) {
            log(`Invalid JSON from server: ${event.data}`, 'error');
            return;
        }

        // process messages based on their action
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
                log("🔵 Received startTimer from server, showing Platform Ready Timer");
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
                break;

            case "updatePlatformReadyTime":
                log(`Updating platform ready timer: ${data.timeLeft}s`);
                if (timerDisplay) {timerDisplay.innerText = `${data.timeLeft}s`;
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

            case "updateNextAttemptTime":
                if (!data.index) break; // guard
                if (data.timeLeft <= 0) {
                    let row = nextAttemptTimers[data.index];
                    if (row) {
                        multiNextAttemptTimers.removeChild(row);
                        delete nextAttemptTimers[data.index];
                    }
                } else {
                    if (!nextAttemptTimers[data.index]) {
                        let newRow = document.createElement("div");
                        newRow.classList.add("timer");
                        multiNextAttemptTimers.insertBefore(newRow, multiNextAttemptTimers.firstChild);
                        nextAttemptTimers[data.index] = newRow;
                    }
                    nextAttemptTimers[data.index].textContent = `Next Attempt #${data.index}: ${data.timeLeft}s`;
                    multiNextAttemptTimers.classList.remove("hidden");
                }
                break;

            case "nextAttemptExpired":
                if (data.index && nextAttemptTimers[data.index]) {
                    multiNextAttemptTimers.removeChild(nextAttemptTimers[data.index]);
                    delete nextAttemptTimers[data.index];
                }
                break;

            default:
                log(`⚠️ Unknown action: ${data.action}`, "warn");
        }
    };
});

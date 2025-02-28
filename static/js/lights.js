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
}  //todo: where does this log go? can we have it print to console? I'm only seeing go logs in console and saving

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
        // todo: update status
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
                log("🔵 Received startTimer event from WebSocket");
                // todo: add code here to show/hide a timer (eg Platform ready timer or next attempt timer)
                break;
            case "updatePlatformReadyTime":
                log(`Updating platform ready timer: ${data.timeLeft}s`);
                if (timerDisplay) {
                    timerDisplay.innerText = `${data.timeLeft}s`;
                }
                break;
            case "platformReadyExpired":
                log("⏰ Platform Ready Timer Expired!");
                if (timerDisplay) {
                    timerDisplay.innerText = "EXPIRED";  // todo:
                                                         //  put back hidden parameter in lights.html
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
                // todo: paint the big circles
                const leftCircle   = document.getElementById("leftCircle");
                const centerCircle = document.getElementById("centerCircle");
                const rightCircle  = document.getElementById("rightCircle");
                leftCircle.style.backgroundColor   = (data.leftDecision   === "white") ? "white" : "red";
                centerCircle.style.backgroundColor = (data.centerDecision === "white") ? "white" : "red";
                rightCircle.style.backgroundColor  = (data.rightDecision  === "white") ? "white" : "red";
                break;

            case "updateNextAttemptTime":
                log(`Next Attempt Timer: ${data.timeLeft}s (Index: ${data.index || 1})`);
                // todo: update second timer:
                const secondTimer = document.getElementById("secondTimer");
                if (secondTimer) {
                    secondTimer.innerText = data.timeLeft + "s";
                }  // todo:
                   //  include an incrementing index
                   //  to differentiate multiple next attempt timers,
                   //  starting from 1 for each next attempt timer,
                   //  and they need to stack line-on-line on the lights window,
                   //  below the lights.
                   //  They persist until they reach zero then disappear
                break;

            case "nextAttemptExpired":
                log("Next attempt timer has expired; clearing or resetting UI");
                // todo: hide the next attempt timer after it reaches 0s
                break;


            default:
                log(`⚠️ Unknown action: ${data.action}`, "warn");
        }
    };
});

// todo: add back functions for next attempt timers, judge decision UI updates, health status updates

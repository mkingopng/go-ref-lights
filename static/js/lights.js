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

window.addEventListener("DOMContentLoaded", function () {
    if (!meetName) {
        log("⚠️ Meet name not found. WebSocket will not be initialized.", "error");
        return;
    }

    // create the WebSocket URL using the initial meet name
    let websocketUrl = `ws://localhost:8080/referee-updates?meetName=${meetName}`;  // fix_me for production
    socket = new WebSocket(websocketUrl);

    // set up WebSocket event handlers
    socket.onopen = function () {
        log("✅ WebSocket connection established (Lights).");
    };

    socket.onclose = function (event) {
        log(`⚠️ WebSocket connection closed (Lights): ${event.code} - ${event.reason}`, "warn");
    };

    socket.onerror = function (error) {
        log(`⚠️ WebSocket error: ${error}`, "error");
    };

    log("DOM fully loaded and parsed");

    // helper function to get a consistent meet name
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

    // use the verified meet name for later actions
    const meetName = getMeetName();
    if (!meetName) return;

    // cache DOM elements
    const timerDisplay = document.getElementById('timer');

    // define a single onmessage handler for the WebSocket
    socket.onmessage = (event) => {
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
            case "startTimer":
                log("🔵 Received startTimer event from WebSocket");
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
                    timerDisplay.innerText = "EXPIRED";
                }
                break;
            default:
                log(`⚠️ Unknown action: ${data.action}`, "warn");
        }
    };
});

// todo: add back functions for next attempt timers, judge decision UI updates, health status updates
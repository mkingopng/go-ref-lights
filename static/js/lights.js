// static/js/lights.js
"use strict";

let socket;

// Utility function for logging
function log(message, level = 'debug') {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] ${level.toUpperCase()}: ${message}`;

    // Log to console
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

    // Send logs to a server for saving to a file
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', error));
}

window.addEventListener("DOMContentLoaded", function () {
    let meetNameElement = document.getElementById("meetName");
    let meetName = meetNameElement ? meetNameElement.dataset.meetName : "";

    if (!meetName) {
        log("⚠️ Meet name not found. WebSocket will not be initialized.", "error");
        return;
    }

    let websocketUrl = `ws://localhost:8080/referee-updates?meetName=${meetName}`;
    socket = new WebSocket(websocketUrl);

    socket.onopen = function () {
        log("✅ WebSocket connection established (Lights).");
    };

    socket.onclose = function (event) {
        log(`⚠️ WebSocket connection closed (Lights): ${event.code} - ${event.reason}`, "warn");
    };

    socket.onerror = function (error) {
        log("⚠️ WebSocket error:", "error");
    };

    socket.onmessage = function (event) {
        let data;
        try {
            data = JSON.parse(event.data);
            log(`📩 Received WebSocket message: ${JSON.stringify(data)}`, 'debug');
        } catch (e) {
            log(`Invalid JSON from server: ${event.data}`, 'error');
            return;
        }
    };

    log("DOM fully loaded and parsed");

    function getMeetName() {
        let meetNameElement = document.getElementById("meetName");
        let meetName = meetNameElement ? meetNameElement.dataset.meetName : null;

        if (!meetName) {
            meetName = sessionStorage.getItem("meetName") || new URLSearchParams(window.location.search).get("meetName");
        }

        if (meetName) {
            sessionStorage.setItem("meetName", meetName);
            log(`✅ Meet name set: ${meetName}`);
        } else {
            log("⚠️ Meet name is missing! Redirecting to meet selection.", "warn");
            alert("Error: No meet selected. Redirecting.");
            window.location.href = "/meets";
        }
        return meetName;
    }

    const meetNameCheck = getMeetName();
    if (!meetNameCheck) return;

    // Cache DOM elements
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');

    // Platform Ready Button Logic
    if (platformReadyButton) {
        platformReadyButton.addEventListener("click", () => {
            if (socket.readyState === WebSocket.OPEN) {
                log("🟢 Platform Ready button clicked, sending startTimer action.");
                socket.send(JSON.stringify({ action: "startTimer", meetName: meetNameCheck }));
                platformReadyTimerContainer.classList.remove("hidden");
            } else {
                log("❌ WebSocket is not ready. Cannot send startTimer action.", "error");
            }
        });
    } else {
        log("⚠️ Platform Ready button not found.", "warn");
    }

    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
            log(`📩 WebSocket message received: ${JSON.stringify(data)}`, 'debug');
        } catch (e) {
            log(`Invalid JSON from server: ${event.data}`, 'error');
            return;
        }

        switch (data.action) {
            case "startTimer":
                log("🔵 Received startTimer event from WebSocket");
                break;
            case "updatePlatformReadyTime":
                log(`Updating platform ready timer: ${data.timeLeft}s`);
                if (timerDisplay) timerDisplay.innerText = `${data.timeLeft}s`;
                break;
            case "platformReadyExpired":
                log("⏰ Platform Ready Timer Expired!");
                if (timerDisplay) timerDisplay.innerText = "EXPIRED";
                break;
            default:
                log(`⚠️ Unknown action: ${data.action}`, "warn");
        }
    };
});

// todo: add back functions for next attempt timers, judge decision UI updates, health status updates
// static/js/referee-common.js
"use strict";

let socket;

// utility function for logging
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

    // send logs to a server for saving to a file
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', error));
}

document.addEventListener('DOMContentLoaded', function() {
    // validate required globals
    if (typeof websocketUrl === 'undefined') {
        log("websocketUrl is not defined", "error");
        return;
    }
    if (typeof judgeId === 'undefined') {
        log("judgeId is not defined", "error");
        return;
    }

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

    const meetName = getMeetName();  // CHANGE ME: Optionally rename to 'meetName' for consistency.
    if (!meetName) return;  // Stop if no meet name

    // Initialize the global WebSocket (do not shadow the global 'socket')
    const wsUrl = `ws://localhost:8080/referee-updates?meetName=${encodeURIComponent(meetName)}`; // CHANGE ME: Use encodeURIComponent.
    socket = new WebSocket(wsUrl);

    // grab common DOM elements
    const healthEl = document.getElementById("healthStatus");
    const whiteButton = document.getElementById('whiteButton');
    const redButton   = document.getElementById('redButton');
    const startTimerButton = document.getElementById('startTimerButton');
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');

    // platform Ready Button Logic
    if (platformReadyButton) {
        platformReadyButton.addEventListener("click", () => {
            if (socket.readyState === WebSocket.OPEN) {
                log("🟢 Platform Ready button clicked, sending startTimer action.", "info");
                // here, use the consistent meet name from getMeetName()
                socket.send(JSON.stringify({ action: "startTimer", meetName: meetName }));
                // Show the timer container
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
            } else {
                log("❌ WebSocket is not ready. Cannot send startTimer action.", "error");
            }
        });
    } else {
        log("⚠️ Platform Ready button not found.", "warn");
    }

    // WebSocket event: opened
    socket.onopen = function() {
        log(`WebSocket connected for judgeId: ${judgeId}`, "info");
        // immediately register as connected
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId,
            meetName: meetName
        };
        socket.send(JSON.stringify(registerMsg));
    };

    // WebSocket event: onmessage
    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            log(`Invalid JSON from server: ${event.data}. Error: ${e.message}`, "error"); // CHANGE ME: Include error details.
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
            default:
                log(`Unhandled action: ${data.action}`, "debug");
        }
    };

    // webSocket event: onerror
    socket.onerror = function(error) {
        log(`WebSocket error (${judgeId}): ${error}`, "error");
    };

    // webSocket event: onclose
    socket.onclose = function(event) {
        log(`WebSocket closed (${judgeId}): ${event.code} - ${event.reason}`, "info");
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // utility function to send JSON messages over the socket
    function sendMessage(obj) {
        if (socket.readyState === WebSocket.OPEN) {
            const messageString = JSON.stringify(obj)
            socket.send(messageString);
            log(`Sent message: ${messageString}`, "info");
        } else {
            log(`Cannot send message; socket not open (readyState = ${socket.readyState})`, "warn");
        }
    }

    // if these buttons exist, wire them up
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "white"
            });
        });
    }
    if (redButton) {
        redButton.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "red"
            });
        });
    }
    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            // Send multiple messages: reset lights, reset timer, then start timer
            sendMessage({
                action: "resetLights",
                meetName: meetName,
                judgeId: judgeId,
            });
            sendMessage({
                action: "resetTimer",
                meetName: meetName,
                judgeId: judgeId,
            });
            sendMessage({
                action: "startTimer",
                meetName: meetName,
                judgeId: judgeId,
            });
        });
    }
});

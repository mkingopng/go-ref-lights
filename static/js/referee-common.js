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
    const meetName = getMeetName();
    if (!meetName) return;

    // initialise the global WebSocket (do not shadow the global 'socket')
    const scheme = (window.location.protocol === "https:") ? "wss" : "ws";
    const wsUrl = `${scheme}://${window.location.host}/referee-updates?meetName=${meetName}`;
    socket = new WebSocket(wsUrl);

    // grab common DOM elements
    const healthEl = document.getElementById("healthStatus");
    const whiteButton = document.getElementById('whiteButton');
    const redButton   = document.getElementById('redButton');
    const startTimerButton = document.getElementById('startTimerButton');
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');

    // Platform Ready Button Logic:
    // only attach Platform Ready button event for center referee
    if (judgeId === "center") {
        const platformReadyButton = document.getElementById('platformReadyButton'); // Expect this element on center page only
        const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
        if (platformReadyButton) {
            platformReadyButton.addEventListener("click", () => {
                console.log("[referee-common.js] 'Platform Ready' button clicked; sending startTimer");
                if (socket.readyState === WebSocket.OPEN) {
                    log("🟢 Platform Ready button clicked, sending startTimer action.", "info");
                    socket.send(JSON.stringify({ action: "startTimer", meetName: meetName }));
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
    }

    // set up WebSocket event handlers
    // WebSocket event: onopen
    socket.onopen = function() {
        log(`WebSocket connected for judgeId: ${judgeId}`, "info");
        const registerMsg = { action: "registerRef", judgeId, meetName };
        socket.send(JSON.stringify(registerMsg));
    };

    // WebSocket event: onmessage
    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            log(`Invalid JSON: ${event.data} - ${e.message}`, "error");
            return;
        }

        switch (data.action) {
            case "occupancyChanged":
                console.log("Occupancy update received:", data);
                updateOccupancyUI(data);
                break;
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

// Function to update the UI
    function updateOccupancyUI(data) {
        document.getElementById("leftUser").innerText = data.leftUser || "Vacant";
        document.getElementById("centerUser").innerText = data.centerUser || "Vacant";
        document.getElementById("rightUser").innerText = data.rightUser || "Vacant";
    }

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

    // buttons
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            // CHANGE: we add a short guard log plus the actual send:
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "white"
            })
            log(`[RefereeCommon] Judge '${judgeId}' clicked GOOD LIFT (white).`, "info");
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
            log(`[RefereeCommon] Judge '${judgeId}' clicked NO LIFT (red).`, "info");
        });
    }
    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            // send multiple messages: reset lights, reset timer, start timer
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

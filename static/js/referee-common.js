// static/js/referee-common.js
"use strict";

// We'll define a global variable 'socket' so other code can reference it.
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

    // also send logs to server
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', error));
}

document.addEventListener('DOMContentLoaded', function() {

    // helper to get meetName from <div id="meetName" data-meet-name="foo">
    function getMeetName() {
        const elem = document.getElementById("meetName");
        let meetName = elem ? elem.dataset.meetName : null;

        // fallback to sessionStorage or URL query param
        if (!meetName) {
            meetName = sessionStorage.getItem("meetName") ||
                new URLSearchParams(window.location.search).get("meetName");
        }
        if (meetName) {
            sessionStorage.setItem("meetName", meetName);
            log(`✅ Meet name set: ${meetName}`, "info");
        } else {
            log("⚠️ Meet name is missing! Redirecting to /meets.", "warn");
            alert("Error: No meet selected. Redirecting.");
            window.location.href = "/meets";
        }
        return meetName;
    }

    // retrieve meetName
    const meetName = getMeetName();
    if (!meetName) return; // bail if no meet

    // build WebSocket URL (with correct scheme)
    const scheme = (window.location.protocol === "https:") ? "wss" : "ws";
    const wsUrl = `${scheme}://${window.location.host}/referee-updates?meetName=${meetName}`;

    // create Reconnecting WebSocket
    // (Requires reconnecting-websocket.min.js to be loaded first in the HTML)
    socket = new ReconnectingWebSocket(wsUrl, null, {
        reconnectInterval: 2000,   // 2 seconds
        maxReconnectAttempts: null // infinite
    });

    // references for your DOM elements
    const healthEl = document.getElementById("healthStatus");
    const whiteButton = document.getElementById('whiteButton');
    const redButton   = document.getElementById('redButton');
    const startTimerButton = document.getElementById('startTimerButton');
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');

    // single onopen
    socket.onopen = function() {
        log(`WebSocket connected for judgeId: ${judgeId}`, "info");
        // send a register message
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId,
            meetName: meetName
        };
        socket.send(JSON.stringify(registerMsg));
    };

    // onmessage => handle inbound messages
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
                // If data.connectedRefIDs includes me, I'm connected
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

    // handle errors
    socket.onerror = function(error) {
        log(`WebSocket error (${judgeId}): ${error}`, "error");
    };

    // handle close (the ReconnectingWebSocket will attempt reconnect automatically)
    socket.onclose = function(event) {
        log(`WebSocket closed (${judgeId}): ${event.code} - ${event.reason}`, "info");
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // function to update occupancy
    function updateOccupancyUI(data) {
        const leftUserEl   = document.getElementById("leftUser");
        const centerUserEl = document.getElementById("centerUser");
        const rightUserEl  = document.getElementById("rightUser");
        if (leftUserEl)   leftUserEl.innerText   = data.leftUser   || "Vacant";
        if (centerUserEl) centerUserEl.innerText = data.centerUser || "Vacant";
        if (rightUserEl)  rightUserEl.innerText  = data.rightUser  || "Vacant";
    }

    // convenience function for sending JSON messages
    function sendMessage(obj) {
        if (socket.readyState === WebSocket.OPEN) {
            const msgStr = JSON.stringify(obj);
            socket.send(msgStr);
            log(`Sent message: ${msgStr}`, "info");
        } else {
            log(`Cannot send message; socket not open (readyState = ${socket.readyState})`, "warn");
        }
    }

    // optionally attach logic for the center referee's "Platform Ready"
    if (judgeId === "center" && platformReadyButton) {
        platformReadyButton.addEventListener("click", () => {
            log("'Platform Ready' button clicked; sending startTimer", "debug");
            if (socket.readyState === WebSocket.OPEN) {
                sendMessage({ action: "startTimer", meetName: meetName });
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
            } else {
                log("❌ WebSocket is not ready. Cannot send startTimer action.", "error");
            }
        });
    }

    // handle White/Red button clicks
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "white"
            });
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

    // if you have a "startTimerButton" for something else
    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            // send multiple messages: reset lights, reset timer, start timer
            sendMessage({ action: "resetLights", meetName, judgeId });
            sendMessage({ action: "resetTimer", meetName, judgeId });
            sendMessage({ action: "startTimer", meetName, judgeId });
        });
    }
});

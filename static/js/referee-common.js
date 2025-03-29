// static/js/referee-common.js
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

    // also send logs to server
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', error));
}

// We assume each referee page sets 'judgeId' in a <script> above this file:
//   <script> let judgeId = "center"; </script>
// Then loads this JS.

document.addEventListener('DOMContentLoaded', function() {

    // helper to get meetName from <div id="meetName" data-meet-name="...">
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
    const redButton = document.getElementById('redButton');
    const platformReadyButton = document.getElementById('platformReadyButton');

    // If you want a visible timer on the referee page:
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');

    // Possibly track occupant data:
    const leftUserEl   = document.getElementById("leftUser");
    const centerUserEl = document.getElementById("centerUser");
    const rightUserEl  = document.getElementById("rightUser");

    // onopen
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

            // 1) Occupancy changes
            case "occupancyChanged":
                log(`occupancyChanged: L=${data.leftUser} C=${data.centerUser} R=${data.rightUser}`, "debug");
                if (leftUserEl)   leftUserEl.innerText   = data.leftUser   || "Vacant";
                if (centerUserEl) centerUserEl.innerText = data.centerUser || "Vacant";
                if (rightUserEl)  rightUserEl.innerText  = data.rightUser  || "Vacant";
                break;

            // 2) Health checks
            case "refereeHealth": {
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;
            }
            case "healthError":
                log(`Health error: ${data.message}`, "debug");
                break;

            // 3) Timers + decisions
            case "startTimer":
                log("🔵 Received startTimer in referee-common.js; clearing results, show timer if needed", "debug");
                // If you want to show a timer on the referee page:
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
                // Possibly reset any local state
                break;

            case "updatePlatformReadyTime":
                log(`⌛ updatePlatformReadyTime: ${data.timeLeft} sec left`, "debug");
                if (data.timeLeft <= 0) {
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.add("hidden");
                    }
                } else {
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.remove("hidden");
                    }
                    if (timerDisplay) {
                        timerDisplay.textContent = data.timeLeft + "s";
                    }
                }
                break;

            case "updateNextAttemptTime":
                log(`RefereeCommon: ignoring updateNextAttemptTime (judgeId=${judgeId}). Add UI logic if needed.`, "debug");
                // You can add logic here if referees should also see next attempt timers
                break;

            case "judgeSubmitted":
                log(`RefereeCommon: Another judge submitted a decision: judgeId=${data.judgeId}`, "debug");
                // If you want to show a UI indicator that left/center/right has submitted, handle it here
                break;

            case "displayResults":
                log(`RefereeCommon: final decisions => L=${data.leftDecision}, C=${data.centerDecision}, R=${data.rightDecision}`, "debug");
                // If the referee page wants to show final results, handle them:
                // e.g. document.getElementById("someEl").innerText = `L=${data.leftDecision},C=${data.centerDecision},R=${data.rightDecision}`;
                break;

            case "clearResults":
                log("RefereeCommon: clearing results UI. (If referee page shows lights or timer, reset them)", "debug");
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.add("hidden");
                }
                if (timerDisplay) {
                    timerDisplay.textContent = "";
                }
                // Possibly revert local decision indicators
                break;

            case "platformReadyExpired":
                log("RefereeCommon: Platform Ready Timer Expired", "debug");
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.add("hidden");
                }
                break;

            case "resetLights":
                log("RefereeCommon: resetLights action (usually relevant to the /lights page). Doing nothing here.", "debug");
                break;

            default:
                log(`⚠️ Unknown action: ${data.action}`, "debug");
        }
    };

    // handle errors
    socket.onerror = function(error) {
        log(`WebSocket error (${judgeId}): ${error}`, "error");
    };

    // handle close (ReconnectingWebSocket will attempt reconnect automatically)
    socket.onclose = function(event) {
        log(`WebSocket closed (${judgeId}): ${event.code} - ${event.reason}`, "info");
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

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

    // If your "Platform Ready" button is on the center referee page:
    if (judgeId === "center" && platformReadyButton) {
        platformReadyButton.addEventListener("click", () => {
            log("'Platform Ready' button clicked; sending startTimer", "debug");
            sendMessage({ action: "startTimer", meetName: meetName });
        });
    }

    // handle White/Red button clicks
    const whiteBtn = document.getElementById('whiteButton');
    if (whiteBtn) {
        whiteBtn.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "white"
            });
            log(`[RefereeCommon] Judge '${judgeId}' clicked GOOD LIFT (white).`, "info");
        });
    }

    const redBtn = document.getElementById('redButton');
    if (redBtn) {
        redBtn.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetName,
                judgeId: judgeId,
                decision: "red"
            });
            log(`[RefereeCommon] Judge '${judgeId}' clicked NO LIFT (red).`, "info");
        });
    }
});

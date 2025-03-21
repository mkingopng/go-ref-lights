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

// We assume that each referee page sets 'judgeId' in <script> above this file:
//   <script> let judgeId = "center"; </script>
// Then loads this JS.

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
    const healthEl      = document.getElementById("healthStatus");
    const whiteButton   = document.getElementById('whiteButton');
    const redButton     = document.getElementById('redButton');
    const startTimerBtn = document.getElementById('startTimerButton');
    const platformReadyButton = document.getElementById('platformReadyButton');

    // If you want a visible timer in referee page:
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');

    // This might be for occupant display:
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

            // existing occupant / seat info
            case "occupancyChanged":
                log(`occupancyChanged: L=${data.leftUser} C=${data.centerUser} R=${data.rightUser}`, "debug");
                if (leftUserEl)   leftUserEl.innerText   = data.leftUser   || "Vacant";
                if (centerUserEl) centerUserEl.innerText = data.centerUser || "Vacant";
                if (rightUserEl)  rightUserEl.innerText  = data.rightUser  || "Vacant";
                break;

            case "refereeHealth": {
                // If data.connectedRefIDs includes me, I'm connected
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;
            }

            case "healthError":
                alert(data.message);
                break;

            // ------------------------------
            // *** The ones previously "Unhandled" ***
            // ------------------------------

            case "startTimer":
                log("🔵 Received startTimer in referee-common.js; clearing results, show timer if needed", "debug");
                // If you want to show a timer on the referee page:
                if (platformReadyTimerContainer) platformReadyTimerContainer.classList.remove("hidden");
                // Possibly reset some local state
                break;

            case "updatePlatformReadyTime":
                // The server is sending timeLeft
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

            case "clearResults":
                log("RefereeCommon: clearing results UI. (If referee page shows lights, do it here)", "debug");
                // In your referee page, maybe you don't do anything; or you could revert local state.

                // example:
                // let circleEls = document.querySelectorAll(".circle");
                // circleEls.forEach(el => el.style.backgroundColor = "black");
                break;

            case "judgeSubmitted":
                log(`RefereeCommon: Another judge submitted a decision: judgeId=${data.judgeId}`, "debug");
                // If you want to show a UI indicator that left/center/right has submitted, handle it here
                break;

            case "displayResults":
                log(`RefereeCommon: final decisions => L=${data.leftDecision}, C=${data.centerDecision}, R=${data.rightDecision}`, "debug");
                // If the referee page wants to see the final results, do so:
                // document.getElementById("someEl").innerText = data.leftDecision + data.centerDecision + data.rightDecision;
                break;

            case "platformReadyExpired":
                log("RefereeCommon: Platform Ready Timer Expired", "debug");
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.add("hidden");
                }
                break;

            case "resetLights":
                log("RefereeCommon: resetLights action (usually for lights page). Doing nothing here.", "debug");
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

// static/js/lights.js
"use strict";

let socket;
let platformReadyInterval = null;
let resultsDisplayed = false; // flag to indicate that displayResults has been processed

const multiNextAttemptTimers = document.getElementById("multiNextAttemptTimers");
let nextAttemptTimers = {};       // key = timer.ID, value = some DOM or state
let platformTimerActive = false;  // track if a platform-ready timer is active

// utility function for logging
function log(message, level = 'debug') {
    const timestamp = new Date().toISOString();
    const logMessage = `[${timestamp}] ${level.toUpperCase()}: ${JSON.stringify(message)}`;

    // log to console
    switch (level) {
        case 'error':
            console.error(JSON.stringify(logMessage));
            break;
        case 'warn':
            console.warn(JSON.stringify(logMessage));
            break;
        case 'debug':
            console.debug(JSON.stringify(logMessage));
            break;
        default:
            console.log(JSON.stringify(logMessage));
    }

    // send logs to a server for saving to a file
    fetch('/log', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: logMessage, level: level }),
    }).catch(error => console.error('Failed to send log to server:', JSON.stringify(error)));
}

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
            meetName = sessionStorage.getItem("meetName")
                || new URLSearchParams(window.location.search).get("meetName");
        }
        if (meetName) {
            sessionStorage.setItem("meetName", meetName);
            log(`✅ Meet name set: ${meetName}`, "info");
        } else {
            log("⚠️ Meet name is missing! Redirecting to meet selection.", "warn");
            window.location.href = "/meets";
        }
        return meetName;
    }

    // helper function to update the platform ready timer UI
    function updatePlatformReadyTimer(timer) {
        const timeLeft = timer.TimeLeft ?? 0;
        const container = document.getElementById("platformReadyTimerContainer");
        const timerSpan = document.getElementById("timer");
        if (!container || !timerSpan) return;
        if (timeLeft > 0 && timer.Active) {
            container.classList.remove("hidden");
            platformTimerActive = true;
            timerSpan.textContent = `${timeLeft}s`;
        } else {
            container.classList.add("hidden");
            platformTimerActive = false;
        }
    }

    // helper function to update a next attempt timer UI element
    function updateNextAttemptTimer(timer) {
        const existing = nextAttemptTimers[timer.ID];
        const timeLeft = timer.TimeLeft ?? 0;

        if (timeLeft <= 0 || !timer.Active) {
            removeNextAttemptTimer(timer.ID);
            return;
        }

        if (!existing) {
            const div = document.createElement("div");
            div.id = `nextAttemptTimer_${timer.ID}`;
            div.classList.add("single-attempt-timer");
            div.textContent = `Next Attempt: ${timeLeft}s`;
            multiNextAttemptTimers.classList.remove("hidden");
            multiNextAttemptTimers.appendChild(div);
            nextAttemptTimers[timer.ID] = div;
        } else {
            existing.textContent = `Next Attempt: ${timeLeft}s`;
        }
    }

    // main handler for the "updateNextAttemptTime" action
    function handleUpdateNextAttemptTime(data) {
        if (!data.timers || !Array.isArray(data.timers)) return;
        data.timers.forEach((timer) => {
            if (timer.type === "platformReady") {
                updatePlatformReadyTimer(timer);
            } else if (timer.type === "nextAttempt") {
                updateNextAttemptTimer(timer);
            } else {
                console.warn("Unknown timer type:", timer.type, "for timer ID", timer.ID);
            }
        });
    }

    function removeNextAttemptTimer(timerId) {
        const div = nextAttemptTimers[timerId];
        if (div && div.parentNode) {
            div.parentNode.removeChild(div);
        }
        delete nextAttemptTimers[timerId];

        if (Object.keys(nextAttemptTimers).length === 0) {
            multiNextAttemptTimers.classList.add("hidden");
        }
    }

    // -------------------------------------------------------------
    // The main function to parse the entire final result object
    // and display it. (Already a placeholder, but reference if you
    // want separate UI updates.)
    // -------------------------------------------------------------
    function displayResults(msg) {
        console.log("Final decisions => Left:", msg.leftDecision,
            "Center:", msg.centerDecision,
            "Right:", msg.rightDecision);

        document.getElementById("leftLight").className   = (msg.leftDecision === "white") ? "greenLight" : "redLight";
        document.getElementById("centerLight").className = (msg.centerDecision === "white") ? "greenLight" : "redLight";
        document.getElementById("rightLight").className  = (msg.rightDecision === "white") ? "greenLight" : "redLight";
    }

    function updatePlatformTimer(secondsLeft) {
        const timerEl = document.getElementById("platformTimer");
        if (!timerEl) return;
        timerEl.textContent = `Platform Timer: ${secondsLeft} seconds remaining`;
    }

    function updateNextAttemptTimers(timers) {
        console.log("Next Attempt Timers =>", timers);
    }

    // constants
    const meetName = getMeetName();
    if (!meetName) return;
    const judgeId = "lights";

    // build your WebSocket URL
    const scheme = (window.location.protocol === "https:") ? "wss" : "ws";
    const wsUrl = `${scheme}://${window.location.host}/referee-updates?meetName=${meetName}`;

    // ReconnectingWebSocket settings
    socket = new ReconnectingWebSocket(wsUrl, null, {
        reconnectInterval: 2000,   // 2 seconds
        maxReconnectAttempts: null // infinite
    });

    // grab common DOM elements
    const timerDisplay = document.getElementById('timer');
    const healthEl = document.getElementById("healthStatus");
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const statusEl = document.getElementById("connectionStatus");
    const messageEl = document.getElementById("message");

    // WebSocket lifecycle events
    socket.onopen = function () {
        log("✅ WebSocket connection established (Lights).", "info");
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
        if (statusEl) {
            statusEl.innerText = "Disconnected";
            statusEl.style.color = "red";
        }
    };

    socket.onerror = function (error) {
        log(`⚠️ WebSocket error: ${error}`, "error");
    };

    // main websocket message handler

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

            case "refereeHealth": {
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

            case "startTimer":
                log("🔵 Received startTimer from server, starting Platform Ready Timer countdown");
                resultsDisplayed = false;

                if (platformReadyInterval) {
                    clearInterval(platformReadyInterval);
                    platformReadyInterval = null;
                }

                // clear any leftover nextAttemptTimers
                Object.keys(nextAttemptTimers).forEach(id => {
                    if (multiNextAttemptTimers && nextAttemptTimers[id]) {
                        multiNextAttemptTimers.removeChild(nextAttemptTimers[id]);
                    }
                    delete nextAttemptTimers[id];
                });
                if (multiNextAttemptTimers) {
                    multiNextAttemptTimers.classList.add("hidden");
                }
                if (platformReadyTimerContainer) {
                    platformReadyTimerContainer.classList.remove("hidden");
                }
                if (timerDisplay) {
                    // data.timeLeft might be included
                    timerDisplay.innerText = (data.timeLeft !== undefined)
                        ? `${data.timeLeft}s`
                        : "60s";
                }
                break;

            case "updatePlatformReadyTime":
                log(`⌛ Handling updatePlatformReadyTime: ${data.timeLeft}s left`, "debug");
                if (data.timeLeft <= 0) {
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.add("hidden");
                    }
                } else {
                    if (platformReadyTimerContainer) {
                        platformReadyTimerContainer.classList.remove("hidden");
                    }
                    if (timerDisplay) {
                        timerDisplay.innerText = `${data.timeLeft}s`;
                    }
                }
                break;

            case "updateNextAttemptTime":
                log("✅ Entering handleUpdateNextAttemptTime", "debug");
                handleUpdateNextAttemptTime(data);
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

                let whiteCount = 0;
                let redCount = 0;
                [data.leftDecision, data.centerDecision, data.rightDecision].forEach(dec => {
                    if (dec === "white") { whiteCount++; } else { redCount++; }
                });

                if (whiteCount >= 2) {
                    messageEl.innerText = "Good Lift";
                    messageEl.style.color = "green";
                } else {
                    messageEl.innerText = "No Lift";
                    messageEl.style.color = "red";
                }
                messageEl.classList.add("flash");

                // clear the message text in 15 seconds
                setTimeout(() => {
                    messageEl.innerText = "";
                    messageEl.classList.remove("flash");
                }, 15000);

                resultsDisplayed = true;
                break;

            case "clearResults":
                log("Clearing results from Lights UI (white vs red circles, judge indicators).");
                leftCircle.style.backgroundColor   = "black";
                centerCircle.style.backgroundColor = "black";
                rightCircle.style.backgroundColor  = "black";
                leftIndicator.style.backgroundColor   = "grey";
                centerIndicator.style.backgroundColor = "grey";
                rightIndicator.style.backgroundColor  = "grey";

                if (messageEl) {
                    messageEl.innerText = "";
                    messageEl.classList.remove("flash");
                }
                resultsDisplayed = false;
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

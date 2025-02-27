// static/js/lights.js
"use strict";

let socket;

window.addEventListener("DOMContentLoaded", function () {
    let meetNameElement = document.getElementById("meetName");
    let meetName = meetNameElement ? meetNameElement.dataset.meetName : "";

    if (!meetName) {
        console.error("⚠️ Meet name not found. WebSocket will not be initialized.");
        return;
    }

    let websocketUrl = `ws://localhost:8080/referee-updates?meetName=${meetName}`;
    socket = new WebSocket(websocketUrl);

    socket.onopen = function () {
        console.log("✅ WebSocket connection established (Lights).");
    };

    socket.onclose = function (event) {
        console.warn("⚠️ WebSocket connection closed (Lights):", event.code, event.reason);
    };

    socket.onerror = function (error) {
        console.error("⚠️ WebSocket error:", error);
    };

    socket.onmessage = function (event) {
        console.log("📩 Raw WebSocket message received (Lights):", event.data);
    };
});

document.addEventListener('DOMContentLoaded', () => {
    log("DOM fully loaded and parsed");
    console.log("Lights.js loaded.");

    // retrieve meetName from multiple sources
    function getmeetName() {
        const meetNameElement = document.getElementById("meetName");
        let meetName = meetNameElement ? meetNameElement.dataset.meetName : null;

        if (!meetName) {
            meetName = sessionStorage.getItem("meetName") || new URLSearchParams(window.location.search).get("meetName");
        }

        if (meetName) {
            sessionStorage.setItem("meetName", meetName);
            console.log("✅ meetName set:", meetName);
        } else {
            console.warn("⚠️ meetName is missing! WebSocket may not function correctly.");
            alert("Error: No meet selected. Redirecting to meet selection.");
            window.location.href = "/meets"; // Redirect user to meet selection
        }
        return meetName;
    }

    const meetName = getmeetName();

    const wsUrl = `ws://${window.location.host}/referee-updates?meetName=${encodeURIComponent(meetName)}`;
    socket = new WebSocket(wsUrl);

    console.log("Lights page loaded");
    console.log("meetName:", meetName);
    console.log("WebSocket URL:", wsUrl);

    let connectedReferees = 0; // define globally
    let requiredReferees = 3; // add a default value or get from server

    if (!meetName) {
        log("⚠️ meetName is missing! WebSocket will not work properly.", "error");
        return;
    }

    // cache references to DOM elements
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');
    const messageElement = document.getElementById('message');
    const connectionStatusElement = document.getElementById('connectionStatus');

    // Platform Ready Button Logic
    if (platformReadyButton) {
        platformReadyButton.addEventListener("click", () => {
            if (socket.readyState === WebSocket.OPEN) {
                log(`🟢 Platform Ready button clicked, sending startTimer action`);
                socket.send(JSON.stringify({action: "startTimer", meetName: meetName}));
                platformReadyTimerContainer.classList.remove("hidden");
            } else {
                log("❌ WebSocket is not ready. Cannot send startTimer action.", "error");
            }
        });
    }

    // store judge decisions in an object
    const judgeDecisions = {
        left: null,
        centre: null,
        right: null
    };

    // store a reference to container for all next-attempt timers:
    const multiTimerContainer = document.getElementById('multiNextAttemptTimers');

    // keep track of DOM elements for each timer row in a dictionary:
    const nextAttemptRows = {};

    function updatePlatformReadyTimerOnUI(timeLeft) {
        log(`Updating platform ready timer UI: ${timeLeft}s`);
        if (platformReadyTimerContainer) {
            platformReadyTimerContainer.classList.remove('hidden');
        }
        if (timerDisplay) {
            timerDisplay.innerText = `${timeLeft}s`;
        }
    }

    // attach websocket event listeners
    socket.onopen = () => {
        log("WebSocket connection established (Lights).");
    };
    socket.onerror = (error) => {
        log(`WebSocket error (Lights): ${error}`, "error");
    };
    socket.onclose = (event) => {
        log(`WebSocket connection closed (Lights): ${event.code} - ${event.reason}`);
    };

    //utility function for logging
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
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({message: logMessage, level: level}),
        }).catch(error => {
            console.error('Failed to send log to server:', error);
        });
    }

    //  timer UI Handling (Server-Driven)
    function handlePlatformReadyExpired() {
        log("Platform Ready timer expired");
        if (timerDisplay) timerDisplay.innerText = '0s';
        // todo: delay, clear
    }

    //  referee decision handling
    function showJudgeSubmissionIndicator(judgeId) {
        log(`Showing judge submission indicator for judgeId "${judgeId}"`);
        const indicator = document.getElementById(`${judgeId}Indicator`);
        if (!indicator) {
            log(`Indicator for judgeId "${judgeId}" not found`, 'error');
            return;
        }
        indicator.style.backgroundColor = "green";
    }

    function displayResults(data) {
        log(`Displaying results: ${JSON.stringify(data)}`);
        const {leftDecision, centreDecision, rightDecision} = data;
        paintCircle('leftCircle', leftDecision);
        paintCircle('centreCircle', centreDecision);
        paintCircle('rightCircle', rightDecision);
        judgeDecisions.left = leftDecision;
        judgeDecisions.centre = centreDecision;
        judgeDecisions.right = rightDecision;
        const decisions = [leftDecision, centreDecision, rightDecision];
        const whiteCount = decisions.filter(d => d === "white").length;
        const redCount = decisions.filter(d => d === "red").length;
        if (whiteCount >= 2) {
            displayMessage("Good Lift", "white");
        } else if (redCount >= 2) {
            displayMessage("No Lift", "red");
        }
    }

    function clearResultsUI() {
        log("Clearing results UI");
        paintCircle('leftCircle', null);
        paintCircle('centreCircle', null);
        paintCircle('rightCircle', null);
        displayMessage('', '');
        resetJudgeIndicators();
    }

    //  UI helper functions
    function paintCircle(circleId, decision) {
        log(`Painting circle ${circleId} with decision: ${decision}`);
        const circle = document.getElementById(circleId);
        if (!circle) return;
        circle.style.backgroundColor = decision === "white" ? "white" : decision === "red" ? "red" : "black";
    }

    //  health check UI
    function updateHealthStatus(connected, required) {
        log(`Updating health status: ${connected}/${required} connected`);
        connectedReferees = connected;
        requiredReferees = required;

        if (connectionStatusElement) {
            connectionStatusElement.innerText = `Referees Connected: ${connected}/${required}`;
            connectionStatusElement.style.color = (connected < required) ? "red" : "green";
        }
        if (platformReadyButton) {
            log(`Platform not ready: ${connected}/${required} connected`);
        }
    }

    // reset all judge indicators to grey
    function resetJudgeIndicators() {
        log("Resetting judge indicators");
        const indicators = document.querySelectorAll('.indicator');
        indicators.forEach(indicator => {
            indicator.style.backgroundColor = 'grey';
        });
    }

    // display a message on the screen
    function displayMessage(text, color) {
        log(`Displaying message: ${text} with colour ${color}`);
        if (!messageElement) return;
        messageElement.innerText = text;
        messageElement.style.color = color;
    }

    //  Next Attempt Timer Handling
    function handleNextAttemptExpired(timerIndex) {
        log(`handleNextAttemptExpired called for timer #${timerIndex + 1}`);
        if (nextAttemptRows[timerIndex]) {
            log(`Found timer #${timerIndex + 1} in nextAttemptRows. Removing now.`);
            const {rowDiv} = nextAttemptRows[timerIndex];
            rowDiv.style.transition = "opacity 0.5s ease-out";
            rowDiv.style.opacity = "0";
            setTimeout(() => {
                if (rowDiv.parentNode) {
                    rowDiv.parentNode.removeChild(rowDiv);
                }
                delete nextAttemptRows[timerIndex];
            }, 500);
        } else {
            log(`Timer #${timerIndex + 1} not found in nextAttemptRows!`, 'warn');
        }
    }

    // called when receive "updateNextAttemptTime" msg from the server // fix_me
    function updateNextAttemptTimerOnUI(timeLeft, timerIndex) {
        log(`update next attempt timer UI: ${timeLeft}s for index ${timerIndex}`);
        if (!multiTimerContainer) return;

        // If we don't yet have a row for this index, create one
        if (!nextAttemptRows[timerIndex]) {
            log(`Creating new timer row for index ${timerIndex}`);

            // Create a new <div> for the row
            const rowDiv = document.createElement('div');
            rowDiv.classList.add('timer-container');
            rowDiv.style.marginBottom = '10px';

            // Create a label for the timer row
            const label = document.createElement('div');
            // Displaying index + 1 if you want to show human-friendly numbering
            label.innerText = `Next Attempt #${timerIndex + 1}:`;
            label.classList.add('timer');
            rowDiv.appendChild(label);

            // Create a time display element
            const timeSpan = document.createElement('div');
            timeSpan.classList.add('second-timer');
            timeSpan.innerText = `${timeLeft}s`;
            rowDiv.appendChild(timeSpan);

            multiTimerContainer.appendChild(rowDiv);
            nextAttemptRows[timerIndex] = {rowDiv, label, timeSpan};
        } else {
            log(`Updating existing timer row for index ${timerIndex}`);
            nextAttemptRows[timerIndex].timeSpan.innerText = `${timeLeft}s`;
        }

        // If timeLeft is 0 or less, fade out and remove the timer row
        if (timeLeft <= 0) {
            const {rowDiv} = nextAttemptRows[timerIndex];
            rowDiv.style.transition = "opacity 0.5s ease-out";
            rowDiv.style.opacity = "0";
            setTimeout(() => {
                if (rowDiv.parentNode) {
                    rowDiv.parentNode.removeChild(rowDiv);
                }
                delete nextAttemptRows[timerIndex];
            }, 500);
        }
    }
});

// listen for messages from the server
socket.onmessage = (event) => {
    console.log("📩 WebSocket Message:", data);
    let data;

    try {
        data = JSON.parse(event.data);
        log(`Received Websocket message: ${JSON.stringify(data)}`, 'debug');
    } catch (e) {
        log(`Invalid JSON from server:, ${event.data}`, 'error');
        return;
    }

    // process websocket messages
    switch (data.action) {
        case "startTimer":
            log("🔵 Received startTimer event from WebSocket");
            break;
        case "updatePlatformReadyTime":
            log(`Updating platform ready timer: ${data.timeLeft}s`);
            updatePlatformReadyTimerOnUI(data.timeLeft);
            break;
        case "platformReadyExpired":
            log(`Platform ready timer expired`);
            handlePlatformReadyExpired();
            break;
        case "updateNextAttemptTime":
            log(`Updating next attempt timer: ${data.timeLeft}s for index: ${data.index}`);
            updateNextAttemptTimerOnUI(data.timeLeft, data.index);
            break;
        case "nextAttemptExpired":
            log(`Received nextAttemptExpired event for index: ${data.index}`);
            handleNextAttemptExpired(data.index);
            break;

        // judge decision Handling
        case "judgeSubmitted":
            log(`Judge ${data.judgeId} submitted a decision`);
            showJudgeSubmissionIndicator(data.judgeId);
            break;
        case "displayResults":
            log(`Displaying results: ${JSON.stringify(data)}`);
            displayResults(data);
            break;
        case "clearResults":
            clearResultsUI();
            break;

        // health check
        case "refereeHealth":
            log(`✅ Calling updateHealthStatus: ${data.connectedReferees}/${data.requiredReferees}`);
            updateHealthStatus(data.connectedReferees, data.requiredReferees);
            break;

        case "healthError":
            // if user tried to start timer but not all refs connected
            log(`Health Error: ${data.message}`, "error");
            displayMessage(data.message, "red");
            break;

        default:
            log(`Unknown action: ${data.action}`, "warn");
    }
};

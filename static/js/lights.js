// static/js/lights.js

document.addEventListener('DOMContentLoaded', () => {
    // Cache references to DOM elements
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');
    const secondTimerDisplay = document.getElementById('secondTimer');
    const messageElement = document.getElementById('message');
    const connectionStatusElement = document.getElementById('connectionStatus');

    // We'll store judge decisions in an object
    const judgeDecisions = {
        left: null,
        centre: null,
        right: null
    };

    // Track how many refs are connected (0..3)
    let connectedReferees = 0;

    // Check for the global websocketUrl
    if (typeof websocketUrl === 'undefined') {
        console.error("websocketUrl is not defined");
        return;
    }

    // Initialize the WebSocket connection
    const socket = new WebSocket(websocketUrl);

    socket.onopen = () => {
        console.log("WebSocket connection established (Lights).");
    };

    socket.onerror = (error) => {
        console.error("WebSocket error (Lights):", error);
    };

    socket.onclose = (event) => {
        console.log("WebSocket connection closed (Lights):", event);
    };

    // Listen for messages from the server
    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            console.error("Invalid JSON from server:", event.data);
            return;
        }

        // Check the action
        switch (data.action) {
            // Server-driven timer updates
            case "updatePlatformReadyTime":
                updatePlatformReadyTimerOnUI(data.timeLeft);
                break;
            case "platformReadyExpired":
                handlePlatformReadyExpired();
                break;
            case "updateNextAttemptTime":
                updateNextAttemptTimerOnUI(data.timeLeft, data.index);
                break;
            case "nextAttemptExpired":
                console.log(`Received nextAttemptExpired event for index: ${data.index}`);
                handleNextAttemptExpired(data.index);
                break;

            // Judge/Decision Handling
            case "judgeSubmitted":
                showJudgeSubmissionIndicator(data.judgeId);
                break;
            case "displayResults":
                displayResults(data);
                break;
            case "clearResults":
                clearResultsUI();
                break;

            // Health Check
            case "refereeHealth":
                updateHealthStatus(data.connectedReferees, data.requiredReferees);
                break;

            case "healthError":
                // If user tried to start timer but not all refs connected
                displayMessage(data.message, "red");
                break;

            default:
                console.warn("Unknown action:", data.action);
        }
    };

    //  Timer UI Handling (Server-Driven)
    function updatePlatformReadyTimerOnUI(timeLeft) {
        if (platformReadyTimerContainer) {
            platformReadyTimerContainer.classList.remove('hidden');
        }
        if (timerDisplay) {
            timerDisplay.innerText = `${timeLeft}s`;
        }
    }

    function handlePlatformReadyExpired() {
        if (timerDisplay) timerDisplay.innerText = '0s';
        // displayMessage('Time Up', 'yellow');
    }

    // We'll store a reference to our container for all next-attempt timers:
    const multiTimerContainer = document.getElementById('multiNextAttemptTimers');

// Keep track of DOM elements for each timer row in a dictionary:
    const nextAttemptRows = {};

// Called when we see "updateNextAttemptTime" from the server
    function updateNextAttemptTimerOnUI(timeLeft, timerIndex) {
        // Make sure we have a container for all timers
        if (!multiTimerContainer) return;

        // If we don't yet have a row for this index, create one
        if (!nextAttemptRows[timerIndex]) {
            // Create a new <div> for the row
            const rowDiv = document.createElement('div');
            rowDiv.classList.add('timer-container'); // same styling
            rowDiv.style.marginBottom = '10px';

            // Create a label
            const label = document.createElement('div');
            label.innerText = `Next Attempt #${timerIndex + 1}:`;
            label.classList.add('timer');  // so it picks up big font if you like
            rowDiv.appendChild(label);

            // Create a time display
            const timeSpan = document.createElement('div');
            timeSpan.classList.add('second-timer');
            timeSpan.innerText = `${timeLeft}s`;
            rowDiv.appendChild(timeSpan);

            multiTimerContainer.appendChild(rowDiv);
            // Store references
            nextAttemptRows[timerIndex] = { rowDiv, label, timeSpan };
        } else {
            // Update existing
            nextAttemptRows[timerIndex].timeSpan.innerText = `${timeLeft}s`;
        }
    }

    function handleNextAttemptExpired(timerIndex) {
        console.log(`handleNextAttemptExpired called for timer #${timerIndex + 1}`);
        if (nextAttemptRows[timerIndex]) {
            console.log(`Found timer #${timerIndex + 1} in nextAttemptRows. Removing now.`);
            nextAttemptRows[timerIndex].rowDiv.remove();  // Remove from DOM
            delete nextAttemptRows[timerIndex];           // Remove from memory
        } else {
            console.warn(`Timer #${timerIndex + 1} not found in nextAttemptRows!`);
        }
    }


    //  Decision Handling
    function showJudgeSubmissionIndicator(judgeId) {
        const indicator = document.getElementById(`${judgeId}Indicator`);
        if (!indicator) {
            console.error(`Indicator for judgeId "${judgeId}" not found`);
            return;
        }
        indicator.style.backgroundColor = "green";
    }

    function displayResults(data) {
        const { leftDecision, centreDecision, rightDecision } = data;

        paintCircle('leftCircle', leftDecision);
        paintCircle('centreCircle', centreDecision);
        paintCircle('rightCircle', rightDecision);

        judgeDecisions.left   = leftDecision;
        judgeDecisions.centre = centreDecision;
        judgeDecisions.right  = rightDecision;

        const decisions = [leftDecision, centreDecision, rightDecision];
        const whiteCount = decisions.filter(d => d === "white").length;
        const redCount   = decisions.filter(d => d === "red").length;

        if (whiteCount >= 2) {
            displayMessage("Good Lift", "white");
        } else if (redCount >= 2) {
            displayMessage("No Lift", "red");
        }
    }

    function clearResultsUI() {
        paintCircle('leftCircle', null);
        paintCircle('centreCircle', null);
        paintCircle('rightCircle', null);
        displayMessage('', '');
        resetJudgeIndicators();
    }

    //  UI Helper Functions
    function paintCircle(circleId, decision) {
        const circle = document.getElementById(circleId);
        if (!circle) return;
        switch (decision) {
            case "white":
                circle.style.backgroundColor = "white";
                break;
            case "red":
                circle.style.backgroundColor = "red";
                break;
            default:
                circle.style.backgroundColor = "black";
                break;
        }
    }

    function resetJudgeIndicators() {
        const indicators = document.querySelectorAll('.indicator');
        indicators.forEach(indicator => {
            indicator.style.backgroundColor = 'grey';
        });
    }

    function displayMessage(text, color) {
        if (!messageElement) return;
        messageElement.innerText = text;
        messageElement.style.color = color;

        // If "Time Up" => flash
        if (text.includes("Time Up")) {
            messageElement.classList.add('flash');
        } else {
            messageElement.classList.remove('flash');
        }
    }

    //  Health Check UI
    function updateHealthStatus(connected, required) {
        connectedReferees = connected;

        if (connectionStatusElement) {
            connectionStatusElement.innerText = `Referees Connected: ${connected}/${required}`;
            connectionStatusElement.style.color = (connected < required) ? "red" : "green";
        }

        // Optionally disable the "Platform Ready" button if not all refs
        if (platformReadyButton) {
            platformReadyButton.disabled = (connected < required);
        }
    }

    //  Platform Ready Button Logic
    if (platformReadyButton && platformReadyTimerContainer) {
        platformReadyButton.addEventListener('click', () => {
            // Current code toggles local container visibility:
            const isHidden = platformReadyTimerContainer.classList.contains('hidden');
            if (isHidden) {
                // Request server to "startTimer" (server will check if all refs connected)
                socket.send(JSON.stringify({ action: "startTimer" }));
            } else {
                // If it's visible, we request to stop
                socket.send(JSON.stringify({ action: "stopTimer" }));
            }
            // toggle local display
            platformReadyTimerContainer.classList.toggle('hidden');
        });
    } else {
        console.warn("Platform Ready button or container not found.");
    }

});

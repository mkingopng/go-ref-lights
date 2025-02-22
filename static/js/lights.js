// static/js/lights.js

document.addEventListener('DOMContentLoaded', () => {
    // Cache references to DOM elements
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');
    const nextAttemptTimerContainer = document.getElementById('nextAttemptTimerContainer');
    const secondTimerDisplay = document.getElementById('secondTimer');
    const messageElement = document.getElementById('message');

    // We'll store judge decisions in an object
    // If your code needs it, you can keep it.
    // Otherwise, the server's the source of truth now.
    const judgeDecisions = {
        left: null,
        centre: null,
        right: null
    };

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
                updateNextAttemptTimerOnUI(data.timeLeft);
                break;
            case "nextAttemptExpired":
                handleNextAttemptExpired();
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

            default:
                console.warn("Unknown action:", data.action);
        }
    };

    // ------------------------------------------------
    //  Timer UI Handling (Server-Driven)
    // ------------------------------------------------
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
        displayMessage('Time Up', 'yellow');
    }

    function updateNextAttemptTimerOnUI(timeLeft) {
        if (nextAttemptTimerContainer) {
            nextAttemptTimerContainer.classList.remove('hidden');
        }
        if (secondTimerDisplay) {
            secondTimerDisplay.innerText = `${timeLeft}s`;
        }
    }

    function handleNextAttemptExpired() {
        if (secondTimerDisplay) secondTimerDisplay.innerText = '0s';
        displayMessage('Next Attempt Time Up', 'yellow');
    }

    // ------------------------------------------------
    //  Decision Handling
    // ------------------------------------------------
    function showJudgeSubmissionIndicator(judgeId) {
        const indicator = document.getElementById(`${judgeId}Indicator`);
        if (!indicator) {
            console.error(`Indicator for judgeId "${judgeId}" not found`);
            return;
        }
        indicator.style.backgroundColor = "green";
    }

    function displayResults(data) {
        // Destructure the needed properties
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

    // ------------------------------------------------
    //  UI Helper Functions
    // ------------------------------------------------
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

    // ------------------------------------------------
    //  Platform Ready Button Logic
    // ------------------------------------------------
    if (platformReadyButton && platformReadyTimerContainer) {
        platformReadyButton.addEventListener('click', () => {
            // If container is hidden, user wants to "startTimer"
            const isHidden = platformReadyTimerContainer.classList.contains('hidden');
            if (isHidden) {
                // Send to server
                socket.send(JSON.stringify({ action: "startTimer" }));
            } else {
                // Container is visible => user wants to stop
                socket.send(JSON.stringify({ action: "stopTimer" }));
            }
            // Toggle local display
            platformReadyTimerContainer.classList.toggle('hidden');
        });
    } else {
        console.warn("Platform Ready button or container not found.");
    }

});

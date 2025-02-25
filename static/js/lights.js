// static/js/lights.js
document.addEventListener('DOMContentLoaded', () => {
    // cache references to DOM elements
    const platformReadyButton = document.getElementById('platformReadyButton');
    const platformReadyTimerContainer = document.getElementById('platformReadyTimerContainer');
    const timerDisplay = document.getElementById('timer');
    const messageElement = document.getElementById('message');
    const connectionStatusElement = document.getElementById('connectionStatus');

    // store judge decisions in an object
    const judgeDecisions = {
        left: null,
        centre: null,
        right: null
    };

    // track how many refs are connected (0..3)
    let connectedReferees = 0;

    // check for the global websocketUrl
    if (typeof websocketUrl === 'undefined') {
        console.error("❌ websocketUrl is not defined. WebSocket connection cannot be established.");
        return;
    }

    // initialize the WebSocket connection
    const socket = new WebSocket(websocketUrl);

    socket.onopen = () => {
        console.log(`✅ WebSocket connection established (Lights) at: ${websocketUrl}`);
    };
    socket.onerror = (error) => {
        console.error(`⚠️ WebSocket error (Lights):`, error);
    };
    socket.onclose = (event) => {
        console.warn(`🔴 WebSocket closed (Lights). Code: ${event.code}, Reason: ${event.reason || "Unknown"}`);
    };

    // listen for messages from the server
    socket.onmessage = (event) => {
        console.log(`📩 Received WebSocket message at ${new Date().toISOString()}:`, event.data);
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            console.error(`❌ Invalid JSON from server:`, event.data);
            return;
        }

        // check the action
        switch (data.action) {
            // server-driven timer updates
            case "updatePlatformReadyTime":
                console.log(`⏳ Platform Ready Timer Update: ${data.timeLeft}s`);
                updatePlatformReadyTimerOnUI(data.timeLeft);
                break;
            case "platformReadyExpired":
                console.log(`⏳ Platform Ready Timer Expired.`);
                handlePlatformReadyExpired();
                break;
            case "updateNextAttemptTime":
                console.log(`⏳ updated Next Attempt Timer Expired.`);
                updateNextAttemptTimerOnUI(data.timeLeft, data.index);
                break;
            case "nextAttemptExpired":
                console.log(`Received nextAttemptExpired event for index: ${data.index}`);
                handleNextAttemptExpired(data.index);
                break;

            // judge decision Handling
            case "judgeSubmitted":
                console.log(`🎯 Judge submission received from: ${data.judgeId}`);
                showJudgeSubmissionIndicator(data.judgeId);
                break;
            case "displayResults":
                console.log(`🏆 Displaying final results.`);
                displayResults(data);
                break;
            case "clearResults":
                console.log(`🗑 Clearing results from UI.`);
                clearResultsUI();
                break;

            // health check
            case "refereeHealth":
                console.log(`💡 Referee Health Update: ${data.connectedReferees}/${data.requiredReferees} connected.`);
                updateHealthStatus(data.connectedReferees, data.requiredReferees);
                break;

            case "healthError":
                // if user tried to start timer but not all refs connected
                displayMessage(data.message, "red");
                break;

            default:
                console.warn("Unknown action:", data.action);
        }
    };

    //  timer UI Handling (Server-Driven)
    function updatePlatformReadyTimerOnUI(timeLeft) {
        if (platformReadyTimerContainer) {
            platformReadyTimerContainer.classList.remove('hidden');
        }
        if (timerDisplay) {
            console.log(`⏳ Updating Platform Ready Timer UI: ${timeLeft}s`);
            timerDisplay.innerText = `${timeLeft}s`;
        }
    }

    function handlePlatformReadyExpired() {
        if (timerDisplay) timerDisplay.innerText = '0s';
        // displayMessage('Time Up', 'yellow');
    }

    // store a reference to container for all next-attempt timers:
    const multiTimerContainer = document.getElementById('multiNextAttemptTimers');

// keep track of DOM elements for each timer row in a dictionary:
    const nextAttemptRows = {};

// called when we see "updateNextAttemptTime" from the server
    function updateNextAttemptTimerOnUI(timeLeft, timerIndex) {
        // make sure we have a container for all timers
        if (!multiTimerContainer) return;

        // if we don't yet have a row for this index, create one
        if (!nextAttemptRows[timerIndex]) {
            // create a new <div> for the row
            const rowDiv = document.createElement('div');
            rowDiv.classList.add('timer-container');
            rowDiv.style.marginBottom = '10px';

            // create a label
            const label = document.createElement('div');
            label.innerText = `Next Attempt #${timerIndex + 1}:`;
            label.classList.add('timer');  // so it picks up big font if you like
            rowDiv.appendChild(label);

            // create a time display
            const timeSpan = document.createElement('div');
            timeSpan.classList.add('second-timer');
            timeSpan.innerText = `${timeLeft}s`;
            rowDiv.appendChild(timeSpan);

            multiTimerContainer.appendChild(rowDiv);
            // store references
            nextAttemptRows[timerIndex] = { rowDiv, label, timeSpan };
        } else {
            // update existing next attempt timer rows
            nextAttemptRows[timerIndex].timeSpan.innerText = `${timeLeft}s`;
        }
    }

    function handleNextAttemptExpired(timerIndex) {
        console.log(`handleNextAttemptExpired called for timer #${timerIndex + 1}`);

        if (nextAttemptRows[timerIndex]) {
            console.log(`Found timer #${timerIndex + 1} in nextAttemptRows. Removing now.`);

            // Fade out the element before removing it
            const rowDiv = nextAttemptRows[timerIndex].rowDiv;
            rowDiv.style.transition = "opacity 0.5s ease-out";
            rowDiv.style.opacity = "0";

            setTimeout(() => {
                rowDiv.remove();  // Remove from DOM
                delete nextAttemptRows[timerIndex]; // Remove from memory
            }, 500); // Wait for animation to complete
        } else {
            console.warn(`Timer #${timerIndex + 1} not found in nextAttemptRows!`);
        }
    }


    //  decision handling
    function showJudgeSubmissionIndicator(judgeId) {
        const indicator = document.getElementById(`${judgeId}Indicator`);
        if (!indicator) {
            console.error(`Indicator for judgeId "${judgeId}" not found`);
            return;
        }
        indicator.style.backgroundColor = "green";
    }

    function displayResults(data) {
        console.log(`✅ Judge ${judgeId} submitted a decision.`);
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

    //  UI helper functions
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

    // Reset all judge indicators to grey
    function resetJudgeIndicators() {
        const indicators = document.querySelectorAll('.indicator');
        indicators.forEach(indicator => {
            indicator.style.backgroundColor = 'grey';
        });
    }

    // Display a message on the screen
    function displayMessage(text, color) {
        if (!messageElement) return;
        messageElement.innerText = text;
        messageElement.style.color = color;

        // if "Time Up" => flash  // fix_me: redundant
        if (text.includes("Time Up")) {
            messageElement.classList.add('flash');
        } else {
            messageElement.classList.remove('flash');
        }
    }

    //  health check UI
    function updateHealthStatus(connected, required) {
        console.log(`💡 Referee Connection Update: ${connected}/${required} referees connected.`);
        connectedReferees = connected;
        if (connectionStatusElement) {
            connectionStatusElement.innerText = `Referees Connected: ${connected}/${required}`;
            connectionStatusElement.style.color = (connected < required) ? "red" : "green";
        }

        // disable the "Platform Ready" button if not all refs
        if (platformReadyButton) {
            console.log(`🔒 Platform Ready Button: ${connected}/${required} refs connected.`);
            platformReadyButton.disabled = (connected < required);
        }
    }

    //  Platform Ready button logic
    if (platformReadyButton && platformReadyTimerContainer) {
        platformReadyButton.addEventListener('click', () => {
            // current code toggles local container visibility:
            const isHidden = platformReadyTimerContainer.classList.contains('hidden');
            if (isHidden) {
                // request server to "startTimer" (server will check if all refs connected)
                socket.send(JSON.stringify({ action: "startTimer" }));
            } else {
                // if it's visible, we request to stop
                socket.send(JSON.stringify({ action: "stopTimer" }));
            }
            // toggle local display
            platformReadyTimerContainer.classList.toggle('hidden');
        });
    } else {
        console.warn("Platform Ready button or container not found.");
    }
});

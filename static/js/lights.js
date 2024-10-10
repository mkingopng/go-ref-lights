// static/js/lights.js
document.addEventListener('DOMContentLoaded', function() {
    // Ensure websocketUrl is defined
    if (typeof websocketUrl === 'undefined') {
        console.error("websocketUrl is not defined");
        return;
    }

    // Initialize WebSocket connection
    var socket = new WebSocket(websocketUrl);

    socket.onopen = function() {
        console.log("WebSocket connection established for Lights");
    };

    socket.onerror = function(error) {
        console.error("WebSocket error (Lights):", error);
        alert("WebSocket error occurred. Check the console for more details.");
    };

    socket.onclose = function(event) {
        console.log("WebSocket connection closed (Lights):", event);
        alert("WebSocket connection closed.");
    };

    socket.onmessage = function(event) {
        var data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            console.error("Invalid JSON:", event.data);
            return;
        }

        if (data.action === "judgeSubmitted") {
            // Update judge submission indicator
            showJudgeSubmissionIndicator(data.judgeId);
        } else if (data.action === "displayResults") {
            // Display the results from all judges
            displayResults(data);
        } else if (data.action === "startTimer" || data.action === "stopTimer" || data.action === "resetTimer") {
            handleTimerAction(data.action);
        } else {
            console.warn("Unknown action received:", data.action);
        }
    };

    function showJudgeSubmissionIndicator(judgeId) {
        var indicator = document.getElementById(judgeId + "Indicator");
        if (indicator) {
            indicator.style.backgroundColor = "green";
            console.log("Judge Submitted:", judgeId);
        } else {
            console.error("Indicator for judgeId '" + judgeId + "' not found");
        }
    }

    function displayResults(data) {
        // Update circles based on decisions
        updateCircle('leftCircle', data.leftDecision);
        updateCircle('centreCircle', data.centreDecision);
        updateCircle('rightCircle', data.rightDecision);

        // Determine the overall result
        var decisions = [data.leftDecision, data.centreDecision, data.rightDecision];
        var whiteCount = decisions.filter(decision => decision === "white").length;
        var redCount = decisions.filter(decision => decision === "red").length;

        if (whiteCount >= 2) {
            displayMessage('Good Lift', 'white');
        } else if (redCount >= 2) {
            displayMessage('No Lift', 'red');
        } else {
            displayMessage('Mixed Decisions', 'yellow');
        }

        // Start the second timer
        startSecondTimer();
    }

    function updateCircle(circleId, decision) {
        var circle = document.getElementById(circleId);
        if (circle) {
            circle.style.backgroundColor = decision === "white" ? "white" : "red";
            console.log("Circle Updated:", circleId, decision);
        } else {
            console.error("Circle with id '" + circleId + "' not found");
        }
    }

    function displayMessage(text, color) {
        var messageElement = document.getElementById('message');
        messageElement.innerText = text;
        messageElement.style.color = color;
        if (text === 'Time Up') {
            messageElement.classList.add('flash');
        } else {
            messageElement.classList.remove('flash');
        }
    }

    function startSecondTimer() {
        clearInterval(secondTimerInterval);
        secondTimeLeft = 60;
        updateSecondTimerDisplay();
        secondTimerInterval = setInterval(function() {
            secondTimeLeft--;
            updateSecondTimerDisplay();
            if (secondTimeLeft <= 0) {
                clearInterval(secondTimerInterval);
                secondTimeLeft = 0;
                updateSecondTimerDisplay();
                // Optional: Perform action when second timer ends
                displayMessage('Next Attempt', 'green');
            }
        }, 1000);
    }

    function updateSecondTimerDisplay() {
        var secondTimerElement = document.getElementById('secondTimer');
        if (secondTimerElement) {
            secondTimerElement.innerText = secondTimeLeft + 's';
        } else {
            console.error("Element with id 'secondTimer' not found");
        }
    }

    function handleTimerAction(action) {
        switch(action) {
            case "startTimer":
                startTimer();
                break;
            case "stopTimer":
                stopTimer();
                break;
            case "resetTimer":
                resetTimer();
                break;
            default:
                console.warn("Unknown timer action:", action);
        }
    }

    // Timer Variables
    var timerInterval;
    var timeLeft = 60; // Athlete timer

    var secondTimerInterval;
    var secondTimeLeft = 60; // Second timer after judges submit

    // Timer Functions
    function startTimer() {
        if (timerInterval) {
            clearInterval(timerInterval);
        }
        timeLeft = 60; // Reset time
        document.getElementById('timer').innerText = timeLeft + 's';
        timerInterval = setInterval(function() {
            timeLeft--;
            document.getElementById('timer').innerText = timeLeft + 's';
            if (timeLeft <= 0) {
                clearInterval(timerInterval);
                timeLeft = 0;
                document.getElementById('timer').innerText = '0s';
                // Timer reached zero, display message
                displayMessage('Time Up', 'yellow');
            }
        }, 1000);
        console.log("Timer started");
    }

    function stopTimer() {
        if (timerInterval) {
            clearInterval(timerInterval);

            timeLeft = 0;
            document.getElementById('timer').innerText = '0s';
            // Timer reached zero, no UI changes
        }
    }, 1000);
}

function stopTimer() {
    clearInterval(timerInterval);
    displayMessage('Timer Stopped', 'yellow');
}

function resetTimer() {
    clearInterval(timerInterval);
    timeLeft = 60;
    document.getElementById('timer').innerText = timeLeft + 's';
    // Clear any messages
    displayMessage('', '');
    // Reset circle colors
    resetCircles();
    // Clear decisions
    decisions = {};
    removeAllGreenDots();
}

function resetCircles() {
    document.getElementById('leftCircle').style.backgroundColor = 'black';
    document.getElementById('centreCircle').style.backgroundColor = 'black';
    document.getElementById('rightCircle').style.backgroundColor = 'black';
}

function displayMessage(text, color) {
    var messageElement = document.getElementById('message');
    if (text === '') {
        messageElement.style.display = 'none';
    } else {
        messageElement.innerText = text;
        messageElement.style.color = color;
        messageElement.style.display = 'block';
        if (text === 'Time Out') {
            messageElement.classList.add('flash');
        } else {
            messageElement.classList.remove('flash');
        }
            timerInterval = null;
            displayMessage('Timer Stopped', 'yellow');
            console.log("Timer stopped");
        }
    }

function displayAllDecisions(decisionsData) {
    console.log("Aggregated decisions received:", decisionsData);
    // Display all decisions
    var leftDecision = decisionsData.left;
    var centreDecision = decisionsData.centre;
    var rightDecision = decisionsData.right;

    // Update circle colors based on decisions
    updateCircle('leftCircle', leftDecision);
    updateCircle('centreCircle', centreDecision);
    updateCircle('rightCircle', rightDecision);

    // Remove green dots
    removeAllGreenDots();

    // Optionally, display a summary message
    var whiteCount = countDecisions(decisionsData);
    console.log("White Lift Count:", whiteCount);
    if (whiteCount >= 2) {
        displayMessage('Good Lift', 'white');
    } else {
        displayMessage('No Lift', 'red');
    }
}

function updateCircle(circleId, decision) {
    var circle = document.getElementById(circleId);
    if (decision.toLowerCase() === 'good lift') {
        circle.style.backgroundColor = 'white';
    } else {
        circle.style.backgroundColor = 'red';
    }
}

function countDecisions(decisionsData) {
    var whiteCount = 0;
    if (decisionsData.left.toLowerCase() === 'good lift') whiteCount++;
    if (decisionsData.centre.toLowerCase() === 'good lift') whiteCount++;
    if (decisionsData.right.toLowerCase() === 'good lift') whiteCount++;
    return whiteCount;
}

function addGreenDot(referee) {
    var dotContainer = document.getElementById(referee + 'DotContainer');
    console.log(`Adding green dot to: ${referee}DotContainer`);
    if (!dotContainer) {
        console.error(`Dot container not found for referee: ${referee}`);
        return;
    }
    // Prevent multiple dots
    if (document.getElementById(referee + 'Dot')) {
        console.log(`Green dot already exists for referee: ${referee}`);
        return;
    }
    var dot = document.createElement('div');
    dot.className = 'green-dot';
    dot.id = referee + 'Dot';
    dotContainer.appendChild(dot);
    console.log(`Green dot added for referee: ${referee}`);
}

function removeGreenDot(referee) {
    var dot = document.getElementById(referee + 'Dot');
    if (dot) {
        dot.remove();
        console.log(`Green dot removed for referee: ${referee}`);
    }
}

function removeAllGreenDots() {
    ['left', 'centre', 'right'].forEach(function(referee) {
        removeGreenDot(referee);
    });
}

// Function to handle individual referee decisions
function handleIndividualDecision(referee, choice) {
    if (decisions[referee]) {
        alert(`Decision already made for referee: ${referee}`);
        return;
    }
    decisions[referee] = choice;
    addGreenDot(referee);
    sendMessage({ referee: referee, choice: choice });
    console.log(`Decision recorded for referee: ${referee} - ${choice}`);
}

// Wrapper functions for button clicks
function goodLift(referee) {
    handleIndividualDecision(referee, 'Good Lift');
}

function noLift(referee) {
    handleIndividualDecision(referee, 'No Lift');
}
    function resetTimer() {
        if (timerInterval) {
            clearInterval(timerInterval);
            timerInterval = null;
        }
        timeLeft = 60;
        document.getElementById('timer').innerText = timeLeft + 's';
        // Clear any messages
        displayMessage('', '');
        // Reset circles and indicators
        resetForNewLift();
        console.log("Timer reset");
    }

    function resetCircles() {
        document.getElementById('leftCircle').style.backgroundColor = 'black';
        document.getElementById('centreCircle').style.backgroundColor = 'black';
        document.getElementById('rightCircle').style.backgroundColor = 'black';
        console.log("Circles reset to black");
    }

    function resetForNewLift() {
        // Reset circles
        resetCircles();

        // Reset indicators
        var indicators = document.querySelectorAll('.indicator');
        indicators.forEach(indicator => {
            indicator.style.backgroundColor = 'grey';
        });

        // Clear messages
        displayMessage('', '');

        // Stop second timer
        clearInterval(secondTimerInterval);
        var secondTimerElement = document.getElementById('secondTimer');
        if (secondTimerElement) {
            secondTimerElement.innerText = '';
        }
        console.log("Reset for new lift");
    }
});

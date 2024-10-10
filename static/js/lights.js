// static/js/lights.js

// platform Ready Timer Variables
var platformReadyTimerInterval;
var platformReadyTimeLeft = 60; // timer for lifter to make attempt

// next Attempt Timer Variables
var nextAttemptTimerInterval;
var nextAttemptTimeLeft = 60; // timer for lifter to submit next attempt

// judge decisions dict (to store judges decisions)
var judgeDecisions = {
    left: null,
    centre: null,
    right: null
};

document.addEventListener('DOMContentLoaded', function() {
    // ensure websocketUrl is defined
    if (typeof websocketUrl === 'undefined') {
        console.error("websocketUrl is not defined");
        return;
    }

    // initialize WebSocket connection
    var socket = new WebSocket(websocketUrl);

    socket.onopen = function() {
        console.log("WebSocket connection established for Lights");
    };

    socket.onerror = function(error) {
        console.error("WebSocket error (Lights):", error);
    };

    socket.onclose = function(event) {
        console.log("WebSocket connection closed (Lights):", event);
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
            showJudgeSubmissionIndicator(data.judgeId);
        } else if (data.action === "displayResults") {
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
        updateCircle('leftCircle', data.leftDecision);
        updateCircle('centreCircle', data.centreDecision);
        updateCircle('rightCircle', data.rightDecision);

        // store judge decisions for reference
        judgeDecisions.left = data.leftDecision;
        judgeDecisions.centre = data.centreDecision;
        judgeDecisions.right = data.rightDecision;

        // determine the overall result
        var decisions = [data.leftDecision, data.centreDecision, data.rightDecision];
        var whiteCount = decisions.filter(decision => decision === "white").length;
        var redCount = decisions.filter(decision => decision === "red").length;

        if (whiteCount >= 2) {
            displayMessage('Good Lift', 'white');
        } else if (redCount >= 2) {
            displayMessage('No Lift', 'red');
        }

        // start the second timer
        startSecondTimer();

        // clear the message and reset the second timer after 10 seconds
        setTimeout(function() {
            displayMessage('', '');

            // reset platform ready timer to 60 sec, but do NOT start it
            platformReadyTimeLeft = 60;
            document.getElementById('timer').innerText = platformReadyTimeLeft + 's';
            clearInterval(platformReadyTimerInterval);  // Ensure the timer is NOT running
            platformReadyTimerInterval = null;  // Nullify the timer interval to prevent automatic restart
        }, 10000);
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
        clearInterval(nextAttemptTimerInterval);
        nextAttemptTimeLeft = 60;
        updateSecondTimerDisplay();
        nextAttemptTimerInterval = setInterval(function() {
            nextAttemptTimeLeft--;
            updateSecondTimerDisplay();
            if (nextAttemptTimeLeft <= 0) {
                clearInterval(nextAttemptTimerInterval);
                nextAttemptTimeLeft = 0;
                updateSecondTimerDisplay();
                // displayMessage('', 'yellow');

                // Clear the message and reset the second timer after 10 seconds
                setTimeout(function() {
                    displayMessage('', '');  // clear the message
                    nextAttemptTimeLeft = 60;  // reset the second timer to 60 seconds
                    updateSecondTimerDisplay();  // update the timer display
                }, 10000);  // delay 10 seconds b4 resetting timer
            }
        }, 1000);
    }

    function updateSecondTimerDisplay() {
        var secondTimerElement = document.getElementById('secondTimer');
        if (secondTimerElement) {
            secondTimerElement.innerText = nextAttemptTimeLeft + 's';
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

    // Platform Ready Timer Functions
    function startTimer() {
        if (platformReadyTimerInterval) {
            clearInterval(platformReadyTimerInterval);
        }
        platformReadyTimeLeft = 60; // Reset time
        document.getElementById('timer').innerText = platformReadyTimeLeft + 's';
        platformReadyTimerInterval = setInterval(function() {
            platformReadyTimeLeft--;
            document.getElementById('timer').innerText = platformReadyTimeLeft + 's';
            if (platformReadyTimeLeft <= 0) {
                clearInterval(platformReadyTimerInterval);
                platformReadyTimeLeft = 0;
                document.getElementById('timer').innerText = '0s';
                displayMessage('Time Up', 'yellow');
            }
        }, 1000);
        console.log("Timer started");
    }

    function stopTimer() {
        if (platformReadyTimerInterval) {
            clearInterval(platformReadyTimerInterval);
            platformReadyTimerInterval = null;
            console.log("Platform Ready Timer stopped");
        }
    }

    function resetTimer() {
        if (platformReadyTimerInterval) {
            clearInterval(platformReadyTimerInterval);
            platformReadyTimerInterval = null;
        }
        platformReadyTimeLeft = 60;
        document.getElementById('timer').innerText = platformReadyTimeLeft + 's';
        displayMessage('', '');
        console.log("Platform Ready Timer reset");
    }

    function resetCircles() {
        var leftCircle = document.getElementById('leftCircle');
        var centreCircle = document.getElementById('centreCircle');
        var rightCircle = document.getElementById('rightCircle');

        if (leftCircle) {
            leftCircle.style.backgroundColor = 'black';
            console.log("Left Circle reset");
        } else {
            console.error("Left Circle not found");
        }

        if (centreCircle) {
            centreCircle.style.backgroundColor = 'black';
            console.log("Centre Circle reset");
        } else {
            console.error("Centre Circle not found");
        }

        if (rightCircle) {
            rightCircle.style.backgroundColor = 'black';
            console.log("Right Circle reset");
        } else {
            console.error("Right Circle not found");
        }
    }


    function resetForNewLift() {
        resetCircles();

        // Reset indicators
        var indicators = document.querySelectorAll('.indicator');
        indicators.forEach(indicator => {
            indicator.style.backgroundColor = 'grey';
        });

        displayMessage('', '');
        console.log("Reset for new lift");
    }

    // Platform Ready Button Event Handler
    var platformReadyButton = document.getElementById('platformReadyButton');
    var platformReadyContainer = document.getElementById('platformReadyContainer');

    if (platformReadyButton && platformReadyContainer) {
        platformReadyButton.addEventListener('click', function() {
            platformReadyContainer.classList.toggle('hidden');  // Toggle visibility of the platform ready container

            // Start the platform ready timer only if it is visible
            if (!platformReadyContainer.classList.contains('hidden')) {
                startTimer();  // Start the Platform Ready timer when the button is pressed

                // Reset circles and decisions
                resetCircles();
                judgeDecisions.left = null;
                judgeDecisions.centre = null;
                judgeDecisions.right = null;
                console.log("Circles and decisions reset");
            } else {
                stopTimer();  // optionally stop the timer when hidden
            }
        });
    } else {
        console.error("Platform Ready button or container not found.");
    }
});

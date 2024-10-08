// static/js/lights.js

var timerInterval;
var timeLeft = 60; // Athlete timer

var secondTimerInterval;
var secondTimeLeft = 60; // Second timer after judges submit

socket.onmessage = function(event) {
    var data = JSON.parse(event.data);

    if (data.action === "judgeSubmitted") {
        // Update judge submission indicator
        showJudgeSubmissionIndicator(data.judgeId);
    } else if (data.action === "displayResults") {
        // Display the results from all judges
        displayResults(data);
    } else if (data.action) {
        switch(data.action) {
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
            // Unknown action
        }
    }
};

function showJudgeSubmissionIndicator(judgeId) {
    var indicator = document.getElementById(judgeId + "Indicator");
    if (indicator) {
        indicator.style.backgroundColor = "green";
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
    if (whiteCount >= 2) {
        displayMessage('Good Lift', 'white');
    } else {
        displayMessage('No Lift', 'red');
    }

    // Start the second timer
    startSecondTimer();
}

function updateCircle(circleId, decision) {
    var circle = document.getElementById(circleId);
    if (circle) {
        circle.style.backgroundColor = decision === "white" ? "white" : "red";
    }
}

function startTimer() {
    clearInterval(timerInterval);
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
    // Reset circles and indicators
    resetForNewLift();
}

function resetCircles() {
    document.getElementById('leftCircle').style.backgroundColor = 'black';
    document.getElementById('centreCircle').style.backgroundColor = 'black';
    document.getElementById('rightCircle').style.backgroundColor = 'black';
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
        }
    }, 1000);
}

function updateSecondTimerDisplay() {
    var secondTimerElement = document.getElementById('secondTimer');
    if (secondTimerElement) {
        secondTimerElement.innerText = secondTimeLeft + 's';
    }
}

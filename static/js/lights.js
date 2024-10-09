// static/js/lights.js

var timerInterval;
var timeLeft = 60; // in seconds
var decisions = {}; // Stores decisions from referees

// Timer Functions
function startTimer() {
    sendMessage({ action: "startTimer" });
}

function stopTimer() {
    sendMessage({ action: "stopTimer" });
}

function resetTimer() {
    sendMessage({ action: "resetTimer" });
}

// WebSocket Message Handler
socket.onmessage = function(event) {
    var data = JSON.parse(event.data);
    if (data.left && data.centre && data.right) {
        // Aggregated decisions received
        displayAllDecisions(data);
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
                console.warn("Unknown action:", data.action);
        }
    }
};

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

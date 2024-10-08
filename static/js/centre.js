// static/js/centre.js
// Timer Variables
var timerInterval;
var timeLeft = 60; // in seconds

// Initialize WebSocket connection (ensure websocketUrl is defined)
var socket = new WebSocket(websocketUrl);

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

// Function to send decision
function sendDecision(decision) {
    var messageObj = {
        "judgeId": "centre",
        "decision": decision
    };
    sendMessage(messageObj);
}

// Function to send messages
function sendMessage(messageObj) {
    if (socket.readyState === WebSocket.OPEN) {
        var message = JSON.stringify(messageObj);
        socket.send(message);
    } else {
        // Handle error
    }
}

// Event handlers for buttons (assumed to be in your HTML)
document.getElementById('whiteButton').addEventListener('click', function() {
    sendDecision('white');
});

document.getElementById('redButton').addEventListener('click', function() {
    sendDecision('red');
});

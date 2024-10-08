// static/js/centre.js

// Initialize WebSocket connection
var socket = new WebSocket(websocketUrl);

// Function to send decision
function sendDecision(decision) {
    var messageObj = {
        "judgeId": "centre",
        "decision": decision
    };
    sendMessage(messageObj);
}

// Function to send timer actions
function sendTimerAction(action) {
    var messageObj = {
        "action": action
    };
    sendMessage(messageObj);
}

// Function to send messages
function sendMessage(messageObj) {
    if (socket.readyState === WebSocket.OPEN) {
        var message = JSON.stringify(messageObj);
        socket.send(message);
        console.log("Action sent successfully:", messageObj); // For debugging
    } else {
        console.error("WebSocket is not open. ReadyState:", socket.readyState);
        // Optionally, implement retry logic or alert the user
    }
}

// Event handlers for buttons
document.addEventListener('DOMContentLoaded', function() {
    var whiteButton = document.getElementById('whiteButton');
    var redButton = document.getElementById('redButton');
    var startTimerButton = document.getElementById('startTimerButton');
    var stopTimerButton = document.getElementById('stopTimerButton');
    var resetTimerButton = document.getElementById('resetTimerButton');

    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            sendDecision('white');
        });
    } else {
        console.error("Element with id 'whiteButton' not found");
    }

    if (redButton) {
        redButton.addEventListener('click', function() {
            sendDecision('red');
        });
    } else {
        console.error("Element with id 'redButton' not found");
    }

    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            sendTimerAction('startTimer');
        });
    } else {
        console.error("Element with id 'startTimerButton' not found");
    }

    if (stopTimerButton) {
        stopTimerButton.addEventListener('click', function() {
            sendTimerAction('stopTimer');
        });
    } else {
        console.error("Element with id 'stopTimerButton' not found");
    }

    if (resetTimerButton) {
        resetTimerButton.addEventListener('click', function() {
            sendTimerAction('resetTimer');
        });
    } else {
        console.error("Element with id 'resetTimerButton' not found");
    }
});

// static/js/right.js

// Initialize WebSocket connection
var socket = new WebSocket(websocketUrl);

// Function to send decision
function sendDecision(decision) {
    var messageObj = {
        "judgeId": "right", // Correct judgeId
        "decision": decision
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
});

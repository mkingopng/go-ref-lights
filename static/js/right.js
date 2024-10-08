// static/js/left.js

// Initialize WebSocket connection (ensure websocketUrl is defined)
var socket = new WebSocket(websocketUrl);

// Function to send decision
function sendDecision(decision) {
    var messageObj = {
        "judgeId": "right", // Change to "centre" or "right" in respective files
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

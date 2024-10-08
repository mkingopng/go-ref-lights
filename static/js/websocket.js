// static/js/websocket.js

// Initialize WebSocket connection
var socket = new WebSocket(websocketUrl);

socket.onopen = function() {
    // Connection established
};

socket.onerror = function(error) {
    // Handle error
};

socket.onclose = function(event) {
    // Connection closed
};

// Function to send messages
function sendMessage(messageObj) {
    if (socket.readyState === WebSocket.OPEN) {
        var message = JSON.stringify(messageObj);
        socket.send(message);
    } else {
        // Handle error
    }
}

// static/js/websocket.js

// initialize WebSocket connection
var socket = new WebSocket(websocketUrl);

// socket.onopen = function() {
//     console.log("WebSocket connection established.");
// };

// socket.onerror = function(error) {
//     console.error("WebSocket error:", error);
//     alert("WebSocket error occurred. Check the console for more details.");
// };

// socket.onclose = function(event) {
//     console.log("WebSocket connection closed:", event);
//     displayConnectionStatus("WebSocket connection closed.", "orange");
// };

// function to send messages
function sendMessage(messageObj) {
    if (socket.readyState === WebSocket.OPEN) {
        var message = JSON.stringify(messageObj);
        socket.send(message);
        console.log("Sent message:", message);
        // UI Feedback
        // alert("Action sent successfully.");
    } else {
        console.error("WebSocket is not open. Unable to send message.");
        // alert("Unable to send action. WebSocket is not connected.");
    }
}

// function to display connection status
function displayConnectionStatus(message, color) {
    var statusElement = document.getElementById('connectionStatus');
    if (statusElement) {
        statusElement.innerText = message;
        statusElement.style.color = color;
    }
}

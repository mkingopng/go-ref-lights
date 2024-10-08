// static/js/websocket.js

// Initialize WebSocket connection
var socket = new WebSocket(websocketUrl);

socket.onopen = function() {
    console.log("WebSocket connection established.");
    updateConnectionStatus("Connected");
};

socket.onerror = function(error) {
    console.error("WebSocket Error:", error);
    updateConnectionStatus("Error");
    alert("WebSocket connection error. Please try refreshing the page.");
};

socket.onclose = function(event) {
    console.log("WebSocket connection closed:", event);
    updateConnectionStatus("Disconnected");
    alert("WebSocket connection closed. Please refresh the page.");
};

// Function to send messages
function sendMessage(messageObj) {
    if (socket.readyState === WebSocket.OPEN) {
        var message = JSON.stringify(messageObj);
        socket.send(message);
        console.log("Message sent:", messageObj);
    } else {
        console.error("WebSocket is not open. Ready state:", socket.readyState);
        updateConnectionStatus("Disconnected");
        alert("Cannot send message. WebSocket is not connected.");
    }
}

// Function to update connection status on the page
function updateConnectionStatus(status) {
    var statusElement = document.getElementById('connectionStatus');
    if (statusElement) {
        statusElement.innerText = "Connection Status: " + status;
        switch(status) {
            case "Connected":
                statusElement.style.color = "green";
                break;
            case "Error":
                statusElement.style.color = "red";
                break;
            case "Disconnected":
                statusElement.style.color = "orange";
                break;
            default:
                statusElement.style.color = "white";
        }
    }
}

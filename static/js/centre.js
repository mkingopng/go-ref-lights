// static/js/centre.js

document.addEventListener('DOMContentLoaded', function() {
    // Ensure websocketUrl is defined
    if (typeof websocketUrl === 'undefined') {
        console.error("websocketUrl is not defined");
        return;
    }

    // Initialize WebSocket connection
    var socket = new WebSocket(websocketUrl);

    socket.onopen = function() {
        console.log("WebSocket connection established for Centre Referee");
    };

    socket.onerror = function(error) {
        console.error("WebSocket error (Centre Referee):", error);
        // alert("WebSocket error occurred. Check the console for more details.");
    };

    socket.onclose = function(event) {
        console.log("WebSocket connection closed (Centre Referee):", event);
        // alert("WebSocket connection closed.");
    };

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
            console.log("Action sent successfully (Centre Referee):", messageObj);
            // alert("Action sent successfully!");
        } else {
            console.error("WebSocket is not open (Centre Referee). ReadyState:", socket.readyState);
            // alert("Failed to send action. WebSocket is not open.");
        }
    }

    // Event handlers for buttons
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

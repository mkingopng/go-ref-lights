// static/js/centre.js

// Timer Variables
var timerInterval;
var timeLeft = 60; // in seconds

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

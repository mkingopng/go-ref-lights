// static/js/lights.js

var timerInterval;
var timeLeft = 60; // in seconds

socket.onmessage = function(event) {
    var data = JSON.parse(event.data);
    if (data.circleId) {
        var circle = document.getElementById(data.circleId);
        if (circle) {
            circle.style.backgroundColor = data.color;
            checkCircles();
        }
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
}

function resetCircles() {
    document.getElementById('leftCircle').style.backgroundColor = 'black';
    document.getElementById('centreCircle').style.backgroundColor = 'black';
    document.getElementById('rightCircle').style.backgroundColor = 'black';
}

function displayMessage(text, color) {
    var messageElement = document.getElementById('message');
    messageElement.innerText = text;
    messageElement.style.color = color;
    if (text === 'Time Out') {
        messageElement.classList.add('flash');
    } else {
        messageElement.classList.remove('flash');
    }
}

function checkCircles() {
    var leftColor = document.getElementById('leftCircle').style.backgroundColor;
    var centreColor = document.getElementById('centreCircle').style.backgroundColor;
    var rightColor = document.getElementById('rightCircle').style.backgroundColor;

    var colors = [leftColor, centreColor, rightColor];
    var whiteCount = colors.filter(color => color.toLowerCase() === 'white').length;
    var redCount = colors.filter(color => color.toLowerCase() === 'red').length;

    if (whiteCount + redCount === 3) {
        if (whiteCount >= 2) {
            // Good Lift
            displayMessage('Good Lift', 'white');
        } else {
            // No Lift
            displayMessage('No Lift', 'red');
        }
    }
}

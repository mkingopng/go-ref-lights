// static/js/referee-common.js
document.addEventListener('DOMContentLoaded', function() {

    if (typeof websocketUrl === 'undefined') {
        console.error("websocketUrl is not defined");
        return;
    }
    if (typeof judgeId === 'undefined') {
        console.error("judgeId is not defined");
        return;
    }

    // initialize WebSocket
    const socket = new WebSocket(websocketUrl);

    // grab common DOM elements
    const healthEl = document.getElementById("healthStatus");

    // "Centre" has extra timer buttons
    const whiteButton = document.getElementById('whiteButton');
    const redButton   = document.getElementById('redButton');
    const startTimerButton = document.getElementById('startTimerButton');

    // WebSocket event: opened
    socket.onopen = function() {
        console.log(`WebSocket connected for judgeId: ${judgeId}`);

        // immediately register as connected
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId
            // When i have multi-meet, I might also add "meetName":"STATE_CHAMPS" or something
        };
        socket.send(JSON.stringify(registerMsg));
    };

    // WebSocket event: message
    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            console.error("Invalid JSON from server:", event.data);
            return;
        }

        switch (data.action) {
            case "refereeHealth":
                // the server sends {connectedRefIDs:[], connectedReferees: 2, requiredReferees: 3, ...}
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;
            case "healthError":
                // example: "Cannot start timer: Not all referees are connected!"
                alert(data.message);
                break;
            default:
                console.log("Unhandled action:", data.action);
        }
    };

    // webSocket event: error
    socket.onerror = function(error) {
        console.error(`WebSocket error (${judgeId}):`, error);
    };

    // webSocket event: close
    socket.onclose = function(event) {
        console.log(`WebSocket closed (${judgeId}):`, event);
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // utility to send JSON
    function sendMessage(obj) {
        if (socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify(obj));
            console.log("Sent message:", obj);
        } else {
            console.warn(`Cannot send message; socket not open (readyState = ${socket.readyState})`);
        }
    }

    // if these buttons exist, wire them up
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            sendMessage({ judgeId: judgeId, decision: "white" });
        });
    }
    if (redButton) {
        redButton.addEventListener('click', function() {
            sendMessage({ judgeId: judgeId, decision: "red" });
        });
    }
    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            sendMessage({ action: "resetLights" });
            sendMessage({ action: "resetTimer" });
            sendMessage({ action: "startTimer" });
        });
    }
});

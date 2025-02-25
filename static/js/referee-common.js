// static/js/referee-common.js
document.addEventListener('DOMContentLoaded', function() {

    console.log("🏁 Referee script initializing...");
    console.log(`🔍 Checking required variables: websocketUrl=${typeof websocketUrl}, judgeId=${typeof judgeId}`);

    if (typeof websocketUrl === 'undefined') {
        console.error("❌ websocketUrl is not defined. Cannot establish WebSocket connection.");
        return;
    }
    if (typeof judgeId === 'undefined') {
        console.error("❌ judgeId is not defined. Cannot proceed.");
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
        console.log(`🟢 WebSocket connected for judgeId: ${judgeId}`);

        // immediately register as connected
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId
            // When i have multi-meet, I might also add "meetName":"STATE_CHAMPS" or something
        };

        try {
            socket.send(JSON.stringify(registerMsg));
            console.log("📨 Sent referee registration:", registerMsg);
        } catch (error) {
            console.error("❌ Failed to send referee registration:", error);
        }
    };

    // WebSocket event: message
    socket.onmessage = (event) => {
        console.log("📩 WebSocket message received:", event.data);

        let data;
        try {
            data = JSON.parse(event.data);
        } catch (error) {
            console.error("❌ Failed to parse WebSocket message:", event.data, "Error:", error);
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
                console.warn("⚠️ Unhandled WebSocket action:", data.action);
        }
    };

    // webSocket event: error
    socket.onerror = function(error) {
        console.error(`❌ WebSocket error for judgeId: ${judgeId}`, "Error:", error);
    };

    // webSocket event: close
    socket.onclose = function(event) {
        console.warn(`🔴 WebSocket closed for judgeId: ${judgeId}`, "Code:", event.code, "Reason:", event.reason);
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // utility to send JSON
    function sendMessage(obj) {
        if (socket.readyState === WebSocket.OPEN) {
            try {
                socket.send(JSON.stringify(obj));
                console.log("📨 Message sent:", obj);
            } catch (error) {
                console.error("❌ Failed to send message:", obj, "Error:", error);
            }
        } else {
            console.warn(`⚠️ Cannot send message; WebSocket not open (readyState = ${socket.readyState})`);
        }
    }

    // if these buttons exist, wire them up
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            console.log(`⚪ White button clicked by judgeId: ${judgeId}`);
            sendMessage({ judgeId: judgeId, decision: "white" });
        });
    }

    if (redButton) {
        redButton.addEventListener('click', function() {
            console.log(`🔴 Red button clicked by judgeId: ${judgeId}`);
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

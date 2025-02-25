// static/js/referee-common.js

document.addEventListener('DOMContentLoaded', function() {
    // We assume "judgeId" and "websocketUrl" were defined in a <script> block on each HTML page.
    // For example
    //   <script>var judgeId = "centre"; var websocketUrl = "...";</script>
    //   <script src="/static/js/referee-common.js"></script>

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

    console.log(`✅ Detected judgeId: ${judgeId}`);
    console.log(`✅ Detected websocketUrl: ${websocketUrl}`);

    // Initialize WebSocket
    const socket = new WebSocket(websocketUrl);

    // Grab common DOM elements
    const healthEl = document.getElementById("healthStatus");

    // For "Centre" we might also have extra timer buttons, so let's find them safely
    const whiteButton = document.getElementById('whiteButton');
    const redButton   = document.getElementById('redButton');
    const startTimerButton = document.getElementById('startTimerButton');
    const stopTimerButton  = document.getElementById('stopTimerButton');
    const resetTimerButton = document.getElementById('resetTimerButton');

    // WebSocket event: opened
    socket.onopen = function() {
        console.log(`🟢 WebSocket connected for judgeId: ${judgeId}`);

        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId
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
                console.log("🔍 Referee health update received:", data);
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;

            case "healthError":
                console.warn("⚠️ Health error received:", data.message);
                alert(data.message);
                break;

            default:
                console.warn("⚠️ Unhandled WebSocket action:", data.action);
        }
    };

    // WebSocket event: error
    socket.onerror = function(error) {
        console.error(`❌ WebSocket error for judgeId: ${judgeId}`, "Error:", error);
    };

    // WebSocket event: close
    socket.onclose = function(event) {
        console.warn(`🔴 WebSocket closed for judgeId: ${judgeId}`, "Code:", event.code, "Reason:", event.reason);
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // Utility to send JSON
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

    // If these buttons exist, wire them up
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
            sendMessage({ action: "startTimer" });
        });
    }
    if (stopTimerButton) {
        stopTimerButton.addEventListener('click', function() {
            sendMessage({ action: "stopTimer" });
        });
    }
    if (resetTimerButton) {
        resetTimerButton.addEventListener('click', function() {
            sendMessage({ action: "resetTimer" });
        });
    }
});

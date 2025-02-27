// static/js/referee-common.js
document.addEventListener('DOMContentLoaded', function() {
    // custom logger for standardised logging
    const Logger = (function() {
        const isDebug = true;  // set to false in production to disable debug logging

        // helper function to send log messages to the server's /log endpoint.
        function sendLog(level, message) {
            fetch('/log', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ message: message, level: level })
            }).catch(err => {
                // if sending the log fails, output a warning in the console.
                console.warn(`[WARN ${new Date().toISOString()}] Failed to send ${level} log to server:`, err);
            });
        }

        return {
            info: function(...args) {
                const message = `[INFO ${new Date().toISOString()}] ${args.join(" ")}`;
                console.info(message);
                sendLog("info", message);
            },
            warn: function(...args) {
                const message = `[WARN ${new Date().toISOString()}] ${args.join(" ")}`;
                console.warn(message);
                sendLog("warn", message);
            },
            error: function(...args) {
                const message = `[ERROR ${new Date().toISOString()}] ${args.join(" ")}`;
                console.error(message);
                sendLog("error", message);
            },
            debug: function(...args) {
                if (isDebug) {
                    const message = `[DEBUG ${new Date().toISOString()}] ${args.join(" ")}`;
                    console.debug(message);
                    sendLog("debug", message);
                }
            }
        };
    })();

    // validate required globals
    if (typeof websocketUrl === 'undefined') {
        Logger.error("websocketUrl is not defined");
        return;
    }
    if (typeof judgeId === 'undefined') {
        Logger.error("judgeId is not defined");
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
        Logger.info(`WebSocket connected for judgeId: ${judgeId}`);

        // immediately register as connected
        const registerMsg = {
            action: "registerRef",
            judgeId: judgeId,
            meetName: meetId
        };
        socket.send(JSON.stringify(registerMsg));
    };

    // WebSocket event: message
    socket.onmessage = (event) => {
        let data;
        try {
            data = JSON.parse(event.data);
        } catch (e) {
            Logger.error("Invalid JSON from server:", event.data);
            return;
        }

        switch (data.action) {
            case "refereeHealth":
                const isConnected = data.connectedRefIDs.includes(judgeId);
                if (healthEl) {
                    healthEl.innerText = isConnected ? "Connected" : "Disconnected";
                    healthEl.style.color = isConnected ? "green" : "red";
                }
                break;
            case "healthError":
                // for critical errors, we alert the user
                alert(data.message);
                break;
            default:
                Logger.debug("Unhandled action:", data.action);
        }
    };

    // webSocket event: error
    socket.onerror = function(error) {
        Logger.error(`WebSocket error (${judgeId}):`, error);
    };

    // webSocket event: close
    socket.onclose = function(event) {
        Logger.info(`WebSocket closed (${judgeId}):`, event);
        if (healthEl) {
            healthEl.innerText = "Disconnected";
            healthEl.style.color = "red";
        }
    };

    // utility to send JSON
    function sendMessage(obj) {
        if (socket.readyState === WebSocket.OPEN) {
            const messageString = JSON.stringify(obj)
            socket.send(messageString);
            Logger.info("Sent message:", messageString);
        } else {
            Logger.warn(`Cannot send message; socket not open (readyState = ${socket.readyState})`);
        }
    }

    // if these buttons exist, wire them up
    if (whiteButton) {
        whiteButton.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetId,
                judgeId: judgeId,
                decision: "white"
            });
        });
    }
    if (redButton) {
        redButton.addEventListener('click', function() {
            sendMessage({
                action: "submitDecision",
                meetName: meetId,
                judgeId: judgeId,
                decision: "red"
            });
        });
    }
    if (startTimerButton) {
        startTimerButton.addEventListener('click', function() {
            sendMessage({
                action: "resetLights",
                meetName: meetId,
                judgeId: judgeId,
            });
            sendMessage({
                action: "resetTimer",
                meetName: meetId,
                judgeId: judgeId,
            });
            sendMessage({
                action: "startTimer",
                meetName: meetId,
                judgeId: judgeId,
            });
        });
    }
});

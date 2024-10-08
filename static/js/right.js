function sendDecision(decision) {
    var messageObj = {
        "judgeId": "right", // Change accordingly for each judge
        "decision": decision
    };
    sendMessage(messageObj);
}

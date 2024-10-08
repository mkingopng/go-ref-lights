function sendDecision(decision) {
    var messageObj = {
        "judgeId": "left", // Change accordingly for each judge
        "decision": decision
    };
    sendMessage(messageObj);
}

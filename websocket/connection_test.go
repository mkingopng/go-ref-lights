// websocket/connection_test.go

package websocket

//
//import (
//	"encoding/json"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//	"time"
//
//	"github.com/gorilla/websocket"
//	"github.com/stretchr/testify/assert"
//)
//
//// Mock WebSocket Upgrader (to avoid real network connections)
//var testUpgrader = websocket.Upgrader{
//	CheckOrigin: func(r *http.Request) bool {
//		return true
//	},
//}
//
//// Setup a test WebSocket server
//func startTestServer(t *testing.T) (*httptest.Server, *websocket.Conn) {
//	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		ServeWs(w, r)
//	}))
//
//	// Upgrade HTTP to WebSocket connection
//	wsURL := "ws" + server.URL[4:] + "?meetName=TestMeet"
//	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
//	assert.NoError(t, err, "WebSocket connection should succeed")
//
//	return server, conn
//}
//
//// 🟢 Test WebSocket Connection Handling
//func TestServeWs(t *testing.T) {
//	server, conn := startTestServer(t)
//	defer server.Close()
//	defer conn.Close()
//
//	assert.NotNil(t, conn, "WebSocket connection should be established")
//
//	// Ensure connection is registered
//	assert.Equal(t, 1, len(connections), "One connection should be active")
//}
//
//// 🟢 Test Connection Registration and Cleanup
//func TestConnectionRegistration(t *testing.T) {
//	mockConn := &Connection{meetName: "TestMeet"}
//	registerConnection(mockConn)
//	assert.Equal(t, 1, len(connections), "Connection should be registered")
//
//	unregisterConnection(mockConn)
//	assert.Equal(t, 0, len(connections), "Connection should be removed")
//}
//
//// 🟢 Test WebSocket Read Pump (Handling Messages)
//func TestReadPump_ValidMessage(t *testing.T) {
//	server, conn := startTestServer(t)
//	defer server.Close()
//	defer conn.Close()
//
//	message := DecisionMessage{
//		Action:   "registerRef",
//		MeetName: "TestMeet",
//		JudgeID:  "judge1",
//	}
//
//	data, _ := json.Marshal(message)
//	err := conn.WriteMessage(websocket.TextMessage, data)
//	assert.NoError(t, err, "Message should be sent successfully")
//
//	time.Sleep(100 * time.Millisecond) // Allow processing time
//
//	assert.Equal(t, "judge1", getJudgeIDForConnection(conn), "Judge ID should be set correctly")
//}
//
//// 🟢 Test Write Pump (Sending Messages)
//func TestWritePump(t *testing.T) {
//	server, conn := startTestServer(t)
//	defer server.Close()
//	defer conn.Close()
//
//	testConn := &Connection{
//		conn: conn,
//		send: make(chan []byte, 1),
//	}
//	go testConn.writePump()
//
//	testMessage := []byte(`{"action":"testAction"}`)
//	testConn.send <- testMessage
//
//	time.Sleep(100 * time.Millisecond) // Allow processing time
//
//	_, msg, err := conn.ReadMessage()
//	assert.NoError(t, err, "Should be able to read message")
//	assert.JSONEq(t, string(testMessage), string(msg), "Sent and received messages should match")
//}
//
//// 🟢 Test Processing Decision Messages
//func TestProcessDecision(t *testing.T) {
//	mockConn := &Connection{meetName: "TestMeet"}
//	decision := DecisionMessage{
//		Action:   "submitDecision",
//		MeetName: "TestMeet",
//		JudgeID:  "left",
//		Decision: "white",
//	}
//
//	processDecision(mockConn, decision)
//}
//
//// 🟢 Test Broadcasting Messages to a Specific Meet
//func TestBroadcastToMeet(t *testing.T) {
//	server, conn := startTestServer(t)
//	defer server.Close()
//	defer conn.Close()
//
//	mockConn := &Connection{
//		conn:     conn,
//		send:     make(chan []byte, 1),
//		meetName: "TestMeet",
//	}
//	registerConnection(mockConn)
//
//	testMessage := []byte(`{"action":"judgeSubmitted","judgeId":"left"}`)
//	broadcastToMeet("TestMeet", testMessage)
//
//	time.Sleep(100 * time.Millisecond) // Allow processing time
//
//	_, msg, err := conn.ReadMessage()
//	assert.NoError(t, err, "Should receive broadcast message")
//	assert.JSONEq(t, string(testMessage), string(msg), "Broadcasted message should be received correctly")
//}
//
//// 🟢 Test Broadcasting Referee Health Status
//func TestBroadcastRefereeHealth(t *testing.T) {
//	mockConn := &Connection{meetName: "TestMeet", judgeID: "left"}
//	registerConnection(mockConn)
//
//	broadcastRefereeHealth("TestMeet")
//
//	time.Sleep(100 * time.Millisecond) // Allow processing time
//
//	unregisterConnection(mockConn)
//}
//
//// 🔴 Helper function to retrieve a connection’s judge ID
//func getJudgeIDForConnection(conn *websocket.Conn) string {
//	for c := range connections {
//		if c.conn == conn {
//			return c.judgeID
//		}
//	}
//	return ""
//}

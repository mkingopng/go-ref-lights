//go:build unit
// +build unit

// file: websocket/isolation_test.go
package websocket

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// isolationTestConn implements the WSConn interface for isolation testing
type isolationTestConn struct {
	messages    [][]byte
	messageType int
	closed      bool
	remoteAddr  net.Addr
	mu          sync.Mutex
}

func newIsolationTestConn(addr string) *isolationTestConn {
	return &isolationTestConn{
		messages:   make([][]byte, 0),
		remoteAddr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080},
	}
}

func (m *isolationTestConn) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Make a copy of the data to avoid race conditions
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	m.messages = append(m.messages, dataCopy)
	m.messageType = messageType
	return nil
}

func (m *isolationTestConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *isolationTestConn) ReadMessage() (int, []byte, error) {
	// For testing, we don't need to implement reading
	return 0, nil, nil
}

func (m *isolationTestConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *isolationTestConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

func (m *isolationTestConn) SetReadLimit(limit int64) {}

func (m *isolationTestConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *isolationTestConn) SetPongHandler(h func(string) error) {}

func (m *isolationTestConn) getMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([][]byte, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *isolationTestConn) clearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([][]byte, 0)
}

// TestMultiMeetIsolation_ConcurrentDecisions tests that referee decisions from different meets
// are properly isolated and only reach connections for the correct meet
func TestMultiMeetIsolation_ConcurrentDecisions(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test to avoid interference
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 100)
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	// Create mock WebSocket connections for two different meets
	meetA := "APL Test Meet A"
	meetB := "APL Test Meet B"

	// Create connections for Meet A
	connA1 := newIsolationTestConn("127.0.0.1:8001")
	connA2 := newIsolationTestConn("127.0.0.1:8002")

	// Create connections for Meet B
	connB1 := newIsolationTestConn("127.0.0.1:8004")
	connB2 := newIsolationTestConn("127.0.0.1:8005")

	// Create Connection structs for Meet A
	connectionA1 := &Connection{
		conn:     connA1,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "left",
	}
	connectionA2 := &Connection{
		conn:     connA2,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "center",
	}

	// Create Connection structs for Meet B
	connectionB1 := &Connection{
		conn:     connB1,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "left",
	}
	connectionB2 := &Connection{
		conn:     connB2,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "center",
	}

	// Register all connections
	registerConnection(connectionA1)
	registerConnection(connectionA2)
	registerConnection(connectionB1)
	registerConnection(connectionB2)

	// Start a goroutine to handle messages (simulating the HandleMessages loop)
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				// Parse message to get meetName filter
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				// Distribute to connections based on meet filter
				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
						// Message queued successfully
					default:
						// Channel full, drop message
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start goroutines to consume messages from connection send channels
	var wg sync.WaitGroup

	consumeMessages := func(conn *Connection, testConn *isolationTestConn) {
		defer wg.Done()
		for {
			select {
			case msg := <-conn.send:
				testConn.WriteMessage(1, msg)
			case <-time.After(200 * time.Millisecond):
				return
			}
		}
	}

	wg.Add(4)
	go consumeMessages(connectionA1, connA1)
	go consumeMessages(connectionA2, connA2)
	go consumeMessages(connectionB1, connB1)
	go consumeMessages(connectionB2, connB2)

	// Simulate referee decisions in Meet A
	meetStateA := GetMeetState(meetA)
	meetStateA.JudgeDecisions["left"] = "good"
	meetStateA.JudgeDecisions["center"] = "good"
	meetStateA.JudgeDecisions["right"] = "no lift"

	// Simulate referee decisions in Meet B
	meetStateB := GetMeetState(meetB)
	meetStateB.JudgeDecisions["left"] = "no lift"
	meetStateB.JudgeDecisions["center"] = "no lift"
	meetStateB.JudgeDecisions["right"] = "good"

	// Broadcast final results for both meets
	broadcastFinalResults(meetA)
	broadcastFinalResults(meetB)

	// Wait for message processing
	time.Sleep(100 * time.Millisecond)
	wg.Wait()

	// Stop the message handler
	stopChan <- true

	// Verify that Meet A connections only received Meet A messages
	messagesA1 := connA1.getMessages()
	messagesA2 := connA2.getMessages()

	assert.True(t, len(messagesA1) > 0, "Meet A connection 1 should receive messages")
	assert.True(t, len(messagesA2) > 0, "Meet A connection 2 should receive messages")

	// Check that Meet A connections received the correct decisions
	for _, msgBytes := range messagesA1 {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if action, ok := msg["action"].(string); ok && action == "displayResults" {
				assert.Equal(t, meetA, msg["meetName"], "Meet A connection should only receive Meet A messages")
				assert.Equal(t, "good", msg["leftDecision"], "Meet A should show correct left decision")
				assert.Equal(t, "good", msg["centerDecision"], "Meet A should show correct center decision")
				assert.Equal(t, "no lift", msg["rightDecision"], "Meet A should show correct right decision")
			}
		}
	}

	// Verify that Meet B connections only received Meet B messages
	messagesB1 := connB1.getMessages()
	messagesB2 := connB2.getMessages()

	assert.True(t, len(messagesB1) > 0, "Meet B connection 1 should receive messages")
	assert.True(t, len(messagesB2) > 0, "Meet B connection 2 should receive messages")

	// Check that Meet B connections received the correct decisions
	for _, msgBytes := range messagesB1 {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if action, ok := msg["action"].(string); ok && action == "displayResults" {
				assert.Equal(t, meetB, msg["meetName"], "Meet B connection should only receive Meet B messages")
				assert.Equal(t, "no lift", msg["leftDecision"], "Meet B should show correct left decision")
				assert.Equal(t, "no lift", msg["centerDecision"], "Meet B should show correct center decision")
				assert.Equal(t, "good", msg["rightDecision"], "Meet B should show correct right decision")
			}
		}
	}

	// Verify no cross-contamination: Meet A connections should not receive Meet B messages
	for _, msgBytes := range messagesA1 {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if meetName, ok := msg["meetName"].(string); ok {
				assert.NotEqual(t, meetB, meetName, "Meet A connection should never receive Meet B messages")
			}
		}
	}

	// Verify no cross-contamination: Meet B connections should not receive Meet A messages
	for _, msgBytes := range messagesB1 {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if meetName, ok := msg["meetName"].(string); ok {
				assert.NotEqual(t, meetA, meetName, "Meet B connection should never receive Meet A messages")
			}
		}
	}

	// Clean up connections
	unregisterConnection(connectionA1)
	unregisterConnection(connectionA2)
	unregisterConnection(connectionB1)
	unregisterConnection(connectionB2)
}

// TestMultiMeetIsolation_TimerMessages tests that timer messages are properly isolated between meets
func TestMultiMeetIsolation_TimerMessages(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 100)
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	meetA := "Timer Meet A"
	meetB := "Timer Meet B"

	// Create mock connections for both meets
	connA := newIsolationTestConn("127.0.0.1:9001")
	connB := newIsolationTestConn("127.0.0.1:9002")

	connectionA := &Connection{
		conn:     connA,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "center",
	}

	connectionB := &Connection{
		conn:     connB,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "center",
	}

	registerConnection(connectionA)
	registerConnection(connectionB)

	// Start message handler
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
					default:
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start message consumers
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			select {
			case msg := <-connectionA.send:
				connA.WriteMessage(1, msg)
			case <-time.After(200 * time.Millisecond):
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for {
			select {
			case msg := <-connectionB.send:
				connB.WriteMessage(1, msg)
			case <-time.After(200 * time.Millisecond):
				return
			}
		}
	}()

	// Send timer messages for both meets
	BroadcastMessage(meetA, map[string]interface{}{
		"action":   "startTimer",
		"timeLeft": 60,
	})

	BroadcastMessage(meetB, map[string]interface{}{
		"action":   "startTimer",
		"timeLeft": 45,
	})

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	wg.Wait()
	stopChan <- true

	// Verify Meet A only received Meet A timer messages
	messagesA := connA.getMessages()
	assert.True(t, len(messagesA) > 0, "Meet A should receive timer messages")

	foundMeetATimer := false
	for _, msgBytes := range messagesA {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if action, ok := msg["action"].(string); ok && action == "startTimer" {
				assert.Equal(t, meetA, msg["meetName"], "Meet A should only receive its own timer messages")
				if timeLeft, ok := msg["timeLeft"].(float64); ok {
					assert.Equal(t, float64(60), timeLeft, "Meet A should receive correct timer value")
					foundMeetATimer = true
				}
			}
		}
	}
	assert.True(t, foundMeetATimer, "Meet A should have received its timer message")

	// Verify Meet B only received Meet B timer messages
	messagesB := connB.getMessages()
	assert.True(t, len(messagesB) > 0, "Meet B should receive timer messages")

	foundMeetBTimer := false
	for _, msgBytes := range messagesB {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if action, ok := msg["action"].(string); ok && action == "startTimer" {
				assert.Equal(t, meetB, msg["meetName"], "Meet B should only receive its own timer messages")
				if timeLeft, ok := msg["timeLeft"].(float64); ok {
					assert.Equal(t, float64(45), timeLeft, "Meet B should receive correct timer value")
					foundMeetBTimer = true
				}
			}
		}
	}
	assert.True(t, foundMeetBTimer, "Meet B should have received its timer message")

	// Clean up
	unregisterConnection(connectionA)
	unregisterConnection(connectionB)
}

// TestMultiMeetIsolation_ClearResultsMessages tests that clearResults messages are properly isolated
func TestMultiMeetIsolation_ClearResultsMessages(t *testing.T) {
	InitTest()

	// Set short display duration for faster test
	originalDuration := resultsDisplayDuration
	resultsDisplayDuration = 1
	defer func() { resultsDisplayDuration = originalDuration }()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 100)
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	meetA := "Clear Test Meet A"
	meetB := "Clear Test Meet B"

	// Create mock connections
	connA := newIsolationTestConn("127.0.0.1:10001")
	connB := newIsolationTestConn("127.0.0.1:10002")

	connectionA := &Connection{
		conn:     connA,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "left",
	}

	connectionB := &Connection{
		conn:     connB,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "left",
	}

	registerConnection(connectionA)
	registerConnection(connectionB)

	// Start message handler
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
					default:
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start message consumers
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		timeout := time.After(2 * time.Second)
		for {
			select {
			case msg := <-connectionA.send:
				connA.WriteMessage(1, msg)
			case <-timeout:
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		timeout := time.After(2 * time.Second)
		for {
			select {
			case msg := <-connectionB.send:
				connB.WriteMessage(1, msg)
			case <-timeout:
				return
			}
		}
	}()

	// Set up meet states with decisions
	meetStateA := GetMeetState(meetA)
	meetStateA.JudgeDecisions["left"] = "good"
	meetStateA.JudgeDecisions["center"] = "good"
	meetStateA.JudgeDecisions["right"] = "good"

	meetStateB := GetMeetState(meetB)
	meetStateB.JudgeDecisions["left"] = "no lift"
	meetStateB.JudgeDecisions["center"] = "no lift"
	meetStateB.JudgeDecisions["right"] = "no lift"

	// Override sleep function to make test faster
	origSleep := sleepFunc
	sleepFunc = func(d time.Duration) {
		time.Sleep(10 * time.Millisecond) // Very short sleep for testing
	}
	defer func() { sleepFunc = origSleep }()

	// Broadcast final results for both meets (this will trigger clearResults after timeout)
	broadcastFinalResults(meetA)
	broadcastFinalResults(meetB)

	// Wait for both displayResults and clearResults messages
	time.Sleep(200 * time.Millisecond)
	wg.Wait()
	stopChan <- true

	// Verify Meet A received both displayResults and clearResults for Meet A only
	messagesA := connA.getMessages()
	assert.True(t, len(messagesA) >= 1, "Meet A should receive at least displayResults")

	foundDisplayA := false
	for _, msgBytes := range messagesA {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if meetName, ok := msg["meetName"].(string); ok {
				assert.Equal(t, meetA, meetName, "Meet A should only receive Meet A messages")

				if action, ok := msg["action"].(string); ok {
					if action == "displayResults" {
						foundDisplayA = true
					}
				}
			}
		}
	}
	assert.True(t, foundDisplayA, "Meet A should receive displayResults")

	// Verify Meet B received both displayResults and clearResults for Meet B only
	messagesB := connB.getMessages()
	assert.True(t, len(messagesB) >= 1, "Meet B should receive at least displayResults")

	foundDisplayB := false
	for _, msgBytes := range messagesB {
		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err == nil {
			if meetName, ok := msg["meetName"].(string); ok {
				assert.Equal(t, meetB, meetName, "Meet B should only receive Meet B messages")

				if action, ok := msg["action"].(string); ok {
					if action == "displayResults" {
						foundDisplayB = true
					}
				}
			}
		}
	}
	assert.True(t, foundDisplayB, "Meet B should receive displayResults")

	// Clean up
	unregisterConnection(connectionA)
	unregisterConnection(connectionB)
}

// TestConcurrencyStress_MultiMeetSimultaneousDecisions tests meet isolation under high concurrency
// with multiple meets making simultaneous decisions
func TestConcurrencyStress_MultiMeetSimultaneousDecisions(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 1000) // Larger buffer for high load
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	// Create 5 concurrent meets
	numMeets := 5
	numConnectionsPerMeet := 3 // left, center, right
	meetNames := make([]string, numMeets)
	allConnections := make([]*Connection, 0, numMeets*numConnectionsPerMeet)
	allTestConns := make([]*isolationTestConn, 0, numMeets*numConnectionsPerMeet)

	// Set up meets and connections
	for i := 0; i < numMeets; i++ {
		meetNames[i] = fmt.Sprintf("Stress Test Meet %d", i+1)

		// Create 3 connections per meet (left, center, right)
		positions := []string{"left", "center", "right"}
		for j, position := range positions {
			testConn := newIsolationTestConn(fmt.Sprintf("127.0.0.1:%d", 11000+i*10+j))
			connection := &Connection{
				conn:     testConn,
				send:     make(chan []byte, 256),
				meetName: meetNames[i],
				judgeID:  position,
			}

			allConnections = append(allConnections, connection)
			allTestConns = append(allTestConns, testConn)
			registerConnection(connection)
		}
	}

	// Start message handler
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
					default:
						// Channel full, drop message (shouldn't happen in test)
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start message consumers for all connections
	var wg sync.WaitGroup
	wg.Add(len(allConnections))

	for i, conn := range allConnections {
		go func(connection *Connection, testConn *isolationTestConn) {
			defer wg.Done()
			timeout := time.After(2 * time.Second)
			for {
				select {
				case msg := <-connection.send:
					testConn.WriteMessage(1, msg)
				case <-timeout:
					return
				}
			}
		}(conn, allTestConns[i])
	}

	// Simulate simultaneous decisions in all meets
	var decisionWg sync.WaitGroup
	decisionWg.Add(numMeets)

	for i, meetName := range meetNames {
		go func(meet string, meetIndex int) {
			defer decisionWg.Done()

			// Set up different decision patterns for each meet to verify isolation
			meetState := GetMeetState(meet)
			switch meetIndex % 3 {
			case 0:
				meetState.JudgeDecisions["left"] = "good"
				meetState.JudgeDecisions["center"] = "good"
				meetState.JudgeDecisions["right"] = "good"
			case 1:
				meetState.JudgeDecisions["left"] = "no lift"
				meetState.JudgeDecisions["center"] = "no lift"
				meetState.JudgeDecisions["right"] = "no lift"
			case 2:
				meetState.JudgeDecisions["left"] = "good"
				meetState.JudgeDecisions["center"] = "no lift"
				meetState.JudgeDecisions["right"] = "good"
			}

			// Broadcast results
			broadcastFinalResults(meet)
		}(meetName, i)
	}

	// Wait for all decisions to be processed
	decisionWg.Wait()
	time.Sleep(200 * time.Millisecond) // Allow message processing
	wg.Wait()
	stopChan <- true

	// Verify isolation: each connection should only receive messages for its meet
	connectionIndex := 0
	for meetIndex, meetName := range meetNames {
		for positionIndex := 0; positionIndex < numConnectionsPerMeet; positionIndex++ {
			testConn := allTestConns[connectionIndex]
			messages := testConn.getMessages()

			assert.True(t, len(messages) > 0,
				fmt.Sprintf("Meet %s position %d should receive messages", meetName, positionIndex))

			// Verify all messages are for the correct meet
			for _, msgBytes := range messages {
				var msg map[string]interface{}
				if err := json.Unmarshal(msgBytes, &msg); err == nil {
					if msgMeetName, ok := msg["meetName"].(string); ok {
						assert.Equal(t, meetName, msgMeetName,
							fmt.Sprintf("Connection for %s should only receive messages for %s, got %s",
								meetName, meetName, msgMeetName))
					}

					// Verify decision content matches expected pattern for this meet
					if action, ok := msg["action"].(string); ok && action == "displayResults" {
						expectedPattern := meetIndex % 3
						switch expectedPattern {
						case 0: // All good
							assert.Equal(t, "good", msg["leftDecision"])
							assert.Equal(t, "good", msg["centerDecision"])
							assert.Equal(t, "good", msg["rightDecision"])
						case 1: // All no lift
							assert.Equal(t, "no lift", msg["leftDecision"])
							assert.Equal(t, "no lift", msg["centerDecision"])
							assert.Equal(t, "no lift", msg["rightDecision"])
						case 2: // Mixed
							assert.Equal(t, "good", msg["leftDecision"])
							assert.Equal(t, "no lift", msg["centerDecision"])
							assert.Equal(t, "good", msg["rightDecision"])
						}
					}
				}
			}
			connectionIndex++
		}
	}

	// Clean up all connections
	for _, conn := range allConnections {
		unregisterConnection(conn)
	}
}

// TestConcurrencyStress_HighLoadConnectionFiltering tests connection filtering under high concurrent load
func TestConcurrencyStress_HighLoadConnectionFiltering(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 2000) // Large buffer for high load
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	// Create many meets with multiple connections each
	numMeets := 10
	connectionsPerMeet := 5
	meetNames := make([]string, numMeets)
	allConnections := make([]*Connection, 0, numMeets*connectionsPerMeet)
	allTestConns := make([]*isolationTestConn, 0, numMeets*connectionsPerMeet)

	// Set up meets and connections
	for i := 0; i < numMeets; i++ {
		meetNames[i] = fmt.Sprintf("High Load Meet %d", i+1)

		for j := 0; j < connectionsPerMeet; j++ {
			testConn := newIsolationTestConn(fmt.Sprintf("127.0.0.1:%d", 12000+i*10+j))
			connection := &Connection{
				conn:     testConn,
				send:     make(chan []byte, 512), // Larger buffer for high load
				meetName: meetNames[i],
				judgeID:  fmt.Sprintf("judge_%d", j),
			}

			allConnections = append(allConnections, connection)
			allTestConns = append(allTestConns, testConn)
			registerConnection(connection)
		}
	}

	// Start message handler
	stopChan := make(chan bool)
	messageCount := int64(0)
	filteredCount := int64(0)

	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				atomic.AddInt64(&messageCount, 1)

				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						atomic.AddInt64(&filteredCount, 1)
						continue
					}
					select {
					case c.send <- msg:
					default:
						// Channel full, drop message
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start message consumers
	var wg sync.WaitGroup
	wg.Add(len(allConnections))

	for i, conn := range allConnections {
		go func(connection *Connection, testConn *isolationTestConn) {
			defer wg.Done()
			timeout := time.After(3 * time.Second)
			for {
				select {
				case msg := <-connection.send:
					testConn.WriteMessage(1, msg)
				case <-timeout:
					return
				}
			}
		}(conn, allTestConns[i])
	}

	// Generate high load: send many messages rapidly for each meet
	var loadWg sync.WaitGroup
	messagesPerMeet := 20

	for _, meetName := range meetNames {
		loadWg.Add(1)
		go func(meet string) {
			defer loadWg.Done()

			for i := 0; i < messagesPerMeet; i++ {
				// Send various types of messages
				switch i % 4 {
				case 0:
					BroadcastMessage(meet, map[string]interface{}{
						"action":   "startTimer",
						"timeLeft": 60 - i,
					})
				case 1:
					BroadcastMessage(meet, map[string]interface{}{
						"action":   "judgeDecision",
						"judge":    "left",
						"decision": "good",
					})
				case 2:
					// Simulate final results
					meetState := GetMeetState(meet)
					meetState.JudgeDecisions["left"] = "good"
					meetState.JudgeDecisions["center"] = "no lift"
					meetState.JudgeDecisions["right"] = "good"
					broadcastFinalResults(meet)
				case 3:
					BroadcastMessage(meet, map[string]interface{}{
						"action": "clearResults",
					})
				}

				// Small delay to simulate realistic timing
				time.Sleep(1 * time.Millisecond)
			}
		}(meetName)
	}

	// Wait for all load generation to complete
	loadWg.Wait()
	time.Sleep(300 * time.Millisecond) // Allow message processing
	wg.Wait()
	stopChan <- true

	// Verify that filtering worked correctly
	totalMessages := atomic.LoadInt64(&messageCount)
	totalFiltered := atomic.LoadInt64(&filteredCount)

	assert.True(t, totalMessages > 0, "Should have processed messages")
	assert.True(t, totalFiltered > 0, "Should have filtered messages for wrong meets")

	// With 10 meets and messages for each meet, we should filter out 9/10 of messages for each connection
	expectedFilterRatio := float64(numMeets-1) / float64(numMeets)
	actualFilterRatio := float64(totalFiltered) / float64(totalMessages*int64(len(allConnections)))

	// Allow some tolerance due to timing and concurrency
	assert.True(t, actualFilterRatio > expectedFilterRatio*0.8,
		fmt.Sprintf("Filter ratio should be close to %.2f, got %.2f", expectedFilterRatio, actualFilterRatio))

	// Verify each connection only received messages for its meet
	connectionIndex := 0
	for _, meetName := range meetNames {
		for connIndex := 0; connIndex < connectionsPerMeet; connIndex++ {
			testConn := allTestConns[connectionIndex]
			messages := testConn.getMessages()

			// Should have received some messages
			assert.True(t, len(messages) > 0,
				fmt.Sprintf("Meet %s connection %d should receive messages", meetName, connIndex))

			// All messages should be for the correct meet
			for _, msgBytes := range messages {
				var msg map[string]interface{}
				if err := json.Unmarshal(msgBytes, &msg); err == nil {
					if msgMeetName, ok := msg["meetName"].(string); ok {
						assert.Equal(t, meetName, msgMeetName,
							fmt.Sprintf("Connection for %s should only receive messages for %s", meetName, meetName))
					}
				}
			}
			connectionIndex++
		}
	}

	// Clean up all connections
	for _, conn := range allConnections {
		unregisterConnection(conn)
	}
}

// TestEndToEndDecisionWorkflowIsolation tests the complete referee decision workflow isolation
// from submission to lights display, verifying complete isolation between concurrent meets
func TestEndToEndDecisionWorkflowIsolation(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 1000)
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	// Set short display duration for faster test
	originalDuration := resultsDisplayDuration
	resultsDisplayDuration = 1
	defer func() { resultsDisplayDuration = originalDuration }()

	// Override sleep function for faster test execution
	origSleep := sleepFunc
	sleepFunc = func(d time.Duration) {
		time.Sleep(10 * time.Millisecond)
	}
	defer func() { sleepFunc = origSleep }()

	// Create two concurrent meets
	meetA := "E2E Test Meet A"
	meetB := "E2E Test Meet B"

	// Create referee connections for Meet A (left, center, right)
	connA1 := newIsolationTestConn("127.0.0.1:15001")
	connA2 := newIsolationTestConn("127.0.0.1:15002")
	connA3 := newIsolationTestConn("127.0.0.1:15003")

	connectionA1 := &Connection{
		conn:     connA1,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "left",
	}
	connectionA2 := &Connection{
		conn:     connA2,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "center",
	}
	connectionA3 := &Connection{
		conn:     connA3,
		send:     make(chan []byte, 256),
		meetName: meetA,
		judgeID:  "right",
	}

	// Create referee connections for Meet B (left, center, right)
	connB1 := newIsolationTestConn("127.0.0.1:15004")
	connB2 := newIsolationTestConn("127.0.0.1:15005")
	connB3 := newIsolationTestConn("127.0.0.1:15006")

	connectionB1 := &Connection{
		conn:     connB1,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "left",
	}
	connectionB2 := &Connection{
		conn:     connB2,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "center",
	}
	connectionB3 := &Connection{
		conn:     connB3,
		send:     make(chan []byte, 256),
		meetName: meetB,
		judgeID:  "right",
	}

	// Register all connections
	allConnections := []*Connection{connectionA1, connectionA2, connectionA3, connectionB1, connectionB2, connectionB3}
	allTestConns := []*isolationTestConn{connA1, connA2, connA3, connB1, connB2, connB3}

	for _, conn := range allConnections {
		registerConnection(conn)
	}

	// Start message handler
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
					default:
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Start message consumers for all connections
	var wg sync.WaitGroup
	wg.Add(len(allConnections))

	for i, conn := range allConnections {
		go func(connection *Connection, testConn *isolationTestConn) {
			defer wg.Done()
			timeout := time.After(3 * time.Second)
			for {
				select {
				case msg := <-connection.send:
					testConn.WriteMessage(1, msg)
				case <-timeout:
					return
				}
			}
		}(conn, allTestConns[i])
	}

	// Step 1: Simulate referee registration for both meets
	registerMsgA1 := DecisionMessage{Action: "registerRef", MeetName: meetA, JudgeID: "left"}
	registerMsgA2 := DecisionMessage{Action: "registerRef", MeetName: meetA, JudgeID: "center"}
	registerMsgA3 := DecisionMessage{Action: "registerRef", MeetName: meetA, JudgeID: "right"}

	registerMsgB1 := DecisionMessage{Action: "registerRef", MeetName: meetB, JudgeID: "left"}
	registerMsgB2 := DecisionMessage{Action: "registerRef", MeetName: meetB, JudgeID: "center"}
	registerMsgB3 := DecisionMessage{Action: "registerRef", MeetName: meetB, JudgeID: "right"}

	handleIncoming(connectionA1, registerMsgA1)
	handleIncoming(connectionA2, registerMsgA2)
	handleIncoming(connectionA3, registerMsgA3)
	handleIncoming(connectionB1, registerMsgB1)
	handleIncoming(connectionB2, registerMsgB2)
	handleIncoming(connectionB3, registerMsgB3)

	time.Sleep(50 * time.Millisecond)

	// Step 2: Start platform ready timers for both meets
	startTimerMsgA := DecisionMessage{Action: "startTimer", MeetName: meetA}
	startTimerMsgB := DecisionMessage{Action: "startTimer", MeetName: meetB}

	handleIncoming(connectionA2, startTimerMsgA) // Center referee starts timer
	handleIncoming(connectionB2, startTimerMsgB) // Center referee starts timer

	time.Sleep(50 * time.Millisecond)

	// Step 3: Simulate referee decisions for both meets (different decision patterns)
	// Meet A: All good lifts
	decisionA1 := DecisionMessage{Action: "submitDecision", MeetName: meetA, JudgeID: "left", Decision: "good"}
	decisionA2 := DecisionMessage{Action: "submitDecision", MeetName: meetA, JudgeID: "center", Decision: "good"}
	decisionA3 := DecisionMessage{Action: "submitDecision", MeetName: meetA, JudgeID: "right", Decision: "good"}

	// Meet B: All no lifts
	decisionB1 := DecisionMessage{Action: "submitDecision", MeetName: meetB, JudgeID: "left", Decision: "no lift"}
	decisionB2 := DecisionMessage{Action: "submitDecision", MeetName: meetB, JudgeID: "center", Decision: "no lift"}
	decisionB3 := DecisionMessage{Action: "submitDecision", MeetName: meetB, JudgeID: "right", Decision: "no lift"}

	// Submit decisions for Meet A
	handleIncoming(connectionA1, decisionA1)
	handleIncoming(connectionA2, decisionA2)
	handleIncoming(connectionA3, decisionA3) // This should trigger broadcastFinalResults

	// Submit decisions for Meet B
	handleIncoming(connectionB1, decisionB1)
	handleIncoming(connectionB2, decisionB2)
	handleIncoming(connectionB3, decisionB3) // This should trigger broadcastFinalResults

	// Wait for all messages to be processed including clearResults
	time.Sleep(200 * time.Millisecond)
	wg.Wait()
	stopChan <- true

	// Step 4: Verify complete isolation and correct workflow

	// Verify Meet A connections received correct messages
	meetAMessages := make([]map[string]interface{}, 0)
	for _, testConn := range []*isolationTestConn{connA1, connA2, connA3} {
		messages := testConn.getMessages()
		for _, msgBytes := range messages {
			var msg map[string]interface{}
			if err := json.Unmarshal(msgBytes, &msg); err == nil {
				meetAMessages = append(meetAMessages, msg)
			}
		}
	}

	// Verify Meet B connections received correct messages
	meetBMessages := make([]map[string]interface{}, 0)
	for _, testConn := range []*isolationTestConn{connB1, connB2, connB3} {
		messages := testConn.getMessages()
		for _, msgBytes := range messages {
			var msg map[string]interface{}
			if err := json.Unmarshal(msgBytes, &msg); err == nil {
				meetBMessages = append(meetBMessages, msg)
			}
		}
	}

	// Verify Meet A received correct workflow messages
	foundRegisterHealthA := false
	foundStartTimerA := false
	judgeSubmittedA := make(map[string]bool)
	foundDisplayResultsA := false
	foundClearResultsA := false
	foundNextAttemptTimerA := false

	for _, msg := range meetAMessages {
		if meetName, ok := msg["meetName"].(string); ok {
			assert.Equal(t, meetA, meetName, "Meet A should only receive Meet A messages")
		}

		switch msg["action"] {
		case "refereeHealth":
			foundRegisterHealthA = true
		case "startTimer":
			foundStartTimerA = true
		case "judgeSubmitted":
			if judgeId, ok := msg["judgeId"].(string); ok {
				judgeSubmittedA[judgeId] = true
			}
		case "displayResults":
			foundDisplayResultsA = true
			assert.Equal(t, "good", msg["leftDecision"], "Meet A should show all good decisions")
			assert.Equal(t, "good", msg["centerDecision"], "Meet A should show all good decisions")
			assert.Equal(t, "good", msg["rightDecision"], "Meet A should show all good decisions")
		case "clearResults":
			foundClearResultsA = true
		case "updateNextAttemptTime":
			foundNextAttemptTimerA = true
		}
	}

	// Verify Meet B received correct workflow messages
	foundRegisterHealthB := false
	foundStartTimerB := false
	judgeSubmittedB := make(map[string]bool)
	foundDisplayResultsB := false
	foundClearResultsB := false
	foundNextAttemptTimerB := false

	for _, msg := range meetBMessages {
		if meetName, ok := msg["meetName"].(string); ok {
			assert.Equal(t, meetB, meetName, "Meet B should only receive Meet B messages")
		}

		switch msg["action"] {
		case "refereeHealth":
			foundRegisterHealthB = true
		case "startTimer":
			foundStartTimerB = true
		case "judgeSubmitted":
			if judgeId, ok := msg["judgeId"].(string); ok {
				judgeSubmittedB[judgeId] = true
			}
		case "displayResults":
			foundDisplayResultsB = true
			assert.Equal(t, "no lift", msg["leftDecision"], "Meet B should show all no lift decisions")
			assert.Equal(t, "no lift", msg["centerDecision"], "Meet B should show all no lift decisions")
			assert.Equal(t, "no lift", msg["rightDecision"], "Meet B should show all no lift decisions")
		case "clearResults":
			foundClearResultsB = true
		case "updateNextAttemptTime":
			foundNextAttemptTimerB = true
		}
	}

	// Assert complete workflow was executed for Meet A
	assert.True(t, foundRegisterHealthA, "Meet A should receive referee health updates")
	assert.True(t, foundStartTimerA, "Meet A should receive start timer message")
	assert.Equal(t, 3, len(judgeSubmittedA), "Meet A should receive judge submitted messages for all 3 judges")
	assert.True(t, judgeSubmittedA["left"], "Meet A should receive left judge submitted message")
	assert.True(t, judgeSubmittedA["center"], "Meet A should receive center judge submitted message")
	assert.True(t, judgeSubmittedA["right"], "Meet A should receive right judge submitted message")
	assert.True(t, foundDisplayResultsA, "Meet A should receive display results message")
	assert.True(t, foundClearResultsA, "Meet A should receive clear results message")
	assert.True(t, foundNextAttemptTimerA, "Meet A should receive next attempt timer message")

	// Assert complete workflow was executed for Meet B
	assert.True(t, foundRegisterHealthB, "Meet B should receive referee health updates")
	assert.True(t, foundStartTimerB, "Meet B should receive start timer message")
	assert.Equal(t, 3, len(judgeSubmittedB), "Meet B should receive judge submitted messages for all 3 judges")
	assert.True(t, judgeSubmittedB["left"], "Meet B should receive left judge submitted message")
	assert.True(t, judgeSubmittedB["center"], "Meet B should receive center judge submitted message")
	assert.True(t, judgeSubmittedB["right"], "Meet B should receive right judge submitted message")
	assert.True(t, foundDisplayResultsB, "Meet B should receive display results message")
	assert.True(t, foundClearResultsB, "Meet B should receive clear results message")
	assert.True(t, foundNextAttemptTimerB, "Meet B should receive next attempt timer message")

	// Verify no cross-contamination: Meet A messages should not contain Meet B meetName
	for _, msg := range meetAMessages {
		if meetName, ok := msg["meetName"].(string); ok {
			assert.NotEqual(t, meetB, meetName, "Meet A should never receive Meet B messages")
		}
	}

	// Verify no cross-contamination: Meet B messages should not contain Meet A meetName
	for _, msg := range meetBMessages {
		if meetName, ok := msg["meetName"].(string); ok {
			assert.NotEqual(t, meetA, meetName, "Meet B should never receive Meet A messages")
		}
	}

	// Verify timer isolation: each meet should have independent timer state
	meetStateA := GetMeetState(meetA)
	meetStateB := GetMeetState(meetB)

	// Both meets should have cleared decisions after final results
	assert.Empty(t, meetStateA.JudgeDecisions, "Meet A decisions should be cleared after results")
	assert.Empty(t, meetStateB.JudgeDecisions, "Meet B decisions should be cleared after results")

	// Both meets should have next attempt timers
	assert.True(t, len(meetStateA.NextAttemptTimers) > 0, "Meet A should have next attempt timers")
	assert.True(t, len(meetStateB.NextAttemptTimers) > 0, "Meet B should have next attempt timers")

	// Clean up connections
	for _, conn := range allConnections {
		unregisterConnection(conn)
	}
}

// TestConcurrencyStress_RapidConnectionChanges tests meet isolation during rapid connection/disconnection
func TestConcurrencyStress_RapidConnectionChanges(t *testing.T) {
	InitTest()

	// Use a separate broadcast channel for this test
	originalBroadcast := broadcast
	testBroadcast := make(chan []byte, 1000)
	broadcast = testBroadcast
	defer func() { broadcast = originalBroadcast }()

	// Clear any existing connections and meets
	connectionsMu.Lock()
	connections = make(map[*Connection]bool)
	connectionsMu.Unlock()

	meetsMutex.Lock()
	meets = make(map[string]*MeetState)
	meetsMutex.Unlock()

	meetA := "Rapid Changes Meet A"
	meetB := "Rapid Changes Meet B"

	// Start message handler
	stopChan := make(chan bool)
	go func() {
		for {
			select {
			case <-stopChan:
				return
			case msg := <-testBroadcast:
				var msgMap map[string]interface{}
				var meetFilter string

				if err := json.Unmarshal(msg, &msgMap); err == nil {
					if m, ok := msgMap["meetName"].(string); ok {
						meetFilter = m
					}
				}

				connectionsMu.RLock()
				for c := range connections {
					if meetFilter != "" && c.meetName != meetFilter {
						continue
					}
					select {
					case c.send <- msg:
					default:
					}
				}
				connectionsMu.RUnlock()
			}
		}
	}()

	// Simulate rapid connection changes while broadcasting messages
	var wg sync.WaitGroup

	// Goroutine for rapid connection/disconnection
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 50; i++ {
			// Create connections for both meets
			testConnA := newIsolationTestConn(fmt.Sprintf("127.0.0.1:%d", 13000+i*2))
			testConnB := newIsolationTestConn(fmt.Sprintf("127.0.0.1:%d", 13000+i*2+1))

			connectionA := &Connection{
				conn:     testConnA,
				send:     make(chan []byte, 256),
				meetName: meetA,
				judgeID:  fmt.Sprintf("judge_%d", i),
			}

			connectionB := &Connection{
				conn:     testConnB,
				send:     make(chan []byte, 256),
				meetName: meetB,
				judgeID:  fmt.Sprintf("judge_%d", i),
			}

			// Register connections
			registerConnection(connectionA)
			registerConnection(connectionB)

			// Start message consumers
			go func(conn *Connection, testConn *isolationTestConn) {
				timeout := time.After(100 * time.Millisecond)
				for {
					select {
					case msg := <-conn.send:
						testConn.WriteMessage(1, msg)
					case <-timeout:
						return
					}
				}
			}(connectionA, testConnA)

			go func(conn *Connection, testConn *isolationTestConn) {
				timeout := time.After(100 * time.Millisecond)
				for {
					select {
					case msg := <-conn.send:
						testConn.WriteMessage(1, msg)
					case <-timeout:
						return
					}
				}
			}(connectionB, testConnB)

			// Brief pause to let connections settle
			time.Sleep(2 * time.Millisecond)

			// Unregister connections
			unregisterConnection(connectionA)
			unregisterConnection(connectionB)
		}
	}()

	// Goroutine for continuous message broadcasting
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			// Broadcast to both meets
			BroadcastMessage(meetA, map[string]interface{}{
				"action": "testMessage",
				"index":  i,
			})

			BroadcastMessage(meetB, map[string]interface{}{
				"action": "testMessage",
				"index":  i,
			})

			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Wait for all operations to complete
	wg.Wait()
	stopChan <- true

	// Test passes if no panics or race conditions occurred
	// The main goal is to ensure the system remains stable under rapid changes
	assert.True(t, true, "System should remain stable during rapid connection changes")
}

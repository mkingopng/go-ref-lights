// File: tests/simulation_test.go
//go:build integration

package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const (
	testMeetName = "test_mule"
	waitTimeout  = 5 * time.Second
	pollInterval = 100 * time.Millisecond
)

// MessageCapture is a helper to collect and analyze WebSocket messages
type MessageCapture struct {
	messages [][]byte
	mu       sync.Mutex
}

// Add appends a message to the captured collection
func (mc *MessageCapture) Add(msg []byte) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = append(mc.messages, msg)
}

// Count returns the number of messages matching a specific action
func (mc *MessageCapture) Count(action string) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	count := 0
	for _, m := range mc.messages {
		var tmp map[string]interface{}
		if err := json.Unmarshal(m, &tmp); err == nil {
			if a, ok := tmp["action"].(string); ok && a == action {
				count++
			}
		}
	}
	return count
}

// Find returns the first message matching a specific action
func (mc *MessageCapture) Find(action string) map[string]interface{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	for _, m := range mc.messages {
		var tmp map[string]interface{}
		if err := json.Unmarshal(m, &tmp); err == nil {
			if a, ok := tmp["action"].(string); ok && a == action {
				return tmp
			}
		}
	}
	return nil
}

// FindAll returns all messages matching a specific action
func (mc *MessageCapture) FindAll(action string) []map[string]interface{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	var result []map[string]interface{}
	for _, m := range mc.messages {
		var tmp map[string]interface{}
		if err := json.Unmarshal(m, &tmp); err == nil {
			if a, ok := tmp["action"].(string); ok && a == action {
				result = append(result, tmp)
			}
		}
	}
	return result
}

// Clear removes all captured messages
func (mc *MessageCapture) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messages = mc.messages[:0]
}

// LaunchClient represents a WebSocket client in the test
type LaunchClient struct {
	ID       string
	Position string
	Conn     *websocket.Conn
}

// SimulationTest provides a test harness for full system simulations
type SimulationTest struct {
	t               *testing.T
	server          *httptest.Server
	baseURL         string
	wsURL           string
	meetName        string
	director        *websocket.Conn
	directorCapture *MessageCapture
	referees        map[string]*LaunchClient
	cookie          *http.Cookie // For session management
}

// NewSimulationTest creates a new test harness
func NewSimulationTest(t *testing.T, meetName string) *SimulationTest {
	// Set up router using the test helper
	router := SetupTestRouter("test")

	// Create test server
	srv := httptest.NewServer(router)

	baseURL := srv.URL
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/referee-updates?meetName=" + meetName

	sim := &SimulationTest{
		t:               t,
		server:          srv,
		baseURL:         baseURL,
		wsURL:           wsURL,
		meetName:        meetName,
		directorCapture: &MessageCapture{},
		referees:        make(map[string]*LaunchClient),
	}

	// Set up director client
	sim.setupDirector()

	return sim
}

// Close shuts down the test harness
func (s *SimulationTest) Close() {
	// Close all referee connections
	for _, r := range s.referees {
		if r.Conn != nil {
			r.Conn.Close()
		}
	}

	// Close director connection
	if s.director != nil {
		s.director.Close()
	}

	// Close test server
	if s.server != nil {
		s.server.Close()
	}
}

// setupDirector sets up the meet director websocket connection
func (s *SimulationTest) setupDirector() {
	var err error
	s.director, _, err = websocket.DefaultDialer.Dial(s.wsURL, nil)
	require.NoError(s.t, err, "Failed to connect director websocket")

	// Start message capture goroutine
	go func() {
		for {
			_, msg, err := s.director.ReadMessage()
			if err != nil {
				return
			}
			s.directorCapture.Add(msg)
		}
	}()
}

// CreateReferee creates a new referee client with the given position
func (s *SimulationTest) CreateReferee(position string) *LaunchClient {
	c, _, err := websocket.DefaultDialer.Dial(s.wsURL, nil)
	require.NoError(s.t, err, "Failed to connect referee websocket")

	ref := &LaunchClient{
		ID:       position,
		Position: position,
		Conn:     c,
	}

	s.referees[position] = ref

	// Register referee identity
	err = c.WriteJSON(map[string]string{
		"action":   "registerRef",
		"judgeId":  position,
		"meetName": s.meetName,
	})
	require.NoError(s.t, err, "Failed to register referee")

	return ref
}

// SendMessageAs sends a message as a specific referee
func (s *SimulationTest) SendMessageAs(position string, message map[string]interface{}) {
	ref, ok := s.referees[position]
	require.True(s.t, ok, "Referee %s not found", position)

	if _, ok := message["meetName"]; !ok {
		message["meetName"] = s.meetName
	}
	if _, ok := message["judgeId"]; !ok {
		message["judgeId"] = position
	}

	err := ref.Conn.WriteJSON(message)
	require.NoError(s.t, err, "Failed to send message as %s", position)
}

// CloseReferee closes a referee connection
func (s *SimulationTest) CloseReferee(position string) {
	ref, ok := s.referees[position]
	require.True(s.t, ok, "Referee %s not found", position)

	err := ref.Conn.Close()
	require.NoError(s.t, err, "Failed to close referee connection")
	delete(s.referees, position)
}

// ReconnectReferee reconnects a referee after it was closed
func (s *SimulationTest) ReconnectReferee(position string) {
	_, exists := s.referees[position]
	require.False(s.t, exists, "Referee %s already connected", position)

	s.CreateReferee(position)
}

// HTTPPost performs an HTTP POST request to the server
func (s *SimulationTest) HTTPPost(endpoint string, form url.Values) *http.Response {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects automatically so we can check status code 302
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("POST", s.baseURL+endpoint, strings.NewReader(form.Encode()))
	require.NoError(s.t, err, "Failed to create request")

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}

	resp, err := client.Do(req)
	require.NoError(s.t, err, "Failed to execute request")

	// Store cookie if present for session management
	if len(resp.Cookies()) > 0 {
		s.cookie = resp.Cookies()[0]
	}

	return resp
}

// AdminLogin logs in as an admin
func (s *SimulationTest) AdminLogin() {
	// First, check if cookie is already set
	if s.cookie != nil {
		return
	}

	// Perform login using test-login endpoint
	form := url.Values{}
	form.Add("meetName", s.meetName)
	form.Add("username", "test")

	resp := s.HTTPPost("/test-login", form)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Admin login failed")
}

// ForceVacate forcibly vacates a position
func (s *SimulationTest) ForceVacate(position string) {
	s.AdminLogin()

	form := url.Values{}
	form.Add("meetName", s.meetName)
	form.Add("position", position)

	resp := s.HTTPPost("/force-vacate", form)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Force vacate failed")
}

// ResetInstance resets the meet instance
func (s *SimulationTest) ResetInstance() {
	s.AdminLogin()

	form := url.Values{}
	form.Add("meetName", s.meetName)

	resp := s.HTTPPost("/reset-instance", form)
	defer resp.Body.Close()

	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Reset instance failed")
}

// VacatePosition sends a vacate position request
func (s *SimulationTest) VacatePosition(position string) *http.Response {
	// Set up a cookie for the session
	if s.cookie == nil {
		// Need to establish a session first by creating a referee
		if _, ok := s.referees[position]; !ok {
			s.CreateReferee(position)
		}
	}

	form := url.Values{}
	resp := s.HTTPPost("/position/vacate", form)

	return resp
}

// WaitForMessages waits for a specific number of messages with the given action
func (s *SimulationTest) WaitForMessages(action string, count int) bool {
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		if s.directorCapture.Count(action) >= count {
			return true
		}
		time.Sleep(pollInterval)
	}
	return false
}

// TestFullSimulation runs a full system simulation test
func TestFullSimulation(t *testing.T) {
	// Create test harness
	sim := NewSimulationTest(t, testMeetName)
	defer sim.Close()

	// ====== Scenario S1: Three referees claim seats ======
	t.Run("S1: Three referees claim seats", func(t *testing.T) {
		// Clear any previous messages
		sim.directorCapture.Clear()

		// Create three referees
		refIDs := []string{"left", "center", "right"}
		for _, id := range refIDs {
			sim.CreateReferee(id)
		}

		// Wait for 3 occupancyChanged messages
		require.Eventually(t, func() bool {
			return sim.directorCapture.Count("occupancyChanged") >= 3
		}, waitTimeout, pollInterval, "Didn't receive 3 occupancyChanged messages")

		// Check that the last occupancyChanged message contains all three referees
		occupancyMsgs := sim.directorCapture.FindAll("occupancyChanged")
		require.GreaterOrEqual(t, len(occupancyMsgs), 3, "Not enough occupancyChanged messages")

		lastMsg := occupancyMsgs[len(occupancyMsgs)-1]
		require.NotEmpty(t, lastMsg["leftUser"], "Left user not set in occupancy")
		require.NotEmpty(t, lastMsg["centerUser"], "Center user not set in occupancy")
		require.NotEmpty(t, lastMsg["rightUser"], "Right user not set in occupancy")
	})

	// ====== Scenario S2: Platform ready timer and decision submissions ======
	t.Run("S2: Platform ready timer and decisions", func(t *testing.T) {
		// Clear previous messages
		sim.directorCapture.Clear()

		// Center referee starts the timer
		sim.SendMessageAs("center", map[string]interface{}{
			"action": "startTimer",
		})

		// Wait for startTimer broadcast
		require.Eventually(t, func() bool {
			return sim.directorCapture.Count("startTimer") >= 1
		}, waitTimeout, pollInterval, "Didn't receive startTimer message")

		// Wait for some timer updates
		require.Eventually(t, func() bool {
			return sim.directorCapture.Count("updatePlatformReadyTime") >= 3
		}, waitTimeout, pollInterval, "Didn't receive timer updates")

		// Submit decisions from all referees
		refIDs := []string{"left", "center", "right"}
		for _, id := range refIDs {
			sim.SendMessageAs(id, map[string]interface{}{
				"action":   "submitDecision",
				"decision": "good",
			})

			// Wait for judgeSubmitted message
			require.Eventually(t, func() bool {
				return sim.directorCapture.Find("judgeSubmitted") != nil
			}, waitTimeout, pollInterval, "Didn't receive judgeSubmitted message")
		}

		// Wait for final displayResults message
		require.Eventually(t, func() bool {
			return sim.directorCapture.Find("displayResults") != nil
		}, waitTimeout, pollInterval, "Didn't receive displayResults message")

		// Verify displayResults has all three decisions
		displayResultsMsg := sim.directorCapture.Find("displayResults")
		require.NotNil(t, displayResultsMsg, "displayResults message not found")
		require.Equal(t, "good", displayResultsMsg["leftDecision"], "Wrong left decision")
		require.Equal(t, "good", displayResultsMsg["centerDecision"], "Wrong center decision")
		require.Equal(t, "good", displayResultsMsg["rightDecision"], "Wrong right decision")
	})

	// ====== Scenario S3: Referee network drop and reconnect ======
	t.Run("S3: Referee disconnects and reconnects", func(t *testing.T) {
		// Clear previous messages
		sim.directorCapture.Clear()

		// Ensure we have a full set of referees
		if len(sim.referees) < 3 {
			refIDs := []string{"left", "center", "right"}
			for _, id := range refIDs {
				if _, ok := sim.referees[id]; !ok {
					sim.CreateReferee(id)
				}
			}
		}

		// Wait for initial health broadcast showing 3 referees
		require.Eventually(t, func() bool {
			msg := sim.directorCapture.Find("refereeHealth")
			if msg == nil {
				return false
			}
			connectedRefs, ok := msg["connectedReferees"].(float64)
			return ok && connectedRefs == 3
		}, waitTimeout, pollInterval, "Initial referee health check failed")

		// Close one referee connection
		sim.CloseReferee("left")

		// Wait for health update showing 2 referees
		require.Eventually(t, func() bool {
			// Clear old messages and wait for a new health update
			sim.directorCapture.Clear()

			// Wait a bit for health update
			time.Sleep(2 * time.Second)

			msg := sim.directorCapture.Find("refereeHealth")
			if msg == nil {
				return false
			}
			connectedRefs, ok := msg["connectedReferees"].(float64)
			return ok && connectedRefs == 2
		}, waitTimeout, pollInterval, "Referee health didn't update after disconnect")

		// Reconnect the referee
		sim.ReconnectReferee("left")

		// Wait for health update showing 3 referees again
		require.Eventually(t, func() bool {
			msg := sim.directorCapture.Find("refereeHealth")
			if msg == nil {
				return false
			}
			connectedRefs, ok := msg["connectedReferees"].(float64)
			return ok && connectedRefs == 3
		}, waitTimeout, pollInterval, "Referee health didn't update after reconnect")
	})

	// ====== Scenario S4: Referee vacates position ======
	t.Run("S4: Referee vacates position", func(t *testing.T) {
		// Ensure left referee exists
		if _, ok := sim.referees["left"]; !ok {
			sim.CreateReferee("left")
		}

		// Clear messages
		sim.directorCapture.Clear()

		// Vacate position
		resp := sim.VacatePosition("left")
		require.Equal(t, http.StatusFound, resp.StatusCode, "Vacate position didn't return 302 redirect")

		location := resp.Header.Get("Location")
		require.Equal(t, "/logout?reason=vacate", location, "Incorrect redirect location")

		// Wait for occupancyChanged message showing empty left position
		require.Eventually(t, func() bool {
			msg := sim.directorCapture.Find("occupancyChanged")
			if msg == nil {
				return false
			}
			leftUser, ok := msg["leftUser"].(string)
			return ok && leftUser == ""
		}, waitTimeout, pollInterval, "Left position not vacated")
	})

	// ====== Scenario S5: Admin force vacate ======
	t.Run("S5: Admin force vacate", func(t *testing.T) {
		// Ensure center referee exists
		if _, ok := sim.referees["center"]; !ok {
			sim.CreateReferee("center")
		}

		// Clear messages
		sim.directorCapture.Clear()

		// Force vacate center position
		sim.ForceVacate("center")

		// Wait for occupancyChanged message showing empty center position
		require.Eventually(t, func() bool {
			msg := sim.directorCapture.Find("occupancyChanged")
			if msg == nil {
				return false
			}
			centerUser, ok := msg["centerUser"].(string)
			return ok && centerUser == ""
		}, waitTimeout, pollInterval, "Center position not force vacated")
	})

	// ====== Scenario S6: Admin reset instance ======
	t.Run("S6: Admin reset instance", func(t *testing.T) {
		// Make sure we have some state to reset
		if _, ok := sim.referees["right"]; !ok {
			sim.CreateReferee("right")
		}

		// Submit a decision to create some judge decisions state
		sim.SendMessageAs("right", map[string]interface{}{
			"action":   "submitDecision",
			"decision": "bad",
		})

		// Clear messages
		sim.directorCapture.Clear()

		// Reset instance
		sim.ResetInstance()

		// Wait for occupancyChanged message showing all empty positions
		require.Eventually(t, func() bool {
			msg := sim.directorCapture.Find("occupancyChanged")
			if msg == nil {
				return false
			}

			leftUser, leftOk := msg["leftUser"].(string)
			centerUser, centerOk := msg["centerUser"].(string)
			rightUser, rightOk := msg["rightUser"].(string)

			return leftOk && leftUser == "" &&
				centerOk && centerUser == "" &&
				rightOk && rightUser == ""
		}, waitTimeout, pollInterval, "Instance not reset properly")

		// Verify judge decisions map is cleared in the backend
		// This requires checking DefaultStateProvider directly,
		// which we can't easily do in this test.
		// In a real implementation, you'd inject a mock state provider or
		// add an API endpoint that returns this information.
	})
}

// Additional integration tests for high load scenarios

// TestHighLoad tests broadcasting to many clients
func TestHighLoad(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping high load test in short mode")
	}

	// Create test harness
	sim := NewSimulationTest(t, testMeetName)
	defer sim.Close()

	// Create many referee clients (50)
	numReferees := 50

	// Use different IDs for high load test
	for i := 0; i < numReferees; i++ {
		c, _, err := websocket.DefaultDialer.Dial(sim.wsURL, nil)
		require.NoError(t, err)
		defer c.Close()

		id := fmt.Sprintf("load-ref-%d", i)

		// Register referee
		err = c.WriteJSON(map[string]string{
			"action":   "registerRef",
			"judgeId":  id,
			"meetName": sim.meetName,
		})
		require.NoError(t, err)
	}

	// Start a timer
	sim.SendMessageAs("center", map[string]interface{}{
		"action": "startTimer",
	})

	// Verify we can handle the broadcast load
	require.Eventually(t, func() bool {
		return sim.directorCapture.Count("startTimer") >= 1
	}, waitTimeout, pollInterval, "High load broadcast failed")
}

// TestConcurrentMeets tests operating multiple simultaneous meets
func TestConcurrentMeets(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping concurrent meets test in short mode")
	}

	// Number of concurrent meets
	numMeets := 5
	sims := make([]*SimulationTest, numMeets)

	// Create test harnesses for each meet
	for i := 0; i < numMeets; i++ {
		meetName := fmt.Sprintf("TestMeet%d", i)
		sims[i] = NewSimulationTest(t, meetName)
		defer sims[i].Close()

		// Create referees for this meet
		positions := []string{"left", "center", "right"}
		for _, pos := range positions {
			sims[i].CreateReferee(pos)
		}
	}

	// Run parallel activity in all meets
	var wg sync.WaitGroup
	for i := 0; i < numMeets; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sim := sims[index]

			// Start a timer
			sim.SendMessageAs("center", map[string]interface{}{
				"action": "startTimer",
			})

			// Submit decisions
			sim.SendMessageAs("left", map[string]interface{}{
				"action":   "submitDecision",
				"decision": "good",
			})

			sim.SendMessageAs("center", map[string]interface{}{
				"action":   "submitDecision",
				"decision": "bad",
			})

			sim.SendMessageAs("right", map[string]interface{}{
				"action":   "submitDecision",
				"decision": "good",
			})

			// Verify results are displayed
			require.Eventually(t, func() bool {
				return sim.directorCapture.Find("displayResults") != nil
			}, waitTimeout, pollInterval, "Meet %d didn't display results", index)
		}(i)
	}

	// Wait for all meets to complete their activity
	wg.Wait()
}

// TestChaosMonkey randomly disconnects and reconnects clients
func TestChaosMonkey(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping chaos monkey test in short mode")
	}

	sim := NewSimulationTest(t, testMeetName)
	defer sim.Close()

	// Create initial referees
	positions := []string{"left", "center", "right"}
	for _, pos := range positions {
		sim.CreateReferee(pos)
	}

	// Start a timer
	sim.SendMessageAs("center", map[string]interface{}{
		"action": "startTimer",
	})

	// Run chaos monkey in background to randomly disconnect/reconnect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var chaosWg sync.WaitGroup
	chaosWg.Add(1)
	go func() {
		defer chaosWg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
				// Randomly disconnect a referee
				randomPos := positions[int(time.Now().UnixNano())%len(positions)]
				if _, ok := sim.referees[randomPos]; ok {
					sim.CloseReferee(randomPos)
				} else {
					sim.ReconnectReferee(randomPos)
				}
			}
		}
	}()

	// Meanwhile, try to submit decisions
	for i := 0; i < 10; i++ {
		for _, pos := range positions {
			if _, ok := sim.referees[pos]; ok {
				sim.SendMessageAs(pos, map[string]interface{}{
					"action":   "submitDecision",
					"decision": "good",
				})
			}
		}
		time.Sleep(time.Second)
	}

	// Wait for chaos to end
	chaosWg.Wait()

	// Reconnect all referees for clean shutdown
	for _, pos := range positions {
		if _, ok := sim.referees[pos]; !ok {
			sim.ReconnectReferee(pos)
		}
	}
}

// Additional scenarios and suggestions
/*
1. Test broadcast with network latency:
   - Implement a delayed WebSocket dialer that adds artificial latency
   - Verify messages still arrive in correct order

2. Test sudden process termination and graceful recovery:
   - Trigger activity tracker shutdown
   - Restart server and verify state is recovered properly

3. Test increasing broadcast channel buffer size impact:
   - Measure message delivery with different buffer sizes
   - Find optimal buffer size for different load patterns

4. Test multi-day event scenario:
   - Simulate day 1 activity, sleep, day 2 activity
   - Verify system maintains state correctly across "days"

5. Test resilience to bad/malformed messages:
   - Send intentionally malformed JSON
   - Verify system handles errors gracefully
*/

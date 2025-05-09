// tests/remote_simulation_test.go
//go:build remote

package test

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"go-ref-lights/logger"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const (
	// Update these values with the actual remote server details
	remoteAppURL   = "https://referee-lights.michaelkingston.com.au"
	remoteWsURL    = "wss://referee-lights.michaelkingston.com.au/referee-updates"
	remoteMeetName = "test_mule" // Change this to the meet you want to test
	remoteUsername = "test"      // Change this to your test account username
	remotePassword = "mule"      // Change this to your test account password

	// Full meet simulation parameters
	numLifters          = 100              // Number of lifters in the meet
	attemptsPerLifter   = 9                // 3 squat, 3 bench, 3 deadlift
	avgTimeBetweenLifts = 20 * time.Second // Average time between lifters
)

// LiftType represents a type of lift in powerlifting
type LiftType int

const (
	Squat LiftType = iota
	Bench
	Deadlift
)

func (lt LiftType) String() string {
	return [...]string{"Squat", "Bench", "Deadlift"}[lt]
}

// Lifter represents a lifter in the competition
type Lifter struct {
	ID        int
	Name      string
	Attempts  [3]map[LiftType]bool // 3 attempts for each lift type, true=good, false=bad
	Completed bool
}

// RemoteSimulationTest provides a test harness for remote system tests
type RemoteSimulationTest struct {
	t               *testing.T
	appURL          string
	wsURL           string
	meetName        string
	httpClient      *http.Client
	director        *websocket.Conn
	directorCapture *MessageCapture
	referees        map[string]*LaunchClient
	lifters         []*Lifter
	mu              sync.Mutex // For thread-safe operations
}

// NewRemoteSimulationTest creates a new test harness for remote testing
func NewRemoteSimulationTest(t *testing.T, meetName string) *RemoteSimulationTest {
	// Create HTTP client with cookie jar
	jar, err := cookiejar.New(nil)
	require.NoError(t, err, "Failed to create cookie jar")

	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	wsURL := fmt.Sprintf("%s?meetName=%s", remoteWsURL, meetName)

	sim := &RemoteSimulationTest{
		t:               t,
		appURL:          remoteAppURL,
		wsURL:           wsURL,
		meetName:        meetName,
		httpClient:      client,
		directorCapture: &MessageCapture{},
		referees:        make(map[string]*LaunchClient),
	}

	return sim
}

// LoginAsDirector logs in as a meet director
func (s *RemoteSimulationTest) LoginAsDirector() {
	// First set the meet name
	formData := url.Values{}
	formData.Set("meetName", s.meetName)
	resp, err := s.httpClient.PostForm(s.appURL+"/set-meet", formData)
	require.NoError(s.t, err, "Failed to set meet name")
	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Failed to set meet name")
	resp.Body.Close()

	// Then login
	loginForm := url.Values{}
	loginForm.Set("username", remoteUsername)
	loginForm.Set("password", remotePassword)
	resp, err = s.httpClient.PostForm(s.appURL+"/login", loginForm)
	require.NoError(s.t, err, "Failed to login")
	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Login failed")
	resp.Body.Close()

	// Test if we can access the index page
	resp, err = s.httpClient.Get(s.appURL + "/index")
	require.NoError(s.t, err, "Failed to access index")
	require.Equal(s.t, http.StatusOK, resp.StatusCode, "Failed to access index after login")
	resp.Body.Close()

	// Set up director websocket
	s.setupDirector()
}

// SetupDirector sets up the meet director websocket connection
func (s *RemoteSimulationTest) setupDirector() {
	dialer := websocket.DefaultDialer

	// If you need to include cookies in the WebSocket handshake
	jar := s.httpClient.Jar
	cookieURL, err := url.Parse(s.appURL)
	require.NoError(s.t, err, "Failed to parse URL")
	cookies := jar.Cookies(cookieURL)

	header := http.Header{}
	for _, cookie := range cookies {
		header.Add("Cookie", cookie.Name+"="+cookie.Value)
	}

	conn, _, err := dialer.Dial(s.wsURL, header)
	require.NoError(s.t, err, "Failed to connect director websocket")
	s.director = conn

	// Start message capture goroutine
	go func() {
		for {
			_, msg, err := s.director.ReadMessage()
			if err != nil {
				s.t.Logf("Director websocket read error: %v", err)
				return
			}
			s.directorCapture.Add(msg)
		}
	}()
}

// CreateReferee creates a new referee client with the given position
func (s *RemoteSimulationTest) CreateReferee(position string) *LaunchClient {
	header := http.Header{}
	conn, _, err := websocket.DefaultDialer.Dial(s.wsURL, header)
	require.NoError(s.t, err, "Failed to connect referee websocket")

	ref := &LaunchClient{
		ID:       position,
		Position: position,
		Conn:     conn,
	}

	s.referees[position] = ref

	// Register referee identity
	err = conn.WriteJSON(map[string]string{
		"action":   "registerRef",
		"judgeId":  position,
		"meetName": s.meetName,
	})
	require.NoError(s.t, err, "Failed to register referee")

	return ref
}

// SendMessageAs sends a message as a specific referee
func (s *RemoteSimulationTest) SendMessageAs(position string, message map[string]interface{}) {
	s.mu.Lock()
	ref, ok := s.referees[position]
	s.mu.Unlock()

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

// ForceVacate forcibly vacates a position using the director's admin privileges
func (s *RemoteSimulationTest) ForceVacate(position string) {
	formData := url.Values{}
	formData.Set("meetName", s.meetName)
	formData.Set("position", position)

	resp, err := s.httpClient.PostForm(s.appURL+"/force-vacate", formData)
	require.NoError(s.t, err, "Failed to force vacate position")
	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Force vacate failed")
	resp.Body.Close()

	// Wait for occupancy changed message
	ok := s.WaitForMessages("occupancyChanged", 1)
	require.True(s.t, ok, "Did not receive occupancy changed message after force vacate")
}

// ResetInstance resets the meet instance
func (s *RemoteSimulationTest) ResetInstance() {
	formData := url.Values{}
	formData.Set("meetName", s.meetName)

	resp, err := s.httpClient.PostForm(s.appURL+"/reset-instance", formData)
	require.NoError(s.t, err, "Failed to reset meet instance")
	require.Equal(s.t, http.StatusFound, resp.StatusCode, "Reset instance failed")
	resp.Body.Close()

	// Wait for occupancy changed message showing all empty positions
	ok := s.WaitForMessages("occupancyChanged", 1)
	require.True(s.t, ok, "Did not receive occupancy changed message after reset instance")
}

// WaitForMessages waits for a specific number of messages with the given action
func (s *RemoteSimulationTest) WaitForMessages(action string, count int) bool {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if s.directorCapture.Count(action) >= count {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// CloseReferee closes a referee connection to simulate disconnection
func (s *RemoteSimulationTest) CloseReferee(position string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref, ok := s.referees[position]
	require.True(s.t, ok, "Referee %s not found", position)

	err := ref.Conn.Close()
	require.NoError(s.t, err, "Failed to close referee connection")
	delete(s.referees, position)

	// Give the server time to detect the closed connection
	time.Sleep(time.Second)
}

// ReconnectReferee reconnects a referee after it was closed
func (s *RemoteSimulationTest) ReconnectReferee(position string) {
	s.mu.Lock()
	_, exists := s.referees[position]
	s.mu.Unlock()

	require.False(s.t, exists, "Referee %s already connected", position)

	s.CreateReferee(position)

	// Give the server time to process the reconnection
	time.Sleep(time.Second)
}

// SimulateFullMeet creates a random set of lifters and simulates a complete meet
func (s *RemoteSimulationTest) SimulateFullMeet() {
	// Create lifters
	s.createLifters(numLifters)

	// Run the meet
	s.t.Logf("Starting full meet simulation with %d lifters", numLifters)
	s.t.Logf("Each lifter will perform %d attempts (%d total lifts)", attemptsPerLifter, numLifters*attemptsPerLifter)
	s.t.Logf("Estimated duration: %v", time.Duration(numLifters*attemptsPerLifter)*avgTimeBetweenLifts)
	startTime := time.Now()

	// Create referees
	positions := []string{"left", "center", "right"}
	for _, pos := range positions {
		s.CreateReferee(pos)
		s.t.Logf("Created referee for position: %s", pos)
	}

	// Wait for all referees to be registered
	ok := s.WaitForMessages("occupancyChanged", 3)
	require.True(s.t, ok, "Not all referees were registered")

	// Process each lifter
	completedLifts := 0
	totalLifts := numLifters * attemptsPerLifter
	logFrequency := 10 // Log every 10 lifts

	// Log the start of the meet with a separator
	s.t.Logf("=================== MEET STARTED ===================")

	for i, lifter := range s.lifters {
		// Log start of lifter if it's a multiple of 5
		if i%5 == 0 {
			s.t.Logf("Starting lifter %d/%d: %s", i+1, numLifters, lifter.Name)
		}

		// Process each lift type
		for lt := Squat; lt <= Deadlift; lt++ {
			// Three attempts per lift type
			for attempt := 0; attempt < 3; attempt++ {
				// Random variation in time between lifts (15-25 seconds)
				jitter := time.Duration(rand.Intn(10)-5) * time.Second
				waitTime := avgTimeBetweenLifts + jitter

				// Only log the wait time occasionally to reduce noise
				if completedLifts%logFrequency == 0 {
					s.t.Logf("Waiting %v before next lift", waitTime)
				}
				time.Sleep(waitTime)

				// Only log detailed lift info occasionally
				shouldLogDetailed := completedLifts%logFrequency == 0

				if shouldLogDetailed {
					s.t.Logf("Lifter %s attempting %s #%d", lifter.Name, lt, attempt+1)
				}

				// Start the timer (center referee's responsibility)
				s.SendMessageAs("center", map[string]interface{}{
					"action": "startTimer",
				})

				// Wait for timer to start
				ok := s.WaitForMessages("startTimer", 1)
				require.True(s.t, ok, "Timer did not start")

				// Wait for some timer updates
				ok = s.WaitForMessages("updatePlatformReadyTime", 3)
				require.True(s.t, ok, "Did not receive timer updates")

				// Random decision for each referee
				leftDecision := rand.Float32() < 0.7    // 70% chance of good lift
				centerDecision := rand.Float32() < 0.75 // 75% chance of good lift
				rightDecision := rand.Float32() < 0.65  // 65% chance of good lift

				// Majority rule
				goodLift := (leftDecision && centerDecision) ||
					(leftDecision && rightDecision) ||
					(centerDecision && rightDecision)

				// Submit decisions
				s.SendMessageAs("left", map[string]interface{}{
					"action":   "submitDecision",
					"decision": boolToDecision(leftDecision),
				})

				s.SendMessageAs("center", map[string]interface{}{
					"action":   "submitDecision",
					"decision": boolToDecision(centerDecision),
				})

				s.SendMessageAs("right", map[string]interface{}{
					"action":   "submitDecision",
					"decision": boolToDecision(rightDecision),
				})

				// Wait for decisions to be processed
				ok = s.WaitForMessages("displayResults", 1)
				require.True(s.t, ok, "Results were not displayed")

				// Record the result
				lifter.Attempts[attempt][lt] = goodLift
				completedLifts++

				// Log progress periodically to show the test is still running
				if completedLifts%logFrequency == 0 || completedLifts == totalLifts {
					progress := float64(completedLifts) / float64(totalLifts) * 100
					elapsed := time.Since(startTime)
					estTotal := time.Duration(float64(elapsed) * float64(totalLifts) / float64(completedLifts))
					remaining := estTotal - elapsed

					s.t.Logf("Progress: %.1f%% (%d/%d lifts) - %s complete, ~%s remaining",
						progress, completedLifts, totalLifts,
						elapsed.Round(time.Second), remaining.Round(time.Second))
				}

				// Every 10 lifts, simulate a referee disconnection and reconnection
				if completedLifts%10 == 0 {
					// Randomly select a position to disconnect
					pos := positions[rand.Intn(len(positions))]
					s.t.Logf("Simulating network issue: disconnecting %s referee", pos)

					s.CloseReferee(pos)
					time.Sleep(2 * time.Second)
					s.ReconnectReferee(pos)

					s.t.Logf("Reconnected %s referee", pos)
				}

				// Every 30 lifts, simulate a force vacate and reoccupy
				if completedLifts%30 == 0 {
					pos := positions[rand.Intn(len(positions))]
					s.t.Logf("Simulating force vacate for %s referee", pos)

					s.ForceVacate(pos)
					time.Sleep(time.Second)
					s.CreateReferee(pos)

					s.t.Logf("Reoccupied %s position", pos)
				}
			}
		}

		lifter.Completed = true

		// Log completion of lifter if it's a multiple of 5
		if (i+1)%5 == 0 {
			s.t.Logf("Completed lifter %d/%d: %s", i+1, numLifters, lifter.Name)
		}
	}

	totalTime := time.Since(startTime)
	s.t.Logf("=================== MEET COMPLETED ===================")
	s.t.Logf("Meet completed successfully in %s!", totalTime.Round(time.Second))
	s.t.Logf("Processed %d total lifts (%.1f seconds per lift average)",
		completedLifts, totalTime.Seconds()/float64(completedLifts))

	// Generate completion statistics
	goodLifts := 0
	for _, lifter := range s.lifters {
		for attempt := 0; attempt < 3; attempt++ {
			for lt := Squat; lt <= Deadlift; lt++ {
				if lifter.Attempts[attempt][lt] {
					goodLifts++
				}
			}
		}
	}

	successRate := float64(goodLifts) / float64(completedLifts) * 100
	s.t.Logf("Success rate: %.1f%% (%d good lifts out of %d total)",
		successRate, goodLifts, completedLifts)
}

// Helper function to convert boolean to decision string
func boolToDecision(good bool) string {
	if good {
		return "good"
	}
	return "bad"
}

// Create the specified number of lifters
func (s *RemoteSimulationTest) createLifters(count int) {
	s.lifters = make([]*Lifter, count)

	for i := 0; i < count; i++ {
		lifter := &Lifter{
			ID:   i + 1,
			Name: fmt.Sprintf("Lifter-%d", i+1),
		}

		// Initialize attempts
		for j := 0; j < 3; j++ {
			lifter.Attempts[j] = make(map[LiftType]bool)
		}

		s.lifters[i] = lifter
	}
}

// Close shuts down all connections
func (s *RemoteSimulationTest) Close() {
	// Close all referee connections
	s.mu.Lock()
	for _, ref := range s.referees {
		if ref.Conn != nil {
			ref.Conn.Close()
		}
	}
	s.mu.Unlock()

	// Close director connection
	if s.director != nil {
		s.director.Close()
	}
}

// TestFullMeetSimulation simulates a complete powerlifting meet
func TestFullMeetSimulation(t *testing.T) {
	// Skip if not explicitly running remote tests
	if testing.Short() {
		t.Skip("Skipping remote full meet simulation in short mode")
	}

	// Configure logging for long-running test
	configureTestLogging()

	// Set random seed
	rand.Seed(time.Now().UnixNano())

	// Create remote test harness
	sim := NewRemoteSimulationTest(t, remoteMeetName)
	defer sim.Close()

	// Login as director
	sim.LoginAsDirector()

	// First reset the instance to clear any previous state
	sim.ResetInstance()

	// Run the full meet simulation
	sim.SimulateFullMeet()
}

// configureTestLogging sets up appropriate logging for long-running tests
func configureTestLogging() {
	// Create a dedicated log file for this test run
	logFile, err := os.OpenFile(
		fmt.Sprintf("full_meet_simulation_%s.log", time.Now().Format("2006-01-02_15-04-05")),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0o644,
	)
	if err == nil {
		// Redirect test output to both stdout and the log file
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	// Add exclusions for common messages to avoid log spam
	logger.AddExcludePattern("updatePlatformReadyTime") // Filter out timer updates
	logger.AddExcludePattern("health")                  // Filter out health checks
	logger.AddExcludePattern("ping")                    // Filter out WebSocket pings
	logger.AddExcludePattern("heartbeat")               // Filter out heartbeat messages

	// Set log level to INFO to reduce DEBUG noise
	os.Setenv("LOG_LEVEL", "info")
}

// TestRefereeNetworkIssues tests how the system handles referee disconnections and reconnections
func TestRefereeNetworkIssues(t *testing.T) {
	// Skip if not explicitly running remote tests
	if testing.Short() {
		t.Skip("Skipping remote network issues test in short mode")
	}

	// Create remote test harness
	sim := NewRemoteSimulationTest(t, remoteMeetName)
	defer sim.Close()

	// Login as director
	sim.LoginAsDirector()

	// Reset the instance
	sim.ResetInstance()

	// Create initial referees
	positions := []string{"left", "center", "right"}
	for _, pos := range positions {
		sim.CreateReferee(pos)
	}

	// Wait for all referees to be registered
	ok := sim.WaitForMessages("occupancyChanged", 3)
	require.True(t, ok, "Not all referees were registered")

	// Test rapid disconnections and reconnections
	t.Log("Testing rapid disconnections and reconnections")

	for i := 0; i < 5; i++ {
		// Disconnect all referees
		for _, pos := range positions {
			sim.CloseReferee(pos)
		}

		// Wait a moment
		time.Sleep(time.Second)

		// Reconnect all referees
		for _, pos := range positions {
			sim.ReconnectReferee(pos)
		}

		// Wait for health update showing 3 referees
		ok = sim.WaitForMessages("refereeHealth", 1)
		require.True(t, ok, "Did not receive referee health update")
	}

	// Submit some decisions to verify functionality still works
	sim.SendMessageAs("center", map[string]interface{}{
		"action": "startTimer",
	})

	ok = sim.WaitForMessages("startTimer", 1)
	require.True(t, ok, "Timer did not start")

	// Submit decisions from all referees
	for _, pos := range positions {
		sim.SendMessageAs(pos, map[string]interface{}{
			"action":   "submitDecision",
			"decision": "good",
		})
	}

	// Verify results are displayed
	ok = sim.WaitForMessages("displayResults", 1)
	require.True(t, ok, "Results were not displayed")

	t.Log("System successfully handled rapid disconnections and reconnections")
}

// TestHighLoad tests system performance under high load
func TestHighLoad(t *testing.T) {
	// Skip if not explicitly running remote tests
	if testing.Short() {
		t.Skip("Skipping remote high load test in short mode")
	}

	// Create remote test harness
	sim := NewRemoteSimulationTest(t, remoteMeetName)
	defer sim.Close()

	// Login as director
	sim.LoginAsDirector()

	// Reset the instance
	sim.ResetInstance()

	// Create many simultaneous connections (30 is probably safe for production)
	numClients := 30
	clients := make([]*websocket.Conn, numClients)

	t.Logf("Creating %d simultaneous WebSocket connections", numClients)

	for i := 0; i < numClients; i++ {
		// Connect as anonymous clients
		conn, _, err := websocket.DefaultDialer.Dial(sim.wsURL, nil)
		require.NoError(t, err, "Failed to connect client %d", i)
		clients[i] = conn

		// Register with a unique ID
		err = conn.WriteJSON(map[string]string{
			"action":   "registerRef",
			"judgeId":  fmt.Sprintf("load-test-%d", i),
			"meetName": sim.meetName,
		})
		require.NoError(t, err, "Failed to register client %d", i)
	}

	// Create the actual referees
	positions := []string{"left", "center", "right"}
	for _, pos := range positions {
		sim.CreateReferee(pos)
	}

	// Have the center referee start a timer
	sim.SendMessageAs("center", map[string]interface{}{
		"action": "startTimer",
	})

	// Verify timer broadcast received by director
	ok := sim.WaitForMessages("startTimer", 1)
	require.True(t, ok, "Timer broadcast not received")

	// Submit decisions
	for _, pos := range positions {
		sim.SendMessageAs(pos, map[string]interface{}{
			"action":   "submitDecision",
			"decision": "good",
		})
	}

	// Verify results displayed
	ok = sim.WaitForMessages("displayResults", 1)
	require.True(t, ok, "Results were not displayed")

	// Cleanup all connections
	for i, conn := range clients {
		if conn != nil {
			err := conn.Close()
			if err != nil {
				t.Logf("Warning: Error closing client %d: %v", i, err)
			}
		}
	}

	t.Log("High load test completed successfully")
}

// Run remote tests with:
// go test -v -tags=remote -run=TestFullMeetSimulation ./tests

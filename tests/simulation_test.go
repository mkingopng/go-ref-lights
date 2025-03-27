//go:build e2e
// +build e2e

// tests/simulation_test.go
package tests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestEndToEndSimulation_Parallel_Production is an example "end‐to‐end"
// test that tries to open WebSockets for left/center/right referees
// without forcing TLS in your local/CI environment.
func TestEndToEndSimulation_Parallel_Production(t *testing.T) {
	t.Parallel() // If you want parallel test
	wsURL := getLocalOrProductionWSURL()

	// Try to connect “left”
	leftConn, err := dialReferee(wsURL, "left")
	if err != nil {
		t.Errorf("WS dial failed for left: %v", err)
	}
	defer func(leftConn *websocket.Conn) {
		err := leftConn.Close()
		if err != nil {

		}
	}(leftConn)

	// Connect “center”
	centerConn, err := dialReferee(wsURL, "center")
	if err != nil {
		t.Errorf("WS dial failed for center: %v", err)
	}
	defer func(centerConn *websocket.Conn) {
		err := centerConn.Close()
		if err != nil {

		}
	}(centerConn)

	// Connect “right”
	rightConn, err := dialReferee(wsURL, "right")
	if err != nil {
		t.Errorf("WS dial failed for right: %v", err)
	}
	defer func(rightConn *websocket.Conn) {
		err := rightConn.Close()
		if err != nil {

		}
	}(rightConn)

	// Example: do an HTTP request to check a route’s status code
	code, err := doHTTPGet("http://localhost:8080/someRoute")
	if err != nil {
		t.Fatalf("Failed GET /someRoute: %v", err)
	}
	// Suppose we expected a 302 redirect, ensure it's not 404
	if code != 200 {
		t.Errorf("Expected 302, got %d", code)
	}
}

// getLocalOrProductionWSURL decides whether to use ws:// or wss://
func getLocalOrProductionWSURL() string {
	env := os.Getenv("ENV")
	if env == "production" {
		// Real production address
		return "wss://referee-lights.michaelkingston.com.au/referee-updates"
	}
	// Local dev or test environment:
	return "ws://localhost:8080/referee-updates"
}

// dialReferee is a helper that appends e.g. ?position=left to the base URL
// and dials the WebSocket with a short handshake timeout.
func dialReferee(baseWSURL, position string) (*websocket.Conn, error) {
	fullURL := fmt.Sprintf("%s?position=%s", baseWSURL, position)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 5 * time.Second

	c, _, err := dialer.Dial(fullURL, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// doHTTPGet is a tiny helper to GET an endpoint, returning just the status code.
func doHTTPGet(url string) (int, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)
	return resp.StatusCode, nil
}

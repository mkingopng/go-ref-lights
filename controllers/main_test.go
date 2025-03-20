// file: controllers/main_test.go
package controllers

import (
	"go-ref-lights/websocket"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// If needed, init test environment
	websocket.InitTest()
	go websocket.HandleMessages() // start only once

	code := m.Run()
	os.Exit(code)
}

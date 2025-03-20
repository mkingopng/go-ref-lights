// file: controllers/main_test.go
package controllers

import (
	"go-ref-lights/websocket"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	websocket.InitTest()
	code := m.Run()
	os.Exit(code)
}

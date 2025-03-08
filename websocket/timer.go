// Package websocket - websocket/timer.go
package websocket

import (
	"time"
)

// platformReadyTimer represents a timer for the next attempt
var platformReadyTimer *time.Timer

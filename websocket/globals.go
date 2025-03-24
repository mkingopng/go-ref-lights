// Package websocket - websocket/globals.go
package websocket

import (
	"sync"
)

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// track an incrementing ID so each new timer gets a unique ID
var nextAttemptIDCounter int //nolint:unused

// global mutex to synchronise writes
var writeMutex sync.Mutex //nolint:unused

// mutexes for concurrency around timers
var (
	platformReadyMutex = &sync.Mutex{} //nolint:unused
	nextAttemptMutex   = &sync.Mutex{} //nolint:unused
)

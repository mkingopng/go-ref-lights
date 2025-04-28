// Package websocket - websocket/globals.go
package websocket

import (
	"sync"
)

// broadcast is a buffered channel for sending messages to all clients
// Buffer size of 500 absorbs short spikes of traffic without blocking writers
var broadcast = make(chan []byte, 500)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// global mutex to synchronise writes
var writeMutex sync.Mutex //nolint:unused

// mutexes for concurrency around timers
var (
	platformReadyMutex = &sync.Mutex{} //nolint:unused
	nextAttemptMutex   = &sync.Mutex{} //nolint:unused
)

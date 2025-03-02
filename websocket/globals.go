// Package websocket - websocket/globals.go
package websocket

import (
	"sync"
)

// clients track of all connected clients (for broadcast usage)
//var clients = make(map[*websocket.Conn]bool)

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// track an incrementing ID so each new timer gets a unique ID
var nextAttemptIDCounter int

// global mutex to synchronise writes
var writeMutex sync.Mutex

// mutexes for concurrency around timers
var (
	platformReadyMutex = &sync.Mutex{}
	nextAttemptMutex   = &sync.Mutex{}
)

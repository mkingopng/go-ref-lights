// Package websocket - websocket/globals.go
package websocket

// broadcast is a channel for sending messages to all clients
var broadcast = make(chan []byte)

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

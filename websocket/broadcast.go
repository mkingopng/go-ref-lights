// Package websocket handles real-time WebSocket communication between referees and the meet system.
// file: websocket/broadcast.go
package websocket

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-xray-sdk-go/xray"
	"time"

	"go-ref-lights/logger"
)

// allow tests to override the sleep behaviour.
var sleepFunc = time.Sleep

// StartNextAttemptTimer is an exported wrapper that triggers the next attempt timer for the given meet.
func StartNextAttemptTimer(meetState *MeetState) {
	if defaultTimerManager == nil {
		logger.Error.Println("[StartNextAttemptTimer] defaultTimerManager is nil!")
		return
	}
	defaultTimerManager.startNextAttemptTimer(meetState)
}

// HandleMessages listens for messages on the broadcast channel and distributes them to connections.
func HandleMessages() {
	// no request context here, so start from background
	rootCtx := context.Background()

	for {
		// start a short subsegment for each broadcast iteration
		_, bcSeg := xray.BeginSubsegment(rootCtx, "HandleBroadcast")

		msg := <-broadcast // read incoming message
		var msgMap map[string]interface{}
		var meetFilter string

		if err := json.Unmarshal(msg, &msgMap); err == nil {
			if m, ok := msgMap["meetName"].(string); ok {
				meetFilter = m
				err := bcSeg.AddAnnotation("meetFilter", meetFilter)
				if err != nil {
					return
				}
			}
		} else {
			// if JSON parse fails, note it
			err := bcSeg.AddAnnotation("unmarshalError", err.Error())
			if err != nil {
				return
			}
		}

		// optionally note msg size
		err := bcSeg.AddAnnotation("msgLength", len(msg))
		if err != nil {
			return
		}

		// acquire lock, broadcast to each connection
		connectionsMu.RLock()
		for c := range connections {
			if meetFilter != "" && c.meetName != meetFilter {
				continue
			}
			select {
			case c.send <- msg:
				// message queued
			default:
				logger.Warn.Printf("[HandleMessages] Dropping broadcast msg for %v", c.conn.RemoteAddr())
				err := bcSeg.AddAnnotation("droppedMsg", c.conn.RemoteAddr().String())
				if err != nil {
					return
				}
			}
		}
		connectionsMu.RUnlock()

		// end the subsegment
		bcSeg.Close(nil)
	}
}

// BroadcastMessage sends a message to all WebSocket clients associated with the given meet.
func BroadcastMessage(meetName string, message map[string]interface{}) {
	logger.Debug.Printf("[BroadcastMessage] Broadcasting next attempt timers for meet=%s", meetName)

	// convert message to JSON
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error.Printf("[BroadcastMessage] Error marshalling message: %v", err)
		return
	}

	// send the marshalled message to the broadcast channel
	broadcast <- msg
}

// broadcastFinalResults sends the final decisions to all connections in a meet
func broadcastFinalResults(meetName string) {
	meetState := DefaultStateProvider.GetMeetState(meetName) // fetch the current meet state

	// prepare the decision submission message
	submission := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centerDecision": meetState.JudgeDecisions["center"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}

	// convert submission to JSON
	resultMsg, err := json.Marshal(submission)
	if err != nil {
		logger.Error.Printf("[broadcastFinalResults] Error marshalling final results message: %v", err)
		return
	}
	logger.Info.Printf("[broadcastFinalResults] meet=%s -> 'displayResults' with Left=%s, center=%s, Right=%s",
		meetName, meetState.JudgeDecisions["left"], meetState.JudgeDecisions["center"], meetState.JudgeDecisions["right"])

	// broadcast the results to all clients
	broadcast <- resultMsg

	// start the next attempt timer
	StartNextAttemptTimer(meetState)

	// after a timeout, send a message to clear results
	go func() {
		sleepFunc(time.Duration(resultsDisplayDuration) * time.Second)
		// prepare a clear message
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, err := json.Marshal(clearMsg)
		if err != nil {
			logger.Error.Printf("[broadcastFinalResults] Error marshalling clearResults: %v", err)
			return
		}
		// send the clear message to the broadcast channel
		broadcast <- clearJSON
	}()
	// reset judge decisions for the next round
	meetState.JudgeDecisions = make(map[string]string)
}

// broadcastTimeUpdateWithIndex sends a time update message with an index to all clients in the meet.
//
//nolint:unu
//nolint:unused
func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int, meetName string) { //nolint:unused
	msg, err := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
		"meetName": meetName,
	})
	if err != nil {
		logger.Error.Printf("[broadcastTimeUpdateWithIndex] Error marshalling time update: %v", err)
		return
	}

	// send the time update message to the broadcast channel
	broadcast <- msg
}

// SendBroadcastMessage allows raw byte data to be sent over the broadcast channel
func SendBroadcastMessage(data []byte) {
	broadcast <- data
}

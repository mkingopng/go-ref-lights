// Package websocket: contains the WebSocket handler and related functions
package websocket

// GLOBALS

// GLOBAL BROADCAST LOOP

// MESSAGE READING & DISCONNECTION HANDLING

// HEARTBEAT (PING) KEEPS CONNECTIONS ALIVE

// DECISION & REFEREE HANDLING

// TIMER / PLATFORM READY

// not required per Daniel
//func isAllRefsConnected(meetState *MeetState) bool {
//	if meetState.RefereeSessions["left"] == nil {
//		return false
//	}
//	if meetState.RefereeSessions["centre"] == nil {
//		return false
//	}
//	if meetState.RefereeSessions["right"] == nil {
//		return false
//	}
//	return true
//}

// no longer required per Daniel
// stopPlatformReadyTimer stops the Platform Ready Timer
//func stopPlatformReadyTimer(meetState *MeetState) {
//	platformReadyMutex.Lock()
//	defer platformReadyMutex.Unlock()
//	meetState.PlatformReadyActive = false
//	meetState.PlatformReadyTimeLeft = 60
//}

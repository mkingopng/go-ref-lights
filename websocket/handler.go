// Package websocket: contains the WebSocket handler and related functions
// file: websocket/handler.go
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// GLOBALS

// resultsDisplayDuration controls how long final decisions remain displayed
var resultsDisplayDuration = 15

// platformReadyMutex, nextAttemptMutex for your timer logic
var (
	platformReadyMutex = &sync.Mutex{}
	nextAttemptMutex   = &sync.Mutex{}
)

// SINGLE-GOROUTINE MANAGER

// connectionInfo tracks which meet & judge belongs to each connection.
type connectionInfo struct {
	meetName string
	judgeID  string
}

// store all connected clients, plus a channel for broadcasting
var (
	clients           = make(map[*websocket.Conn]bool)
	connectionMapping = make(map[*websocket.Conn]connectionInfo)

	// manager channels
	register   = make(chan registerMsg)
	unregister = make(chan *websocket.Conn)
	broadcast  = make(chan []byte) // for raw message broadcasts
)

// registerMsg is used when a new connection arrives
type registerMsg struct {
	conn *websocket.Conn
	info connectionInfo
}

// the WebSocket Upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// If "Test-Mode", skip
		if r.Header.Get("Test-Mode") == "true" {
			return true
		}
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:8080" ||
			origin == "https://referee-lights.michaelkingston.com.au"
	},
}

// StartWebSocketManager runs in its own goroutine.
// handles registration, unregistration, and broadcast events.
func StartWebSocketManager() {
	log.Println("🟢 WebSocket Manager started. Listening for connections and messages.")
	for {
		select {
		case reg := <-register:
			kickOutIfNeeded(reg.info)
			clients[reg.conn] = true
			connectionMapping[reg.conn] = reg.info
			log.Printf("✅ Referee %s registered (meet: %s). Total connections: %d",
				reg.info.judgeID, reg.info.meetName, len(clients))
			meetState := getMeetState(reg.info.meetName)
			broadcastRefereeHealth(meetState)

		case conn := <-unregister:
			if _, ok := clients[conn]; ok {
				delete(clients, conn)
			}
			if info, ok := connectionMapping[conn]; ok {
				meetState := getMeetState(info.meetName)
				if meetState.RefereeSessions[info.judgeID] == conn {
					meetState.RefereeSessions[info.judgeID] = nil
					log.Printf("🚪 Removing %s from meet %s (disconnect). Active referees: %d",
						info.judgeID, info.meetName, len(meetState.RefereeSessions))
					broadcastRefereeHealth(meetState)
				}
				delete(connectionMapping, conn)
			}
			log.Printf("🔴 Connection closed. Remaining active connections: %d", len(clients))
		case msg := <-broadcast:
			log.Printf("📢 Broadcasting message to %d clients", len(clients))
			for conn := range clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					log.Printf("⚠️ WriteMessage error: %v", err)
					_ = conn.Close()
					unregister <- conn
				}
			}
		}
	}
}

// Kick out older connection if the same meet/judge is already connected
func kickOutIfNeeded(newInfo connectionInfo) {
	for conn, info := range connectionMapping {
		if info.meetName == newInfo.meetName && info.judgeID == newInfo.judgeID {
			log.Printf("🔴 Kicking out old session for ref: %s in meet: %s", info.judgeID, info.meetName)
			_ = conn.Close()
			unregister <- conn
			log.Printf("🔄 Active connections after removal: %d", len(clients)-1)
		}
	}
}

// ServeWs is our main entry point for new WebSocket connections
func ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade error: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusBadRequest)
		return
	}
	log.Printf("✅ WebSocket connected: %v", conn.RemoteAddr())

	meetName := r.URL.Query().Get("meetName")
	if meetName == "" {
		meetName = "DEFAULT_MEET"
	}
	judgeID := r.URL.Query().Get("judgeId")

	log.Printf("📡 New WebSocket connection - Meet: %s, Judge: %s, Total Clients: %d",
		meetName, judgeID, len(clients)+1)

	register <- registerMsg{
		conn: conn,
		info: connectionInfo{
			meetName: meetName,
			judgeID:  judgeID,
		},
	}

	go startHeartbeat(conn)
	go handleReads(conn, meetName, judgeID)
}

// handleReads processes messages from a given connection
func handleReads(conn *websocket.Conn, defaultMeetName, defaultJudgeID string) {
	defer func() {
		log.Printf("⚠️ WebSocket disconnected: %v. Active connections: %d", conn.RemoteAddr(), len(clients)-1)
		_ = conn.Close()
		unregister <- conn
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("⚠️ WebSocket read error from %v: %v", conn.RemoteAddr(), err)
			return
		}

		var dMsg DecisionMessage
		if err := json.Unmarshal(msg, &dMsg); err != nil {
			log.Printf("⚠️ Invalid JSON received from %v: %v", conn.RemoteAddr(), err)
			continue
		}

		if dMsg.MeetName == "" {
			dMsg.MeetName = defaultMeetName
		}
		if dMsg.JudgeID == "" && defaultJudgeID != "" {
			dMsg.JudgeID = defaultJudgeID
		}

		log.Printf("📩 Received action: %s (Judge: %s, Meet: %s)", dMsg.Action, dMsg.JudgeID, dMsg.MeetName)

		switch dMsg.Action {
		case "registerRef":
			registerRef(dMsg, conn)
		case "startTimer", "stopTimer", "resetTimer", "startNextAttemptTimer":
			handleTimerAction(dMsg.Action, dMsg.MeetName)
		default:
			processDecision(dMsg, conn)
		}
	}
}

// startHeartbeat sends ping every 10s to keep connection alive
func startHeartbeat(conn *websocket.Conn) {
	log.Printf("🔄 Starting heartbeat for: %v", conn.RemoteAddr())
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	failedPings := 0
	for range ticker.C {
		if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			failedPings++
			log.Printf("⚠️ WebSocket ping failed (%d/3) for %v: %v", failedPings, conn.RemoteAddr(), err)
			if failedPings >= 3 {
				log.Printf("❌ Closing connection due to repeated ping failures: %v", conn.RemoteAddr())
				_ = conn.Close()
				unregister <- conn
				return
			}
		} else {
			failedPings = 0
		}
	}
}

// registerRef is existing logic for referee registration
func registerRef(msg DecisionMessage, conn *websocket.Conn) {
	if msg.JudgeID == "" {
		log.Println("registerRef received with empty judgeID; ignoring registration.")
		return
	}
	meetState := getMeetState(msg.MeetName)

	// single-session enforcement
	existingConn, exists := meetState.RefereeSessions[msg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		log.Printf("🔴 Kicking out old session for ref: %s in meet: %s", msg.JudgeID, msg.MeetName)
		_ = existingConn.Close()
		unregister <- existingConn // tell manager to remove it
	}
	meetState.RefereeSessions[msg.JudgeID] = conn
	log.Printf("✅ Referee %s registered via registerRef (meet: %s)", msg.JudgeID, msg.MeetName)
	broadcastRefereeHealth(meetState)
}

// processDecision is your existing logic for judge decisions
func processDecision(decisionMsg DecisionMessage, conn *websocket.Conn) {
	meetState := getMeetState(decisionMsg.MeetName)

	// basic validation
	if decisionMsg.JudgeID == "" || decisionMsg.Decision == "" {
		return
	}

	// single-session enforcement
	existingConn, exists := meetState.RefereeSessions[decisionMsg.JudgeID]
	if exists && existingConn != nil && existingConn != conn {
		log.Printf("🔴 Kicking out old session for referee: %s in meet: %s",
			decisionMsg.JudgeID, decisionMsg.MeetName)
		_ = existingConn.Close()
		unregister <- existingConn
		meetState.RefereeSessions[decisionMsg.JudgeID] = nil
	}

	// store new decision
	meetState.RefereeSessions[decisionMsg.JudgeID] = conn
	meetState.JudgeDecisions[decisionMsg.JudgeID] = decisionMsg.Decision

	log.Printf("✅ Decision from %s: %s (meet: %s)",
		decisionMsg.JudgeID, decisionMsg.Decision, decisionMsg.MeetName)

	// let everyone know a judge submitted
	submission := map[string]string{
		"action":  "judgeSubmitted",
		"judgeId": decisionMsg.JudgeID,
	}
	subMsg, _ := json.Marshal(submission)
	broadcast <- subMsg

	// also broadcast updated health
	broadcastRefereeHealth(meetState)

	// if all 3 have responded:
	if len(meetState.JudgeDecisions) == 3 {
		broadcastFinalResults(decisionMsg.MeetName)
	}
}

// broadcastFinalResults remains the same
func broadcastFinalResults(meetName string) {
	meetState := getMeetState(meetName)

	// 1) broadcast final decisions
	result := map[string]string{
		"action":         "displayResults",
		"leftDecision":   meetState.JudgeDecisions["left"],
		"centreDecision": meetState.JudgeDecisions["centre"],
		"rightDecision":  meetState.JudgeDecisions["right"],
	}
	resultMsg, _ := json.Marshal(result)
	broadcast <- resultMsg

	// 2) start next-lifter timer
	startNextAttemptTimer(meetState)

	// 3) clear results after set duration
	go func() {
		time.Sleep(time.Duration(resultsDisplayDuration) * time.Second)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
	}()

	// 4) reset for next lift
	meetState.JudgeDecisions = make(map[string]string)
}

// timer logic (Platform Ready, Next Attempt, etc.)

// handleTimerAction processes timer-related actions
func handleTimerAction(action, meetName string) {
	meetState := getMeetState(meetName)
	switch action {
	case "startTimer":
		if !isAllRefsConnected(meetState) {
			errMsg := map[string]string{
				"action":  "healthError",
				"message": "Cannot start timer: Not all referees are connected!",
			}
			msg, _ := json.Marshal(errMsg)
			broadcast <- msg
			log.Println("❌ Timer not started: All referees not connected.")
			return
		}
		// clear old decisions
		meetState.JudgeDecisions = make(map[string]string)
		// force lights to clear
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON
		startPlatformReadyTimer(meetState)

	case "stopTimer":
		stopPlatformReadyTimer(meetState)

	case "resetTimer":
		resetPlatformReadyTimer(meetState)
		meetState.JudgeDecisions = make(map[string]string)
		clearMsg := map[string]string{"action": "clearResults"}
		clearJSON, _ := json.Marshal(clearMsg)
		broadcast <- clearJSON

	case "startNextAttemptTimer":
		startNextAttemptTimer(meetState)
	}
	log.Printf("✅ Timer action processed: %s (meet: %s)", action, meetName)
}

func isAllRefsConnected(meetState *MeetState) bool {
	return meetState.RefereeSessions["left"] != nil &&
		meetState.RefereeSessions["centre"] != nil &&
		meetState.RefereeSessions["right"] != nil
}

// platformReady, next attempt logic remains
func startPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()
	if meetState.PlatformReadyActive {
		log.Println("⚠️ Platform Ready Timer already running.")
		return
	}
	meetState.PlatformReadyActive = true
	meetState.PlatformReadyTimeLeft = 60
	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			platformReadyMutex.Lock()
			if !meetState.PlatformReadyActive {
				platformReadyMutex.Unlock()
				return
			}
			meetState.PlatformReadyTimeLeft--
			broadcastTimeUpdateWithIndex("updatePlatformReadyTime", meetState.PlatformReadyTimeLeft, 0)
			if meetState.PlatformReadyTimeLeft <= 0 {
				broadcast <- []byte(`{"action":"platformReadyExpired"}`)
				meetState.PlatformReadyActive = false
				meetState.PlatformReadyTimeLeft = 60
				platformReadyMutex.Unlock()
				return
			}
			platformReadyMutex.Unlock()
		}
	}()
}

func stopPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

func resetPlatformReadyTimer(meetState *MeetState) {
	platformReadyMutex.Lock()
	defer platformReadyMutex.Unlock()

	if !meetState.PlatformReadyActive {
		log.Println("⚠️ No active timer to reset.")
		return
	}
	meetState.PlatformReadyActive = false
	meetState.PlatformReadyTimeLeft = 60
}

// next attempts
func startNextAttemptTimer(meetState *MeetState) {
	nextAttemptMutex.Lock()
	defer nextAttemptMutex.Unlock()

	newTimer := NextAttemptTimer{
		TimeLeft: 60,
		Active:   true,
	}
	meetState.NextAttemptTimers = append(meetState.NextAttemptTimers, newTimer)
	idx := len(meetState.NextAttemptTimers) - 1

	ticker := time.NewTicker(time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			nextAttemptMutex.Lock()
			if !meetState.NextAttemptTimers[idx].Active {
				nextAttemptMutex.Unlock()
				return
			}
			meetState.NextAttemptTimers[idx].TimeLeft--
			broadcastTimeUpdateWithIndex("updateNextAttemptTime",
				meetState.NextAttemptTimers[idx].TimeLeft,
				idx,
			)
			if meetState.NextAttemptTimers[idx].TimeLeft <= 0 {
				broadcast <- []byte(`{"action":"nextAttemptExpired"}`)
				meetState.NextAttemptTimers[idx].Active = false
				nextAttemptMutex.Unlock()
				return
			}
			nextAttemptMutex.Unlock()
		}
	}()
}

// broadcastRefereeHealth, broadcastTimeUpdateWithIndex remain the same
func broadcastRefereeHealth(meetState *MeetState) {
	var connectedIDs []string
	for judgeID, c := range meetState.RefereeSessions {
		if c != nil {
			connectedIDs = append(connectedIDs, judgeID)
		}
	}
	msg := map[string]interface{}{
		"action":            "refereeHealth",
		"connectedRefIDs":   connectedIDs,
		"connectedReferees": len(connectedIDs),
		"requiredReferees":  3,
	}
	out, _ := json.Marshal(msg)
	broadcast <- out
}

func broadcastTimeUpdateWithIndex(action string, timeLeft int, index int) {
	msg, _ := json.Marshal(map[string]interface{}{
		"action":   action,
		"timeLeft": timeLeft,
		"index":    index,
	})
	broadcast <- msg
}

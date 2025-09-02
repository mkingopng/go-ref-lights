# Design Document

## Overview

The RefLights system is experiencing a critical concurrency bug where referee decisions from one competition are displaying on the lights screen of other concurrent competitions. After investigating the codebase, I've identified the root cause: several functions in the WebSocket broadcast system are not including the `meetName` field in their messages, causing them to be broadcast to all connections regardless of which meet they belong to.

## Root Cause Analysis

The WebSocket system uses a filtering mechanism in `HandleMessages()` that checks for a `meetName` field in incoming messages:

```go
if err := json.Unmarshal(msg, &msgMap); err == nil {
    if m, ok := msgMap["meetName"].(string); ok {
        meetFilter = m
    }
}

// Later in the loop:
for c := range connections {
    if meetFilter != "" && c.meetName != meetFilter {
        continue  // Skip connections not in this meet
    }
    // Send message to connection
}
```

However, several critical functions are sending messages without the `meetName` field:

1. **`broadcastFinalResults()`** - Sends `displayResults` and `clearResults` messages without `meetName`
2. **`BroadcastMessage()`** - Takes `meetName` parameter but doesn't include it in the message
3. **`realMessenger.BroadcastMessage()`** - Same issue as above

When `meetFilter` is empty (because `meetName` is missing), the filtering logic fails and messages are sent to ALL connections.

## Architecture

### Current Message Flow
```
Decision Made → broadcastFinalResults() → broadcast channel → HandleMessages() → ALL connections (BUG)
```

### Fixed Message Flow
```
Decision Made → broadcastFinalResults() → broadcast channel (with meetName) → HandleMessages() → Filtered connections
```

## Components and Interfaces

### 1. WebSocket Broadcast System (`websocket/broadcast.go`)

**Functions to Fix:**
- `broadcastFinalResults(meetName string)` - Add `meetName` to both `displayResults` and `clearResults` messages
- `BroadcastMessage(meetName string, message map[string]interface{})` - Add `meetName` to message before marshalling

### 2. Messenger Interface (`websocket/messenger.go`)

**Functions to Fix:**
- `realMessenger.BroadcastMessage()` - Add `meetName` to message before marshalling

### 3. Message Structure Standards

All broadcast messages must include:
```go
{
    "action": "...",
    "meetName": "specific-meet-name",
    // ... other fields
}
```

## Data Models

### Message Format
```go
type BroadcastMessage struct {
    Action   string `json:"action"`
    MeetName string `json:"meetName"`  // REQUIRED for filtering
    // Additional fields vary by message type
}
```

### Examples of Fixed Messages

**Display Results Message:**
```go
{
    "action": "displayResults",
    "meetName": "APL Test Meet",
    "leftDecision": "good",
    "centerDecision": "good",
    "rightDecision": "no"
}
```

**Clear Results Message:**
```go
{
    "action": "clearResults",
    "meetName": "APL Test Meet"
}
```

**Timer Messages:**
```go
{
    "action": "startTimer",
    "meetName": "APL Test Meet"
}
```

## Error Handling

### Current Issues
- Messages without `meetName` bypass filtering and reach all connections
- No validation that `meetName` is present in broadcast messages
- Silent failures when filtering doesn't work

### Proposed Solutions
1. **Validation**: Add checks to ensure `meetName` is present before broadcasting
2. **Logging**: Enhanced logging to track which meet receives each message
3. **Testing**: Comprehensive tests for meet isolation

## Testing Strategy

### Unit Tests
1. **Message Filtering Tests**
   - Verify messages with `meetName` only reach correct connections
   - Verify messages without `meetName` are rejected or logged as errors
   - Test edge cases with empty or invalid `meetName`

2. **Broadcast Function Tests**
   - Test `broadcastFinalResults()` includes `meetName` in both messages
   - Test `BroadcastMessage()` adds `meetName` to message
   - Test `realMessenger.BroadcastMessage()` includes `meetName`

### Integration Tests
1. **Multi-Meet Isolation Tests**
   - Create two concurrent meets with different names
   - Have referees make decisions in each meet
   - Verify decisions only appear on correct meet's lights display
   - Test WebSocket connection filtering

2. **Concurrency Tests**
   - Simulate high-load scenarios with multiple meets
   - Verify no cross-contamination under stress
   - Test connection management during concurrent operations

### End-to-End Tests
1. **Full Decision Flow Tests**
   - Complete referee decision workflow for multiple concurrent meets
   - Verify complete isolation from decision to display
   - Test timer isolation between meets

## Implementation Plan

### Phase 1: Fix Core Broadcast Functions
1. Fix `broadcastFinalResults()` to include `meetName` in both messages
2. Fix `BroadcastMessage()` to add `meetName` to message parameter
3. Fix `realMessenger.BroadcastMessage()` to include `meetName`

### Phase 2: Add Validation and Logging
1. Add validation to ensure `meetName` is present in broadcast messages
2. Enhance logging to track message routing by meet
3. Add error handling for missing `meetName`

### Phase 3: Comprehensive Testing
1. Update existing tests to verify `meetName` inclusion
2. Add new multi-meet isolation tests
3. Add concurrency and stress tests

### Phase 4: Verification and Monitoring
1. Deploy to staging environment
2. Test with multiple concurrent meets
3. Monitor logs for proper message filtering
4. Verify no cross-contamination in production

## Backward Compatibility

The fixes are backward compatible because:
- Adding `meetName` to messages doesn't break existing clients
- The filtering logic already expects `meetName` and handles its absence gracefully
- No changes to public APIs or interfaces

## Performance Considerations

- Minimal performance impact from adding `meetName` field to messages
- Improved performance from proper filtering (fewer unnecessary message deliveries)
- No changes to connection management or WebSocket handling

## Security Implications

- Fixes a security issue where meet isolation was compromised
- Prevents accidental information leakage between competitions
- Maintains proper access control boundaries between meets

# Implementation Plan

- [x] 1. Fix broadcastFinalResults function to include meetName in messages
  - Modify the `displayResults` message to include `meetName` field
  - Modify the `clearResults` message to include `meetName` field
  - Update function in `websocket/broadcast.go`
  - _Requirements: 1.1, 1.2, 1.3_

- [x] 2. Fix BroadcastMessage function to include meetName in message parameter
  - Add `meetName` to the message map before marshalling in `websocket/broadcast.go`
  - Ensure the meetName parameter is properly included in the broadcast message
  - _Requirements: 1.1, 1.2, 2.1, 2.2_

- [x] 3. Fix realMessenger.BroadcastMessage to include meetName in message
  - Modify `realMessenger.BroadcastMessage()` in `websocket/messenger.go` to add meetName to message
  - Ensure meetName parameter is included in the message before marshalling
  - _Requirements: 1.1, 1.2, 2.1, 2.2_

- [x] 4. Add validation for meetName in broadcast functions
  - Add checks to ensure meetName is not empty before broadcasting
  - Add error logging when meetName is missing or invalid
  - Update functions in `websocket/broadcast.go` and `websocket/messenger.go`
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 5. Update existing unit tests to verify meetName inclusion
  - Modify `TestBroadcastFinalResults` to verify meetName is in displayResults message
  - Modify `TestBroadcastFinalResults_ClearsAfterTimeout` to verify meetName is in clearResults message
  - Update tests in `websocket/broadcast_test.go`
  - _Requirements: 4.1, 4.2_

- [x] 6. Create multi-meet isolation integration tests
  - Write test that creates two concurrent meets with different names
  - Simulate referee decisions in each meet independently
  - Verify decisions only reach connections for the correct meet
  - Create new test file `websocket/isolation_test.go`
  - _Requirements: 4.1, 4.3, 5.1, 5.2_

- [x] 7. Add concurrency stress tests for meet isolation
  - Create test with multiple concurrent meets making simultaneous decisions
  - Verify no cross-contamination under high load
  - Test connection filtering under concurrent operations
  - Add tests to `websocket/isolation_test.go`
  - _Requirements: 4.4, 5.3, 5.4_

- [x] 8. Update messenger unit tests to verify meetName handling
  - Modify `TestRealMessenger_BroadcastMessage` to verify meetName inclusion
  - Add test cases for missing or empty meetName
  - Update tests in `websocket/messenger_test.go`
  - _Requirements: 4.1, 4.2_

- [x] 9. Add enhanced logging for message routing verification
  - Add debug logs showing which meet each message is routed to
  - Log when messages are filtered out due to meetName mismatch
  - Update logging in `websocket/broadcast.go` HandleMessages function
  - _Requirements: 3.4_

- [x] 10. Create end-to-end test for complete decision workflow isolation
  - Test full referee decision flow from submission to lights display
  - Verify complete isolation between concurrent meets
  - Test timer isolation and decision clearing between meets
  - Add comprehensive test to verify all requirements are met
  - _Requirements: 1.4, 2.3, 2.4, 5.1, 5.2, 5.3, 5.4_

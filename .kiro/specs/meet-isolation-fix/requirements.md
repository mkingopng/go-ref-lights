# Requirements Document

## Introduction

RefLights is experiencing a critical concurrency bug where referee decisions from one competition are displaying on the lights screen of other concurrent competitions. This violates the fundamental isolation principle that different meets should be completely independent. The system is built with Go's concurrency features (goroutines, channels, mutexes) to handle multiple meets simultaneously, but there's clearly a flaw in the isolation mechanism that allows cross-contamination of referee decisions between meets.

## Requirements

### Requirement 1

**User Story:** As a meet director running competition A, I want referee decisions from my competition to only display on my lights screen, so that my competition remains isolated from other concurrent competitions.

#### Acceptance Criteria

1. WHEN a referee makes a decision in competition A THEN the decision SHALL only appear on competition A's lights display
2. WHEN a referee makes a decision in competition A THEN the decision SHALL NOT appear on any other competition's lights display
3. WHEN multiple competitions are running concurrently THEN each competition SHALL maintain complete decision isolation
4. WHEN all three referees in competition A make decisions THEN only competition A's lights SHALL display the results

### Requirement 2

**User Story:** As a referee logged into competition B, I want my decisions to only affect competition B's state, so that I don't accidentally influence other competitions.

#### Acceptance Criteria

1. WHEN I make a decision as a referee in competition B THEN my decision SHALL only be recorded for competition B
2. WHEN I make a decision as a referee in competition B THEN my decision SHALL NOT affect any other competition's state
3. WHEN I am connected to competition B THEN my WebSocket connection SHALL only receive updates relevant to competition B
4. WHEN I submit a decision THEN the system SHALL correctly identify which competition I belong to

### Requirement 3

**User Story:** As a system administrator, I want to identify the root cause of the cross-competition decision bleeding, so that I can implement a permanent fix.

#### Acceptance Criteria

1. WHEN investigating the WebSocket broadcast mechanism THEN the system SHALL properly scope broadcasts to specific meets
2. WHEN examining decision storage THEN the system SHALL properly isolate decisions by meet identifier
3. WHEN reviewing session management THEN the system SHALL correctly associate referee sessions with specific meets
4. WHEN analyzing the lights display logic THEN the system SHALL only show decisions from the correct meet

### Requirement 4

**User Story:** As a developer, I want comprehensive testing for meet isolation, so that this type of concurrency bug cannot occur again.

#### Acceptance Criteria

1. WHEN running concurrent meet tests THEN the system SHALL demonstrate complete decision isolation
2. WHEN testing WebSocket broadcasts THEN messages SHALL only reach clients connected to the relevant meet
3. WHEN testing referee decision flow THEN decisions SHALL be properly scoped to the correct meet
4. WHEN testing edge cases THEN the system SHALL maintain isolation under various concurrent scenarios

### Requirement 5

**User Story:** As a meet organizer, I want confidence that my competition will not be affected by technical issues from other concurrent competitions, so that I can focus on running a smooth event.

#### Acceptance Criteria

1. WHEN other competitions experience issues THEN my competition SHALL continue operating normally
2. WHEN multiple competitions are running THEN each SHALL have independent state management
3. WHEN referees connect or disconnect from other meets THEN my meet's state SHALL remain unaffected
4. WHEN decisions are made in other competitions THEN my lights display SHALL show only my competition's decisions

# RefLights Product Overview

RefLights is a real-time referee lighting system for powerlifting competitions built in Go for speed and concurrency. It enables multiple meets to run concurrently and completely independently, providing fast, reliable decision-making for lifters, referees, and meet organizers.

## Core Workflow
1. **Meet Director Setup**: Logs in via web browser and generates QR codes for each referee position
2. **Referee Check-in**: Referees scan QR codes to occupy positions (Left, Center, Right) - only one referee per position
3. **Platform Ready**: Head/center referee starts 60-second countdown for lifter to take platform
4. **Decision Making**: All 3 referees make "good lift" or "no lift" decisions on mobile devices
5. **Results Display**: Lights show decisions only after all referees have decided
6. **Next Attempt**: 60-second timer for lifter to submit next attempt (multiple timers can run concurrently)
7. **Reset**: Platform ready button clears previous decisions and starts new cycle

## Key Design Principles
- **Ephemeral Decisions**: No accumulation of decisions in memory - cleared on each platform ready
- **Heartbeat Connections**: Maintains referee connections even when using phone for other purposes
- **Position Exclusivity**: Only one referee can occupy each position at a time
- **Fast & Reliable**: Built for speed to avoid distracting lifters or disrupting meets
- **Easy to Use**: Simple interface for referees and meet organizers

## Core Features
- Multi-meet support with complete independence between competitions
- Real-time referee decisions via WebSocket communication
- QR code-based position assignment system
- Dual timer system: Platform ready (60s) and Next attempt (60s, multiple concurrent)
- Visual feedback indicators (green dots) for referee decision status
- Automatic session management with heartbeat monitoring
- Position vacate/claim functionality

## Target Users
- **Lifters**: Want fast, reliable system that doesn't distract from their performance
- **Meet Organizers**: Need flawless, easy-to-use system that works every time
- **Referees**: Require simple, fast, reliable decision-making interface
- **Audience**: Expect smooth meets without technical disruptions

## Production Environment
- Deployed on AWS at https://referee-lights.michaelkingston.com.au
- Uses HTTPS with proper security headers
- CloudWatch logging and monitoring
- Built for high availability and concurrent meet support

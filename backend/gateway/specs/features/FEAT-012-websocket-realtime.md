---
id: FEAT-012
title: WebSocket Realtime Streaming
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P2
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.1 Dashboard (Realtime)"
---

## Mục Tiêu

WebSocket endpoint cho realtime streaming:
- Engine health status changes
- Memory flow events (ingest/recall/forget)
- Pipeline job progress
- Alert notifications

## Scope

### In Scope
- `WS /v1/console/ws` — Authenticated WebSocket connection
- Channels: `engine.health`, `memory.flow`, `pipeline.progress`, `alerts`

### Out of Scope
- Live session replay (future scope)

## Thiết Kế Kỹ Thuật

### WebSocket Protocol

**Connection:** `ws://gateway:8080/v1/console/ws?token=<jwt>`

**Subscribe:**
```json
{ "action": "subscribe", "channels": ["engine.health", "memory.flow"] }
```

**Message format:**
```json
{
  "channel": "engine.health",
  "event": "status_changed",
  "data": {
    "engine": "openviking",
    "old_status": "healthy",
    "new_status": "degraded",
    "reason": "high_latency"
  },
  "timestamp": "2026-05-13T12:00:00Z"
}
```

### Internal Architecture
- **Handler:** `adapter/ws/handler.go` (extend existing)
- **Source:** Subscribe to NATS JetStream subjects for events
- **Fan-out:** Per-connection goroutine with channel filtering

## Acceptance Criteria
- [ ] AC-1: WebSocket connection requires valid JWT
- [ ] AC-2: Subscribe/unsubscribe to channels dynamically
- [ ] AC-3: Engine health changes propagated within 2s
- [ ] AC-4: Memory flow events include engine source badge
- [ ] AC-5: Connection gracefully handles disconnects/reconnects

## Test Requirements
- Unit tests: Channel filtering, message serialization
- Integration tests: WebSocket client with mock NATS
- Minimum coverage: 80%

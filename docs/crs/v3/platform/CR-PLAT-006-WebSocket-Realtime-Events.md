# Change Request: CR-PLAT-006 — WebSocket Real-time Event Streaming

**CR ID:** CR-PLAT-006
**Component:** `backend/gateway`, `backend/apps/memory`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Platform
**Feature:** [F28](../../../features/28-websocket-realtime-events/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-04 | Agent Developer | Console không real-time → phải refresh |
| PP-P5-02 | IDE Plugin User | IDE plugin không nhận live updates |

**Before:** Console phải poll APIs every 5s để detect changes.
**After:** WebSocket connection → events pushed instantly to Console.

---

## 2. Event Types

```json
// health_change
{"event": "health_change", "data": {"service": "cognee-ingestion", "status": "unhealthy"}, "timestamp": "..."}

// memory_stored
{"event": "memory_stored", "data": {"engine": "graphiti", "type": "episodic", "memory_id": "..."}, "timestamp": "..."}

// memory_forgotten
{"event": "memory_forgotten", "data": {"user_id": "...", "engines": ["graphiti", "cognee"]}, "timestamp": "..."}

// session_end
{"event": "session_end", "data": {"session_id": "...", "hook_count": 47}, "timestamp": "..."}

// rate_limit_exceeded
{"event": "rate_limit_exceeded", "data": {"endpoint": "/v1/memory/store", "limit": 200}, "timestamp": "..."}

// observe_event (from F08)
{"event": "observe_event", "data": {"session_id": "...", "hook_type": "llm_call", "observation_id": "..."}, "timestamp": "..."}

// pipeline_complete
{"event": "pipeline_complete", "data": {"session_id": "...", "tiers": [1,2,3]}, "timestamp": "..."}
```

---

## 3. Connection Flow

```
Client → GET /v1/console/ws?token=JWT (WebSocket upgrade)
  Auth: validate JWT, extract tenant_id
  ↓
Server sends: {"event": "connected", "data": {"tenant_id": "..."}}
  ↓
Server subscribes to NATS subjects:
  events.{tenant_id}.>   (all events for this tenant)
  ↓
On NATS event → forward to all WS connections for that tenant
  ↓
Client disconnects → server unsubscribes
```

---

## 4. Reconnection & Event Buffer

```
Client-side: exponential backoff reconnect (1s, 2s, 4s, ..., max 30s)
Server-side event buffer: last 100 events per tenant (Redis)
On reconnect: client sends {"last_event_id": "..."} → server replays missed events
```

---

## 5. Tenant Scoping (Security)

```
CRITICAL: WebSocket stream ONLY delivers events for the authenticated tenant.
NATS subject: events.{tenant_id}.> (scoped by tenant)
No cross-tenant event leakage possible at protocol level.
```

---

## 6. API

| Method | Path | Description |
|---|---|---|
| `GET` | `/v1/console/ws` | WebSocket upgrade (JWT in query param) |
| `GET` | `/v1/console/events` | SSE fallback (for environments without WS) |

---

## 7. Acceptance Criteria

- [ ] WebSocket upgrade succeeds with valid JWT
- [ ] Invalid JWT → close connection with 401 code
- [ ] Memory stored event delivered within 500ms of NATS event
- [ ] Health change event delivered within 200ms
- [ ] Tenant isolation: zero cross-tenant event delivery
- [ ] Event buffer: client can request last 100 events on reconnect
- [ ] SSE fallback endpoint functional

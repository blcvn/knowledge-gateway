# Feature 26 — Session Replay

> **Loại:** AgentMemory | **Priority:** Medium | **Status:** Implemented (CR-AM-005)

## Mô tả

Session Replay cho phép replay lại một agent session từ đầu đến cuối — như "video playback" nhưng cho AI Agent operations. Developer có thể scrub timeline, filter events theo type, control playback speed, và import external session transcripts.

---

## Business Logic

### Timeline Builder

Từ raw observations (Feature 08), Session Replay xây dựng timeline có structure:
- Sắp xếp events theo timestamp
- Group events theo conversation turns
- Identify phase boundaries (planning → execution → reflection)
- Attach metadata (duration, error status, memory impact)

### Playback Controls

- **Play/Pause**: Control playback
- **Speed Control**: 0.5× / 1× / 2× / 4× playback speed
- **Scrubbing**: Click bất kỳ điểm nào trên timeline
- **Jump to**: Jump đến event types cụ thể (e.g., "jump to next error")

### Event Filtering

Filter events by type:
- LLM calls only
- Tool calls only
- Memory operations only
- Errors only
- Decisions only

### Real-time Live View

Ngoài replay historical sessions, có thể xem live session đang chạy:
- SSE stream: `GET /v1/observe/stream` (Feature 08)
- Events được pushed real-time lên UI
- Auto-scroll as new events arrive

### JSONL Transcript Import

Import external session transcripts (e.g., từ Claude Code transcript files):
- JSONL format: one event per line
- Auto-parse hook types
- Create virtual session for replay

---

## Dataflow

### Session Replay Flow

```
GET /v1/console/sessions/{id}/timeline
        │
        ▼
observe-service (replay usecase)
        │
        ├── Load all observations for session (agent_raw_observations)
        ├── Build timeline:
        │         ├── Sort by timestamp
        │         ├── Group by conversation turn
        │         └── Annotate with phase, duration, impact
        │
        └── Return structured timeline


Console UI
        │
        ├── Render timeline as interactive scrubber
        ├── User clicks play → simulate event sequence
        │         └── For each event (at playback speed):
        │                   └── Display event detail in UI
        │
        └── Speed control adjusts setTimeout interval
```

### Live Stream View

```
Console UI → GET /v1/observe/stream  (SSE connection)
        │
        └── Keep SSE connection open

observe-service (SSE handler)
        │
        ├── Subscribe to: observe.event.captured (NATS)
        └── For each event:
                  └── SSE push: "data: {event_json}\n\n"
                            │
                            └── Console UI renders event in real-time
```

### JSONL Import

```
Console UI uploads JSONL file
        │
        ├── POST /v1/observe/sessions/import
        │         ├── Input: JSONL file (one event per line)
        │         └── Parse: detect hook types, timestamps
        │
        └── Create virtual session in DB
                  └── Render replay UI as if it were a real session
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/sessions/{id}/timeline` | Session replay timeline |
| `GET` | `/v1/observe/stream` | Live SSE stream |
| (conceptual) | `/v1/observe/sessions/import` | JSONL import |

---

## Related Features

- Feature 08 (Agent Observe) — produces observations that replay uses
- Feature 21 (Sessions Explorer) — lists sessions available for replay
- Feature 28 (WebSocket) — alternative real-time transport

---

## Business Value

### Pain Points được giải quyết

- **PP-P1-07 (Cannot reproduce bugs)**
- **PP-P4-04 (Cannot trace AI decision)**

### Actors hưởng lợi

P1 AI Agent Developer, P3 ML/AI Engineer, P4 Enterprise Architect

### Giải pháp tham chiếu

- [S7 — Agent Observability](../../bussiness/solutions/S7-agent-observability.md)

### ROI / Kết quả đo được

> Bug reproduction: impossible → 1 click | Claude Code JSONL import | Timeline scrubbing

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*

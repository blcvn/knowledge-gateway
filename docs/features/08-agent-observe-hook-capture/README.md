# Feature 08 — Agent Observe & Hook Capture

> **Loại:** AgentMemory | **Priority:** P0 | **Status:** Implemented (CR-AM-001)

## Mô tả

Observe Service là một **hook capture pipeline** cho AI Agent — thu thập tất cả hoạt động của agent (tool calls, LLM prompts/responses, errors, decisions) và xây dựng session timeline có cấu trúc. Đây là foundation cho Memory Lifecycle, Consolidation Pipeline, và Session Replay.

Không giống logging thông thường, Observe Service **understands** context: biết event nào thuộc session nào, event nào là duplicate, event nào chứa PII/secrets cần redact.

---

## Business Logic

### 12 Lifecycle Hooks

Agent có thể emit events theo 12 hook types:
1. `session_start` — Session bắt đầu
2. `session_end` — Session kết thúc
3. `llm_prompt` — LLM được gọi (capture prompt)
4. `llm_response` — LLM trả về (capture response)
5. `tool_call` — Agent gọi tool
6. `tool_response` — Tool trả về kết quả
7. `memory_read` — Agent đọc memory
8. `memory_write` — Agent ghi memory
9. `error` — Lỗi xảy ra
10. `decision` — Agent đưa ra quyết định
11. `observation` — Agent quan sát môi trường
12. `checkpoint` — Điểm checkpoint quan trọng

### 14-Step Observe Pipeline

Khi một observation được nhận, pipeline xử lý:

1. **Receive** raw observation event
2. **Validate** schema và required fields
3. **Authenticate** session/agent identity
4. **Deduplicate** (DedupMap, 30-second TTL) — loại bỏ events trùng lặp
5. **Privacy Redact** — tự động redact API keys (sk-, Bearer), JWT tokens, PII (email, phone, credit card), database URLs
6. **Parse Hook Type** — xác định trong 12 hook types
7. **Enrich** with session metadata
8. **Classify** — raw observation vs compressed observation
9. **Store Raw** — lưu vào `agent_raw_observations` table
10. **Index** — index cho BM25 search
11. **Embed** — generate vector embedding
12. **Publish NATS** — `observe.event.captured` event
13. **Update Session State** — cập nhật session metrics
14. **Stream SSE** — push real-time tới connected clients

### Session Management

Sessions có 3 trạng thái:
- `active` — Đang diễn ra
- `completed` — Kết thúc bình thường
- `abandoned` — Timeout hoặc bị hủy

### Privacy Redaction

Tự động detect và redact:
- API keys: `sk-*`, `Bearer *`, `AKIA*` (AWS)
- JWT tokens
- Email addresses
- Phone numbers
- Credit card numbers (Luhn check)
- Database URLs (credentials part)

---

## Dataflow

### Hook Capture Pipeline

```
AI Agent (hooks)
        │
        ├── session_start / end
        ├── llm_prompt / response
        ├── tool_call / response
        └── memory_read / write
        │
        ▼
POST /v1/observe/sessions/{id}/observe
        │
        ▼
observe-service (14-step pipeline)
        │
        ├── Step 1-3:  Validate + Auth
        ├── Step 4:    Deduplicate (DedupMap, 30s TTL)
        │                  └── Hash(event content) → skip if seen within 30s
        ├── Step 5:    Privacy Redact
        │                  ├── Regex: sk-[a-zA-Z0-9]{20,}   → "sk-[REDACTED]"
        │                  ├── Regex: Bearer [^\s]+          → "Bearer [REDACTED]"
        │                  ├── Regex: email patterns          → "[EMAIL REDACTED]"
        │                  └── Regex: credit card (Luhn)     → "[CC REDACTED]"
        ├── Step 6-8:  Parse + Enrich + Classify
        ├── Step 9:    Store → PostgreSQL (agent_raw_observations)
        ├── Step 10:   BM25 Index update
        ├── Step 11:   Embed → vector store
        ├── Step 12:   NATS publish: observe.event.captured
        ├── Step 13:   Update session state
        └── Step 14:   SSE stream push


GET /v1/observe/stream   ← SSE endpoint
        │
        └── Real-time stream of observations for monitoring
```

### Session Lifecycle

```
POST /v1/observe/sessions          → Create session
        │
        ├── Init session record: {id, agent_id, state: "active", started_at}
        └── Start session timer (timeout detection)
        │
POST /v1/observe/sessions/{id}/observe  → Capture observations (repeated)
        │
        └── (14-step pipeline above)
        │
POST /v1/observe/sessions/{id}/end      → End session
        │
        ├── Set state = "completed"
        ├── Calculate session metrics (obs count, duration, error rate)
        └── Trigger consolidation pipeline (NATS: consolidation.trigger)
```

---

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/v1/observe/sessions` | Start session |
| `POST` | `/v1/observe/sessions/{id}/observe` | Capture observation |
| `POST` | `/v1/observe/sessions/{id}/end` | End session |
| `GET` | `/v1/observe/sessions/{id}` | Get session detail |
| `GET` | `/v1/observe/sessions` | List sessions |
| `DELETE` | `/v1/observe/sessions/{id}` | Delete session |
| `GET` | `/v1/observe/sessions/{id}/observations` | Get all observations |
| `GET` | `/v1/observe/stream` | SSE real-time stream |

---

## Database Tables

| Table | Nội dung |
|-------|---------|
| `agent_sessions` | Session records, state, metrics |
| `agent_raw_observations` | Raw observation events (before compression) |
| `agent_compressed_observations` | After consolidation pipeline |

---

## Services

| Service | Vai trò |
|---------|---------|
| `observe-service` | 14-step pipeline, session management, SSE stream |
| `obs-service` | Alias / extended version for platform-level observability |

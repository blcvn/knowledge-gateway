# Change Request: CR-AM-001 — Observe Service (Hook Capture Pipeline)

**CR ID:** CR-AM-001  
**Component:** `services/observe-service` [NEW SERVICE]  
**Priority:** Critical  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.1, SRS FR-OBS-001..004, FR-SESSION-001..004  
**Spec:** `references/agentmemory/specs/services/observe-service/spec.md`

---

## 1. Mô tả

Xây dựng **Observe Service** — service nhận hook events từ AI coding agents (Claude Code, Cursor, Codex, v.v.) thông qua HTTP POST và thực thi **14-step pipeline**: validate → dedup → privacy → build → image → mutex → limit → agentId → persist → stream → session → compress → index.

Đây là service nền tảng cho toàn bộ memory capture flow của agentmemory.

---

## 2. Vấn đề hiện tại

`services/memory-service` trong VNP Memory hiện có `IngestUseCase` nhưng:
- Chỉ xử lý ingest từ API call thủ công (không có lifecycle hooks).
- Không có deduplication pipeline.
- Không có privacy redaction cho hook payloads.
- Không hỗ trợ 12 hook types (`session_start`, `prompt_submit`, `pre_tool_use`, `post_tool_use`, `post_tool_failure`, `pre_compact`, `subagent_start`, `subagent_stop`, `notification`, `task_completed`, `stop`, `session_end`).
- Không có per-session mutex để đảm bảo ordering.
- Không có SSE stream real-time cho Viewer.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/observe-service/`

**Port:** `8081` (gRPC), endpoint HTTP cho hook ingestion  
**Binary:** `cmd/observe/main.go`

**Cấu trúc thư mục:**
```
services/observe-service/
├── cmd/observe/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Session, RawObservation, CompressedObservation
│   │   ├── value_object.go     # HookType, ObservationType, Modality
│   │   └── errors.go
│   ├── observe/
│   │   ├── pipeline.go         # 14-step pipeline (main entry)
│   │   ├── dedup.go            # DedupMap (SHA256 hash, 30s TTL)
│   │   ├── synthetic.go        # Synthetic compression (zero LLM)
│   │   └── stream.go           # SSE StreamBroker
│   ├── usecase/
│   │   ├── observe.go          # ObserveUseCase — orchestrates pipeline
│   │   ├── session_start.go    # Explicit session management
│   │   ├── session_end.go      # Trigger summarization
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go       # KVStore, SearchIndexer, EventPublisher, StreamBroadcaster
│   └── adapter/
│       ├── http/
│       │   ├── handler.go      # POST /observe, /observe/session/*
│       │   └── middleware.go   # Auth (HMAC Bearer)
│       ├── repository/
│       │   └── sqlite/         # SQLite KV via iii-style storage
│       └── event/
│           └── publisher.go    # NATS: agentmemory.session.* events
```

### 3.2. Core Domain Models

```go
// internal/domain/entity.go

type HookType string
const (
    HookSessionStart    HookType = "session_start"
    HookPromptSubmit    HookType = "prompt_submit"
    HookPreToolUse      HookType = "pre_tool_use"
    HookPostToolUse     HookType = "post_tool_use"
    HookPostToolFailure HookType = "post_tool_failure"
    HookPreCompact      HookType = "pre_compact"
    HookSubagentStart   HookType = "subagent_start"
    HookSubagentStop    HookType = "subagent_stop"
    HookNotification    HookType = "notification"
    HookTaskCompleted   HookType = "task_completed"
    HookStop            HookType = "stop"
    HookSessionEnd      HookType = "session_end"
)

type ObservationType string
const (
    ObsFileRead    ObservationType = "file_read"
    ObsFileWrite   ObservationType = "file_write"
    ObsFileEdit    ObservationType = "file_edit"
    ObsCommandRun  ObservationType = "command_run"
    ObsSearch      ObservationType = "search"
    ObsWebFetch    ObservationType = "web_fetch"
    ObsConversation ObservationType = "conversation"
    ObsError       ObservationType = "error"
    ObsDecision    ObservationType = "decision"
    ObsDiscovery   ObservationType = "discovery"
    ObsSubagent    ObservationType = "subagent"
    ObsNotification ObservationType = "notification"
    ObsTask        ObservationType = "task"
    ObsImage       ObservationType = "image"
    ObsOther       ObservationType = "other"
)

type Session struct {
    ID               string
    Project          string
    CWD              string
    StartedAt        time.Time
    EndedAt          *time.Time
    Status           SessionStatus   // active | completed | abandoned
    ObservationCount int
    Model            string
    Tags             []string
    FirstPrompt      string
    Summary          string
    CommitSHAs       []string
    AgentID          string
}

type RawObservation struct {
    ID               string
    SessionID        string
    Timestamp        time.Time
    HookType         HookType
    ToolName         string
    ToolInput        any
    ToolOutput       any
    UserPrompt       string
    AssistantResponse string
    Raw              any
    Modality         string  // "text" | "image" | "mixed"
    ImageData        string  // base64
    AgentID          string
}

type CompressedObservation struct {
    ID         string
    SessionID  string
    Timestamp  time.Time
    Type       ObservationType
    Title      string
    Subtitle   string
    Facts      []string
    Narrative  string
    Concepts   []string
    Files      []string
    Importance float64
    Confidence float64
    ImageRef   string
    AgentID    string
}
```

### 3.3. 14-Step Pipeline

```go
// internal/observe/pipeline.go
// Step 1: Validate (sessionId, hookType, timestamp required)
// Step 2: Dedup (SHA256 hash of sessionId+toolName+toolInput, 30s TTL)
// Step 3: Privacy Redaction (strip API keys, tokens, PII patterns)
// Step 4: Build RawObservation
// Step 5: Image Detection (detect base64 in payload → set modality)
// Step 6: Per-session Keyed Mutex Lock (ordering guarantee)
// Step 7: Session Limit Check (MAX_OBS_PER_SESSION, default 500)
// Step 8: AgentID Inheritance (from session if not in payload)
// Step 9: KV Write (persist RawObservation)
// Step 10: Dedup Record (mark hash as seen)
// Step 11: SSE Stream Broadcast (raw_observation event)
// Step 12: Session Update (increment count, update lastActive)
// Step 13: Synthetic Compression (zero LLM, build CompressedObservation)
// Step 14: BM25 + Vector Indexing (schedule async save)
```

### 3.4. HTTP API

```
POST /observe                          # Main hook capture endpoint
POST /observe/session/start            # Explicit session create
POST /observe/session/end              # Session end + trigger summarize
GET  /sessions                         # List sessions
GET  /sessions/{id}                    # Session detail
GET  /sessions/{id}/observations       # Compressed observations list
GET  /sessions/{id}/observations/{oid} # Single observation
DELETE /sessions/{id}                  # Delete session + cascade
GET  /stream                           # SSE stream (real-time viewer)
GET  /health                           # Health check
```

### 3.5. Privacy Redaction Package

```go
// pkg/privacy/redact.go
// Patterns to redact before KV storage:
// - Bearer tokens: /Bearer\s+[A-Za-z0-9._-]+/
// - API Keys: /sk-[A-Za-z0-9]{20,}/
// - AWS keys: /AKIA[A-Z0-9]{16}/
// - Private keys: /-----BEGIN.*PRIVATE KEY-----/
// - JWT tokens: /eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/
// - .env values: /[A-Z_]+=["']?[A-Za-z0-9+/=]{20,}["']?/
// Replace with: [REDACTED]
```

### 3.6. NATS Events Published

| Subject | Payload |
|---|---|
| `agentmemory.session.started` | `{session_id, project, cwd, agent_id}` |
| `agentmemory.session.ended` | `{session_id, observation_count}` |
| `agentmemory.observation.captured` | `{observation_id, session_id, hook_type}` |

### 3.7. Gateway Integration

**[MODIFY]** `gateway/adapter/handler/router.go`

```
POST /v1/observe                       → observe-service:8081
POST /v1/observe/session/start         → observe-service:8081
POST /v1/observe/session/end           → observe-service:8081
GET  /v1/sessions                      → observe-service:8081
GET  /v1/sessions/{id}                 → observe-service:8081
GET  /v1/sessions/{id}/observations    → observe-service:8081
```

---

## 4. Environment Variables

| Variable | Default | Mô tả |
|---|---|---|
| `OBSERVE_PORT` | `8081` | Service HTTP port |
| `AGENTMEMORY_DATA_DIR` | `~/.agentmemory` | SQLite + index dir |
| `MAX_OBS_PER_SESSION` | `500` | Per-session limit |
| `AGENTMEMORY_AUTO_COMPRESS` | `false` | LLM compression opt-in |
| `DEDUP_TTL_SECONDS` | `30` | Dedup window |
| `INDEX_SAVE_INTERVAL` | `30s` | BM25/vector debounce save |
| `AGENTMEMORY_SECRET` | `""` | HMAC auth secret |

---

## 5. Acceptance Criteria

- [x] `POST /observe` với `hookType: "post_tool_use"` tạo `RawObservation` + `CompressedObservation` thành công.
- [x] Gửi 2 request giống nhau trong 30s: response thứ 2 trả `{"deduplicated": true}`.
- [x] Payload chứa `sk-abc123456789` được redact thành `[REDACTED]` trước khi lưu vào KV.
- [x] SSE stream `/stream` nhận được event `raw_observation` real-time khi observation được captured.
- [x] Session `observationCount` tăng đúng với mỗi observation.
- [x] Sau `POST /observe/session/end`, session status = `completed` và NATS event `agentmemory.session.ended` được publish.
- [x] `GET /sessions/{id}/observations` trả về compressed observations có đủ `title`, `facts`, `files`, `importance`.
- [x] Thực hiện 501 observations trong cùng session: request 501 bị từ chối với `session observation limit reached`.

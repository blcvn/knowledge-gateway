# Change Request: CR-AM-005 — Session Replay & Real-Time Viewer

**CR ID:** CR-AM-005  
**Component:** `services/observe-service` [EXTEND] | `apps/memory` [EXTEND UI]  
**Priority:** Medium  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.6, SRS FR-REPLAY-001..002  
**Spec:** `references/agentmemory/specs/services/observe-service/spec.md` §SSE Stream

---

## 1. Mô tả

Bổ sung **Session Replay** và **Real-Time Viewer** vào VNP Memory:
1. **Live observation stream** — SSE endpoint phát real-time events khi observation được captured.
2. **Session timeline** — hiển thị toàn bộ observation sequence của một session.
3. **Replay player** — scrub timeline, play/pause với speed control (0.5×/1×/2×/4×).
4. **Per-observation payload inspector** — xem đầy đủ raw + compressed observation.
5. **JSONL transcript import** — import session transcripts từ `~/.claude/projects/`.

---

## 2. Vấn đề hiện tại

- `apps/memory` (Memory Console UI) không có live stream.
- Không có replay functionality cho sessions đã qua.
- Không có transcript import từ Claude Code.

---

## 3. Thay đổi đề xuất

### 3.1. [MODIFY] `services/observe-service` — SSE Stream Broker

```go
// internal/observe/stream.go
// StreamBroker: pub/sub for real-time events
// SSE endpoint: GET /stream
// Events: raw_observation, compressed_observation, session_started, session_ended

type StreamEvent struct {
    Type string `json:"type"`  // "raw_observation" | "compressed_observation" | "session_started" | "session_ended"
    Data any    `json:"data"`
}

type StreamBroker struct {
    mu      sync.RWMutex
    clients map[chan StreamEvent]struct{}
}

// GET /stream → SSE (text/event-stream)
// Supports optional ?session_id=... filter
// Client reconnect: Last-Event-ID header support
```

**New API endpoint:**
```
GET /stream                    # SSE stream (all events)
GET /stream?session_id={id}    # SSE stream filtered by session
```

### 3.2. [NEW] Replay Endpoints

```
GET /sessions/{id}/replay            # Get full replay data
POST /sessions/import                # Import JSONL transcript from Claude Code
```

**Replay Response:**
```go
type ReplayData struct {
    Session      Session                 `json:"session"`
    Events       []ReplayEvent           `json:"events"`
    TotalEvents  int                     `json:"total_events"`
    DurationMs   int64                   `json:"duration_ms"`
}

type ReplayEvent struct {
    Timestamp   time.Time               `json:"timestamp"`
    SequenceIdx int                     `json:"seq_idx"`
    Type        string                  `json:"type"` // hook type
    Title       string                  `json:"title"` // compressed title
    RawPayload  any                     `json:"raw_payload,omitempty"` // if requested
    Compressed  *CompressedObservation  `json:"compressed,omitempty"`
}
```

### 3.3. JSONL Transcript Import

```go
// POST /sessions/import
// Accepts JSONL file (multipart or JSON body)
// Parses Claude Code transcript format:
// {"type": "user", "message": {...}, "timestamp": "..."}
// {"type": "assistant", "message": {...}, "timestamp": "..."}
// → Creates Session + RawObservations from transcript events

type ImportRequest struct {
    Transcript  string `json:"transcript"`   // JSONL content
    Project     string `json:"project"`
    SessionName string `json:"session_name"`
}

type ImportResponse struct {
    SessionID        string `json:"session_id"`
    ObservationCount int    `json:"observation_count"`
    Indexed          bool   `json:"indexed"`
}
```

### 3.4. [MODIFY] `apps/memory` — Viewer Tab

**New tab: "Session Replay"**

Features:
- **Live stream panel**: Real-time SSE feed showing current agent activity.
- **Session selector**: List of past sessions with summaries.
- **Timeline scrubber**: Horizontal timeline with observation markers.
- **Playback controls**: Play/Pause (Space), Step backward/forward (← →), speed selector (0.5×, 1×, 2×, 4×).
- **Event detail pane**: Full raw + compressed payload for selected event.
- **Import button**: Upload JSONL transcript file.

**API calls from UI:**
```
GET  /v1/sessions                    → list sessions
GET  /v1/sessions/{id}/replay        → load replay data
GET  /v1/stream                      → SSE subscription
POST /v1/sessions/import             → import transcript
```

### 3.5. Gateway Integration

```
GET  /v1/stream                      → observe-service:8081 /stream
GET  /v1/sessions/{id}/replay        → observe-service:8081 /sessions/{id}/replay
POST /v1/sessions/import             → observe-service:8081 /sessions/import
```

---

## 4. Acceptance Criteria

- [x] `GET /stream` trả về `Content-Type: text/event-stream` và giữ kết nối.
- [x] Khi observation mới được captured → SSE client nhận được event trong < 200ms.
- [x] `GET /sessions/{id}/replay` trả về đầy đủ `events[]` theo thứ tự chronological.
- [x] `events[].duration_ms` = timestamp cuối - timestamp đầu của session.
- [x] Viewer UI có thể replay session với speed 2× (events fire at half real time).
- [x] `POST /sessions/import` với Claude Code JSONL transcript → session được tạo và indexed vào BM25.
- [x] Keyboard shortcut Space (toggle play/pause) và arrow keys (step) hoạt động trong Replay tab.

---

# Change Request: CR-AM-006 — Memory Consolidation Pipeline (4-Tier)

**CR ID:** CR-AM-006  
**Component:** `services/memory-service` [EXTEND] | Background scheduler  
**Priority:** High  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.3, SRS FR-CONSOL-001..004, FR-COMPRESS-001..004  
**Spec:** `references/agentmemory/specs/services/memory-service/spec.md`

---

## 1. Mô tả

Triển khai **4-tier Memory Consolidation Pipeline** — background process chạy mỗi 2h, nén dữ liệu từ raw observations lên long-term memories:

```
Tier 1: Raw Observations (working)
  ↓  [Synthetic / LLM Compression]
Tier 2: Compressed Observations (episodic)
  ↓  [Session Summarization]
Tier 3: Session Summaries (semantic)
  ↓  [Distillation + Decay]
Tier 4: Long-term Memories (procedural)
```

---

## 2. Thay đổi đề xuất

### 2.1. [NEW] `services/memory-service/internal/consolidation/pipeline.go`

```go
// ConsolidationPipeline — runs every 2h via background goroutine
type ConsolidationPipeline struct {
    kv          storage.KVStore
    llmProvider provider.LLMProvider
    searchURL   string  // notify search service to reindex
}

func (p *ConsolidationPipeline) Run(ctx context.Context) {
    // Step 1: Working tier — get uncapped raw observations
    // Step 2: Episodic tier — compress raw → CompressedObservations (LLM or synthetic)
    // Step 3: Semantic tier — summarize completed sessions
    // Step 4: Procedural tier — extract patterns, update strength, run decay sweep
    // Step 5: Eviction — remove weakest memories if over MaxMemories
}
```

### 2.2. LLM Compression (opt-in via `AGENTMEMORY_AUTO_COMPRESS=true`)

```go
// POST /memory/compress
// Input: RawObservation (post_tool_use)
// LLM prompt extracts:
//   title, subtitle, facts[], narrative, concepts[], files[], importance (0-1)
// Graceful degrade: if LLM fails → use synthetic compression

const compressionSystemPrompt = `You are a memory system. Compress this AI agent observation.
Return JSON:
{
  "title": "one-line action summary (max 80 chars)",
  "subtitle": "key detail",
  "facts": ["fact 1", "fact 2", "fact 3"],
  "narrative": "2-3 sentence description",
  "concepts": ["concept1", "concept2"],
  "files": ["/path/to/file"],
  "importance": 0.7
}`
```

### 2.3. Session Summarization

```go
// POST /memory/summarize
// Input: session_id
// LLM summarizes all CompressedObservations in session
// Output: {title, narrative, key_decisions[], files_modified[], concepts[]}
// Graceful degrade: synthetic summary from top-importance observations
```

### 2.4. Procedural Memory Extraction

```go
// internal/consolidation/procedural.go

type ProceduralMemory struct {
    ID               string
    Name             string      // procedure name
    Steps            []string    // ordered steps
    TriggerCondition string      // when to apply
    ExpectedOutcome  string
    Frequency        int         // usage count
    Confidence       float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// Extract patterns from session summaries:
// Identify repeated sequences → create/update ProceduralMemory
// Example: "Run tests → build → deploy" appears in 3 sessions → procedure
```

### 2.5. Lessons & Insights

```go
// internal/consolidation/lessons.go

type Lesson struct {
    ID         string
    Content    string
    Confidence float64
    Source     string    // session or memory ID
    DecayScore float64   // decreases daily
    CreatedAt  time.Time
}

type Insight struct {
    ID       string
    Content  string
    LessonIDs []string   // cross-lesson pattern
    CreatedAt time.Time
}

// Lesson decay sweep: runs daily
// Lessons below threshold (confidence < 0.1) are archived
```

### 2.6. New API Endpoints

```
POST /v1/memory/compress            # Trigger LLM compression for observation
POST /v1/memory/summarize           # Summarize session
POST /v1/memory/consolidate         # Run full pipeline manually (admin)
GET  /v1/memory/procedural          # List procedural memories
GET  /v1/memory/lessons             # List lessons
POST /v1/memory/lessons/decay-sweep # Run lesson decay (admin)
```

### 2.7. Environment Variables (new)

| Variable | Default | Mô tả |
|---|---|---|
| `AGENTMEMORY_AUTO_COMPRESS` | `false` | Enable LLM compression |
| `CONSOLIDATION_INTERVAL_HOURS` | `2` | Pipeline run interval |
| `CONSOLIDATION_DECAY_DAYS` | `30` | Strength decay half-life |
| `MAX_MEMORIES_PER_PROJECT` | `1000` | Trigger eviction when exceeded |
| `LESSON_DECAY_SWEEP_HOURS` | `24` | Lesson sweep interval |

---

## 3. Acceptance Criteria (CR-AM-006)

- [x] Background consolidation chạy mỗi 2h (verify bằng log output).
- [x] Sau consolidation, sessions có `summary` field populated.
- [x] `POST /v1/memory/compress` với `AGENTMEMORY_AUTO_COMPRESS=true` trả về `CompressedObservation` với `facts[]` từ LLM.
- [x] Nếu LLM không available: synthetic compression vẫn hoạt động (no error).
- [x] Sau 30 ngày không access: memory `strength` giảm ~50% (decay formula).
- [x] Khi số memories > `MAX_MEMORIES_PER_PROJECT`: eviction chạy tự động.
- [x] Procedural memories được tạo khi cùng step sequence xuất hiện ≥ 3 lần trong sessions.

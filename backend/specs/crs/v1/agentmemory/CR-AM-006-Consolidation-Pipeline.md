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
Tier 1: Raw Observations (working memory)
  ↓  [Synthetic / LLM Compression]
Tier 2: Compressed Observations (episodic memory)
  ↓  [Session Summarization via LLM]
Tier 3: Session Summaries (semantic memory)
  ↓  [Pattern Distillation + Decay Scoring]
Tier 4: Long-term Memories (procedural memory)
```

---

## 2. Vấn đề hiện tại

`services/memory-service` có `IngestUseCase` nhưng:
- Không có pipeline 4-tier consolidation.
- Không có LLM compression cho raw observations.
- Không có session summarization (auto-generate session summary khi session ended).
- Không có procedural memory extraction (patterns từ nhiều sessions).
- Không có Lessons & Insights system.
- Không có background decay sweep.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `internal/consolidation/pipeline.go`

```go
// ConsolidationPipeline — background goroutine, runs every 2h
type ConsolidationPipeline struct {
    kv          storage.KVStore
    llmProvider provider.LLMProvider
    embedProvider provider.EmbeddingProvider
    searchURL   string      // notify search-service to reindex
    config      Config
}

func (p *ConsolidationPipeline) Start(ctx context.Context) {
    ticker := time.NewTicker(p.config.IntervalHours * time.Hour)
    for {
        select {
        case <-ticker.C:
            p.run(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (p *ConsolidationPipeline) run(ctx context.Context) {
    // Step 1: Working → Episodic
    // For each completed session without compressed observations:
    //   compress each raw observation (synthetic or LLM)
    
    // Step 2: Episodic → Semantic
    // For each completed session without summary:
    //   summarize session (LLM or synthetic)
    
    // Step 3: Semantic → Procedural
    // Distill cross-session patterns into:
    //   - ProceduralMemory (repeated workflows)
    //   - Lessons (specific learnings)
    //   - Insights (cross-lesson patterns)
    
    // Step 4: Decay + Eviction
    //   - Apply strength decay to all memories
    //   - Evict memories below threshold
    //   - Run lesson decay sweep
}
```

### 3.2. LLM Compression (opt-in)

**Environment:** `AGENTMEMORY_AUTO_COMPRESS=true`

```go
// POST /memory/compress
type CompressRequest struct {
    ObservationID string `json:"observation_id"`
    SessionID     string `json:"session_id"`
    Force         bool   `json:"force,omitempty"` // re-compress even if exists
}

const compressionSystemPrompt = `You are a memory system. Compress this AI agent observation.
Return ONLY valid JSON:
{
  "title": "one-line action summary (max 80 chars)",
  "subtitle": "key detail or null",
  "facts": ["key fact 1", "key fact 2", "key fact 3"],
  "narrative": "2-3 sentence description of what happened and why",
  "concepts": ["entity1", "entity2"],
  "files": ["/path/to/affected/file"],
  "importance": 0.7
}`

// Graceful degrade: if LLM unavailable or returns error → use synthetic compression
// Circuit breaker: after 3 LLM failures → switch to noop mode for 5 minutes
```

### 3.3. Session Summarization

```go
// internal/consolidation/summarize.go
// Triggered: when session ends (hook: session_end)
// Also: background pipeline picks up sessions without summaries

const summarizeSystemPrompt = `Summarize this AI agent coding session.
Return JSON:
{
  "title": "1 sentence session title",
  "narrative": "2-3 paragraph description",
  "key_decisions": ["decision 1", "decision 2"],
  "files_modified": ["/path/to/file"],
  "concepts": ["concept1", "concept2"]
}`

type SessionSummary struct {
    SessionID        string    `json:"session_id"`
    Title            string    `json:"title"`
    Narrative        string    `json:"narrative"`
    KeyDecisions     []string  `json:"key_decisions"`
    FilesModified    []string  `json:"files_modified"`
    Concepts         []string  `json:"concepts"`
    ObservationCount int       `json:"observation_count"`
    CreatedAt        time.Time `json:"created_at"`
}
```

### 3.4. Procedural Memory Extraction

```go
// internal/consolidation/procedural.go

type ProceduralMemory struct {
    ID               string
    Name             string      // procedure name e.g. "Deploy Workflow"
    Steps            []string    // ordered step descriptions
    TriggerCondition string      // when to apply
    ExpectedOutcome  string
    Frequency        int         // how many times this pattern appeared
    Confidence       float64     // 0-1
    TenantID         string
    Project          string
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

// Detection algorithm:
// 1. For each pair of sessions: compute step sequence similarity
// 2. If same sequence appears >= MIN_FREQUENCY (default 3) times:
//    → Create/update ProceduralMemory
// 3. LLM: generate name, trigger_condition, expected_outcome from step sequence
```

### 3.5. Lessons & Insights

```go
// internal/consolidation/lessons.go

type Lesson struct {
    ID          string
    Content     string    // e.g. "Using async/await reduces callback hell"
    Confidence  float64   // 0-1, decays daily
    Source      string    // session_id or memory_id
    Categories  []string  // e.g. ["typescript", "best-practice"]
    TenantID    string
    Project     string
    AccessCount int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Insight struct {
    ID        string
    Content   string    // higher-level pattern across lessons
    LessonIDs []string  // source lessons
    Confidence float64
    CreatedAt  time.Time
}

// Lesson decay: daily sweep
// decay_rate = exp(-1.0 / LESSON_HALF_LIFE_DAYS)
// confidence = confidence * decay_rate
// Archive when confidence < 0.05
```

### 3.6. New API Endpoints

```
POST /v1/memory/compress              # Trigger LLM compression for observation
POST /v1/memory/summarize             # Summarize session  
POST /v1/memory/consolidate           # Run full consolidation pipeline (admin)
GET  /v1/memory/procedural            # List procedural memories
GET  /v1/memory/procedural/{id}       # Get procedural memory detail
GET  /v1/memory/lessons               # List lessons (filter: project, categories)
GET  /v1/memory/lessons/{id}          # Get lesson
POST /v1/memory/lessons/decay-sweep   # Run lesson decay sweep (admin)
GET  /v1/memory/insights              # List insights
```

### 3.7. Circuit Breaker for LLM

```go
// pkg/provider/circuit_breaker.go
type CircuitBreaker struct {
    state           string    // "closed" | "open" | "half-open"
    failureCount    int
    lastFailure     time.Time
    threshold       int       // default 3
    cooldownPeriod  time.Duration  // default 5 minutes
}

// closed → open: failureCount >= threshold
// open → half-open: after cooldownPeriod
// half-open → closed: successful LLM call
// half-open → open: failed LLM call
```

### 3.8. New PostgreSQL Tables

```sql
CREATE TABLE session_summaries (
    session_id   TEXT PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    title        TEXT,
    narrative    TEXT,
    key_decisions TEXT[],
    files_modified TEXT[],
    concepts     TEXT[],
    observation_count INT,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE procedural_memories (
    id           UUID PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    project      TEXT,
    name         TEXT NOT NULL,
    steps        TEXT[],
    trigger_condition TEXT,
    expected_outcome  TEXT,
    frequency    INT DEFAULT 1,
    confidence   FLOAT8 DEFAULT 0.5,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE lessons (
    id           UUID PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    project      TEXT,
    content      TEXT NOT NULL,
    confidence   FLOAT8 DEFAULT 0.7,
    source       TEXT,
    categories   TEXT[],
    access_count INT DEFAULT 0,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE insights (
    id           UUID PRIMARY KEY,
    tenant_id    TEXT NOT NULL,
    content      TEXT NOT NULL,
    lesson_ids   UUID[],
    confidence   FLOAT8 DEFAULT 0.6,
    created_at   TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 4. Environment Variables

| Variable | Default | Mô tả |
|---|---|---|
| `AGENTMEMORY_AUTO_COMPRESS` | `false` | Enable LLM compression per observation |
| `CONSOLIDATION_INTERVAL_HOURS` | `2` | Pipeline run interval |
| `CONSOLIDATION_DECAY_DAYS` | `30` | Memory strength decay half-life |
| `MAX_MEMORIES_PER_PROJECT` | `1000` | Eviction threshold |
| `LESSON_HALF_LIFE_DAYS` | `14` | Lesson confidence decay |
| `LESSON_DECAY_SWEEP_HOURS` | `24` | Lesson sweep interval |
| `MIN_PROCEDURE_FREQUENCY` | `3` | Min sessions to create ProceduralMemory |
| `LLM_CIRCUIT_BREAKER_THRESHOLD` | `3` | Failures before circuit opens |
| `LLM_COOLDOWN_MINUTES` | `5` | Circuit open duration |

---

## 5. Acceptance Criteria

- [x] Background consolidation goroutine chạy mỗi 2h (verify qua log: `"consolidation pipeline completed"`).
- [x] Sau session kết thúc (`session_end` hook): `session_summaries` table được populate trong ≤ 30s.
- [x] `POST /v1/memory/compress` với `AGENTMEMORY_AUTO_COMPRESS=false`: sử dụng synthetic compression, no LLM call.
- [x] `POST /v1/memory/compress` với `AGENTMEMORY_AUTO_COMPRESS=true`: gọi LLM, trả về `facts[]`.
- [x] LLM failure 3 lần liên tiếp → circuit breaker opens → compression falls back to synthetic.
- [x] Memory `strength` sau 30 ngày = original_strength × 0.5 (verify bằng `GET /v1/memory/agent/{id}/retention`).
- [x] Cùng 4-step workflow trong 3 sessions → `GET /v1/memory/procedural` trả về ProceduralMemory tương ứng.
- [x] `POST /v1/memory/lessons/decay-sweep` giảm confidence của lessons không được access trong 14 ngày.

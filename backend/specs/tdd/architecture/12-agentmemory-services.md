# AgentMemory Services — Technical Design

> **Updated**: 2026-09-03 — synced from code
> **Services**: `observe-service`, `orchestration-service`, `pipeline-service`
> **Group**: AgentMemory Layer — trên top của 6 memory engines

---

## 1. observe-service

**Module**: `vnp-memory/services/observe-service`
**Role**: Hook capture pipeline cho AI agents (Claude Code, Cursor, Codex, etc.)

### Domain Models (từ `internal/domain/entity.go`)

```go
type Session struct {
    ID               string
    TenantID         string
    Project          string
    CWD              string
    Model            string
    AgentID          string
    Status           string     // "active" | "completed" | "abandoned"
    FirstPrompt      string
    Summary          string
    ObservationCount int
    Tags             []string
    CommitSHAs       []string
    StartedAt        time.Time
    EndedAt          *time.Time
    LastActiveAt     time.Time
}

type RawObservation struct {
    ID        string
    SessionID string
    TenantID  string
    HookType  string     // 12 hook types (per HLD C4 canonical definition):
                         // session_start, session_end, llm_prompt, llm_response,
                         // tool_call, tool_result, memory_read, memory_write,
                         // plan_step, decision, error, checkpoint
                         // Note: current code uses alternate names — see ADR alignment task
    ObsType   string     // 15 observation sub-types: tool_call, tool_success, error,
                         // conversation, file_write, file_read, search, exec,
                         // commit, build, test, install, api_call, memory, decision
    ToolName  string
    Payload   []byte     // JSON, PII-redacted before persist
    CreatedAt time.Time
}
```

### 14-Step Pipeline (HLD canonical — docs/hld/C3-component.md)

> **HLD definition** (C3): Validate → Auth → Dedup → Redact → Parse → Enrich → Classify
> → Store(postgres) → Index(BM25) → Embed(vector) → Publish(NATS) → Update Session → Stream SSE
>
> **Current implementation** (`observe/pipeline.go`): 13 steps — Auth delegated to Gateway middleware,
> Embed step pending implementation. All other steps present with different naming.

| Step | HLD name | Current impl | Status |
|---|---|---|---|
| 1 | Validate | validate | ✅ |
| 2 | Auth + TenantID | (gateway middleware) | ✅ delegated |
| 3 | Dedup (30s TTL) | dedup | ✅ |
| 4 | Redact PII+secrets | privacy | ✅ |
| 5 | Parse payload | build | ✅ |
| 6 | Enrich metadata | agentId | ✅ partial |
| 7 | Classify hook type | (implicit in build) | ✅ |
| 8 | Store → PostgreSQL | persist | ✅ |
| 9 | Index BM25 | index | ✅ |
| 10 | Embed → vector | (pending) | 🔲 CR-AGENT-002 |
| 11 | Publish NATS | (NATS on EndSession) | ✅ |
| 12 | Update session state | session | ✅ |
| 13 | Stream SSE | stream | ✅ |
| 14 | Compress (rule-based) | compress | ✅ |

```
HLD 14-step pipeline implementation status:
1.validate → 2.auth(gateway) → 3.dedup → 4.privacy → 5.build → 6.image
→ 7.mutex → 8.limit → 9.agentId → 10.persist → 11.stream → 12.session
→ 13.compress → 14.index
[Embed step: pending — see CR-AGENT-002]
```

### API Endpoints (HTTP)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/observe` | Submit hook observation |
| POST | `/sessions` | Start session |
| POST | `/sessions/{id}/end` | End session + trigger consolidation |
| GET | `/sessions/{id}` | Get session detail |
| GET | `/sessions/{id}/observations` | List raw observations |
| GET | `/stream` | SSE live stream |

---

## 2. orchestration-service

**Module**: `vnp-memory/services/orchestration-service`
**Role**: Distributed lease management và agent-to-agent signaling

### Domain Models

```go
type Lease struct {
    LeaseID    string        // UUID
    Resource   string        // "user_profile:u1", "session:s1", "memory:m1"
    TenantID   string
    AgentID    string
    ExpiresAt  time.Time     // TTL anti-deadlock
    CreatedAt  time.Time
}

// Acquire: INSERT ... ON CONFLICT DO UPDATE WHERE expires_at < NOW()
// Only steal expired leases; active leases return ErrLeaseHeld

type Signal struct {
    SignalID   string
    FromAgent  string
    ToAgent    string
    TenantID   string
    Type       SignalType    // "handoff" | "alert" | "update" | "query"
    Payload    map[string]any
    CreatedAt  time.Time
}

type SignalType string
const (
    SignalHandoff SignalType = "handoff"
    SignalAlert   SignalType = "alert"
    SignalUpdate  SignalType = "update"
    SignalQuery   SignalType = "query"
)
```

### PostgreSQL Schema

```sql
CREATE TABLE leases (
    lease_id    UUID PRIMARY KEY,
    resource    TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    agent_id    TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (resource, tenant_id)
);

CREATE TABLE signals (
    signal_id   UUID PRIMARY KEY,
    from_agent  TEXT NOT NULL,
    to_agent    TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    type        TEXT NOT NULL,
    payload     JSONB,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    read_at     TIMESTAMPTZ
);
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/leases` | Acquire lease (409 if held) |
| DELETE | `/leases/{id}` | Release lease |
| GET | `/leases` | List active leases (admin) |
| POST | `/signals` | Send signal between agents |
| GET | `/signals?agent_id={id}` | Get pending signals |

---

## 3. pipeline-service

**Module**: `vnp-memory/services/pipeline-service`
**Role**: 4-tier memory consolidation pipeline (mirror neuroscience sleep model)

### Tier Architecture

```
TIER 1 — LLM Compression (mirrors NREM Stage 1-2)
  Trigger: NATS "agent.session.complete" event OR every 50 hooks
  Input: raw hooks grouped by 5-minute windows
  LLM: compress batch → 70-90% size reduction
  Circuit breaker: stop if LLM fails 3x consecutive
  Output: compressed_blobs table

TIER 2 — Session Summary (mirrors NREM Stage 3-4)
  Input: all Tier 1 outputs for one session
  LLM: "What happened in this session?"
  Output: session_summary { attempted, succeeded, failed, decisions, entities }

TIER 3 — Procedure Extraction (mirrors REM)
  Trigger: N sessions threshold (configurable, default: 5)
  Input: multiple session summaries
  LLM: extract reusable procedures from patterns
  Output: procedural memory → OpenViking L1

TIER 4 — Cross-Session Insights (mirrors deep sleep integration)
  Trigger: weekly batch OR N sessions (configurable: 20)
  Input: session summaries across multiple agents
  LLM: cross-agent patterns, lessons learned
  Output: adaptive memory → Supermemory
```

### Domain Models

```go
type ConsolidationJob struct {
    JobID     string
    SessionID string
    AgentID   string
    TenantID  string
    Tier      int           // 1 | 2 | 3 | 4
    Status    JobStatus     // "queued" | "running" | "completed" | "failed"
    Input     int           // hooks_input count
    Output    int           // summaries_output count
    Reduction float64       // compression percentage
    StartedAt time.Time
    EndedAt   *time.Time
}

type CompressedBlob struct {
    ID        string
    SessionID string
    TenantID  string
    Content   string
    Tier      int
    Window    time.Time     // 5-minute window start
    CreatedAt time.Time
}

type SessionSummary struct {
    ID          string
    SessionID   string
    TenantID    string
    Attempted   []string
    Succeeded   []string
    Failed      []string
    Decisions   []string
    Entities    []string
    CreatedAt   time.Time
}
```

### NATS Integration

```
Subscribe: "agent.session.complete" → trigger Tier 1 + Tier 2
Subscribe: "pipeline.tier2.done" → check if N sessions threshold → Tier 3
Cron: weekly → Tier 4 batch

Publish: "pipeline.tier1.done" { job_id, session_id }
Publish: "pipeline.tier2.done" { job_id, session_id, summary_id }
Publish: "pipeline.tier3.done" { job_id, procedure_ids[] }
Publish: "pipeline.tier4.done" { job_id, insight_ids[] }
```

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/consolidate` | Manual trigger (session_id, tier) |
| GET | `/jobs/{id}` | Get job status + stats |
| GET | `/jobs?session_id={id}` | List jobs for session |

---

## 4. Service Interactions

```
observe-service ──(NATS: agent.session.complete)──▶ pipeline-service
                                                          │ Tier 1+2
                                                          ▼
                                               observe-service (compressed blobs)
                                               
orchestration-service ◀── all agents (lease acquire/release, signals)

pipeline-service Tier 3 ──▶ ov-resource (procedural memory)
pipeline-service Tier 4 ──▶ sm-memory (adaptive memory / insights)
```

---

## 4. memory-service [AgentMemory Layer — per HLD]

**Module**: `vnp-memory/services/memory-service`
**Role**: Agent Memory Lifecycle — per HLD C2/C3 (F09)
**Layer**: AgentMemory (not Platform) per HLD canonical grouping

> **Implementation note**: memory-service provides Jaccard versioning, salience-based
> eviction, and 6 memory type support. It is architecturally grouped under AgentMemory
> in HLD, though its Go module currently sits alongside platform services.

### HLD Memory Types (C4 canonical — 6 types)

```go
// Per HLD C4-code.md — target interface
type MemoryType string
const (
    MemoryTypeEpisodic      MemoryType = "episodic"
    MemoryTypeSemantic      MemoryType = "semantic"
    MemoryTypeConversational MemoryType = "conversational"
    MemoryTypeProfile       MemoryType = "profile"
    MemoryTypeProcedural    MemoryType = "procedural"
    MemoryTypeAdaptive      MemoryType = "adaptive"  // Supermemory engine
    // "auto" for LLM-classified routing
)
```

### Core Components (per HLD C3)

| Component | Function |
|---|---|
| **Jaccard Versioning Engine** | similarity threshold → merge or create new version |
| **Eviction Manager** | salience = importance × recency × frequency |
| **Memory Decay** | time-based score decay for all types |
| **Slots Manager** | scope+label based key-value memory slots |

### StoreRequest (HLD canonical — TenantID in body)

```go
// Per HLD C4 — TenantID carried in request for cross-service consistency
type StoreRequest struct {
    TenantID string         `json:"tenant_id"` // propagated from gateway auth
    UserID   string         `json:"user_id"`
    Content  string         `json:"content"`
    Type     MemoryType     `json:"type"`
    Metadata map[string]any `json:"metadata,omitempty"`
}
```

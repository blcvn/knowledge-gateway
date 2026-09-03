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
    HookType  string     // 12 types: session_start, prompt_submit, pre_tool_use,
                         // post_tool_use, post_tool_failure, pre_compact,
                         // subagent_start, subagent_stop, notification,
                         // task_completed, stop, session_end
    ToolName  string
    Payload   []byte     // JSON, PII-redacted before persist
    CreatedAt time.Time
}
```

### 14-Step Pipeline

```
1. validate       — schema check HookType valid
2. dedup          — SHA256 hash, 30s TTL window
3. privacy        — PII redaction (shared/pkg/privacy)
4. build          — construct RawObservation struct
5. image          — extract image attachments (if any)
6. mutex          — per-session ordering mutex
7. limit          — max observations per session check
8. agentId        — validate + normalize agent ID
9. persist        — save to PostgreSQL
10. stream        — SSE broadcast to viewers
11. session       — update session metadata (LastActiveAt, ObservationCount)
12. compress      — synthetic compression (no LLM, rule-based)
13. index         — update BM25 in-memory index for observe-search
(NATS publish)     — "agent.session.complete" on EndSession
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

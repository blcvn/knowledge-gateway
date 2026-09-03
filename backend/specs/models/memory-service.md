# memory-service — Data Models

> **Service**: `services/memory-service`
> **Role**: Unified memory facade — aggregates Zep, Memobase, and Supermemory (sm) sub-domains into a single service.
> Handles agent memory with forgetting-curve mechanics, consolidation, and audit trails.

---

## agentmemory — Agent Memory (core sub-domain)

### AgentMemory

```go
type AgentMemory struct {
    ID                   string
    TenantID             string
    Project              string
    Type                 MemoryType
    Title                string
    Content              string
    Concepts             []string
    Files                []string
    SessionIDs           []string
    Strength             float64    // 0.0 - 1.0 (Ebbinghaus decay)
    Version              int
    ParentID             string
    Supersedes           []string   // IDs this memory supersedes
    RelatedIDs           []string
    SourceObservationIDs []string
    IsLatest             bool
    ForgetAfter          *time.Time
    AgentID              string
    FlaggedEviction      bool
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

type MemoryType string
// pattern | preference | architecture | bug | workflow | fact
```

### MemorySlot

```go
type MemorySlot struct {
    TenantID    string
    Project     string
    Scope       string    // "project" | "global"
    Label       string
    Content     string
    Description string
    SizeLimit   int
    Pinned      bool      // immune to eviction
    ReadOnly    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Consolidation Entities

```go
type SessionSummary struct {
    SessionID        string
    TenantID         string
    Title            string
    Narrative        string
    KeyDecisions     []string
    FilesModified    []string
    Concepts         []string
    ObservationCount int
    CreatedAt        time.Time
}

type ProceduralMemory struct {
    ID               string
    TenantID         string
    Project          string
    Name             string
    Steps            []string
    StepHash         string    // for dedup
    TriggerCondition string
    ExpectedOutcome  string
    Frequency        int
    Confidence       float64
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

type Lesson struct {
    ID          string
    TenantID    string
    Project     string
    Content     string
    Confidence  float64
    Source      string
    Categories  []string
    AccessCount int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Insight struct {
    ID         string
    TenantID   string
    Content    string
    LessonIDs  []string
    Confidence float64
    CreatedAt  time.Time
}

type RawObs struct {
    ID                string
    SessionID         string
    TenantID          string
    HookType          string
    ToolName          string
    ToolInput         []byte
    ToolOutput        []byte
    UserPrompt        string
    AssistantResponse string
}

type CompressedObs struct {
    ID         string
    SessionID  string
    ObsType    string
    Title      string
    Subtitle   string
    Facts      []string
    Narrative  string
    Concepts   []string
    Files      []string
    Importance float64
    Confidence float64
}
```

### AuditEntry

```go
type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    Operation   string        // 25 operation types (see below)
    TargetIDs   []string
    PerformedBy string
    Project     string
    TenantID    string
    Details     map[string]any
    Reason      string
}

// Operation constants:
// observe, remember, supersede, forget, governance_delete, evict, auto_forget,
// compress, summarize, consolidate, slot_write, slot_delete, session_start,
// session_end, session_delete, import_transcript, search_query, context_build,
// signal_send, lease_acquire, lease_release, checkpoint_create,
// checkpoint_resolve, snapshot_create, decay_sweep
```

---

## memobase — Working Memory (Memobase integration)

```go
type Blob struct {
    ID        string         `json:"id"`
    UserID    string         `json:"user_id"`
    TenantID  string         `json:"tenant_id"`
    Type      string         `json:"type"` // "conversation" | "fact" | "document" | "image"
    Content   string         `json:"content"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Embedding []float32      `json:"embedding,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type UserContext struct {
    UserID   string     `json:"user_id"`
    TenantID string     `json:"tenant_id"`
    Summary  string     `json:"summary"`
    Profiles []*Profile `json:"profiles,omitempty"`
    Events   []*Event   `json:"events,omitempty"`
    Tokens   int        `json:"tokens"`
}

type Profile struct {
    Key       string    `json:"key"`
    Value     string    `json:"value"`
    Category  string    `json:"category"` // "preference" | "fact" | "goal" | "habit"
    Score     float64   `json:"score"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Event struct {
    ID        string         `json:"id"`
    UserID    string         `json:"user_id"`
    EventType string         `json:"event_type"`
    Content   string         `json:"content"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type Buffer struct {
    UserID         string  `json:"user_id"`
    Blobs          []*Blob `json:"blobs"`
    TokenCount     int     `json:"token_count"`
    FlushThreshold int     `json:"flush_threshold"`
}
```

---

## sm — Supermemory Integration

```go
type SMMemory struct {
    ID        string         `json:"id"`
    TenantID  string         `json:"tenant_id"`
    Content   string         `json:"content"`
    Tags      []string       `json:"tags,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Embedding []float32      `json:"embedding,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type SMDocument struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    Type      string    `json:"type"` // "markdown" | "pdf" | "html"
    URL       string    `json:"url,omitempty"`
    CreatedAt time.Time `json:"created_at"`
}

type SMProfile struct {
    UserID   string      `json:"user_id"`
    TenantID string      `json:"tenant_id"`
    Memories []*SMMemory `json:"memories,omitempty"`
    Tags     []string    `json:"tags,omitempty"`
    Stats    ProfileStats `json:"stats"`
}

type ProfileStats struct {
    TotalMemories int   `json:"total_memories"`
    TotalTokens   int64 `json:"total_tokens"`
}

type RAGResponse struct {
    Context string      `json:"context"`
    Sources []*SMMemory `json:"sources,omitempty"`
    Tokens  int         `json:"tokens"`
}
```

---

## zep — Zep Integration

```go
type ZepUser struct {
    UserID    string         `json:"user_id"`
    Email     string         `json:"email"`
    FirstName string         `json:"first_name"`
    LastName  string         `json:"last_name"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type ZepSession struct {
    SessionID string         `json:"session_id"`
    UserID    string         `json:"user_id"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type ZepMemory struct {
    SessionID string       `json:"session_id"`
    Messages  []ZepMessage `json:"messages,omitempty"`
    Summary   *ZepSummary  `json:"summary,omitempty"`
    Facts     []string     `json:"facts,omitempty"`
}

type ZepMessage struct {
    Role      string         `json:"role"` // "user" | "assistant"
    Content   string         `json:"content"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    CreatedAt time.Time      `json:"created_at"`
}

type ZepSummary struct {
    Content    string    `json:"content"`
    TokensUsed int       `json:"tokens_used"`
    CreatedAt  time.Time `json:"created_at"`
}

type GraphFact struct {
    UUID     string   `json:"uuid"`
    Name     string   `json:"name"`
    Fact     string   `json:"fact"`
    Episodes []string `json:"episodes,omitempty"`
}
```

---

## Sources
- [`services/memory-service/internal/domain/agentmemory/entity.go`](../../services/memory-service/internal/domain/agentmemory/entity.go)
- [`services/memory-service/internal/domain/agentmemory/value_object.go`](../../services/memory-service/internal/domain/agentmemory/value_object.go)
- [`services/memory-service/internal/domain/agentmemory/consolidation_entities.go`](../../services/memory-service/internal/domain/agentmemory/consolidation_entities.go)
- [`services/memory-service/internal/domain/agentmemory/audit.go`](../../services/memory-service/internal/domain/agentmemory/audit.go)
- [`services/memory-service/internal/domain/memobase/entity.go`](../../services/memory-service/internal/domain/memobase/entity.go)
- [`services/memory-service/internal/domain/sm/entity.go`](../../services/memory-service/internal/domain/sm/entity.go)
- [`services/memory-service/internal/domain/zep/entity.go`](../../services/memory-service/internal/domain/zep/entity.go)

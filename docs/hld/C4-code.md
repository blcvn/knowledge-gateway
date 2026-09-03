# C4 — Code Level (Key Domain Models & Interfaces)

> **C4 Level 4:** Các domain models và interfaces quan trọng nhất.
> Đây là "contract" giữa các services — không phải implementation details.

---

## Core Domain Types

### Memory Unit

```go
// Shared: backend/shared/pkg/adapters/
type MemoryUnit struct {
    ID        string            `json:"id"`
    TenantID  string            `json:"tenant_id"`   // Multi-tenancy isolation
    UserID    string            `json:"user_id"`
    AgentID   string            `json:"agent_id,omitempty"`
    SessionID string            `json:"session_id,omitempty"`
    Type      MemoryType        `json:"type"`        // episodic/semantic/conversational/profile/procedural/adaptive
    Content   string            `json:"content"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
    Score     float64           `json:"score,omitempty"`    // relevance score from search
    CreatedAt time.Time         `json:"created_at"`
    UpdatedAt time.Time         `json:"updated_at"`
}

type MemoryType string
const (
    MemoryTypeEpisodic      MemoryType = "episodic"
    MemoryTypeSemantic      MemoryType = "semantic"
    MemoryTypeConversational MemoryType = "conversational"
    MemoryTypeProfile       MemoryType = "profile"
    MemoryTypeProcedural    MemoryType = "procedural"
    MemoryTypeAdaptive      MemoryType = "adaptive"
    MemoryTypeAuto          MemoryType = "auto"     // LLM classifies
)
```

### Unified Memory API Interface

```go
// Gateway domain — backend/gateway/domain/
type MemoryStore interface {
    Store(ctx context.Context, req *StoreRequest) (*StoreResponse, error)
    Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error)
    Forget(ctx context.Context, req *ForgetRequest) (*ForgetResponse, error)
    Timeline(ctx context.Context, req *TimelineRequest) (*TimelineResponse, error)
}

type StoreRequest struct {
    TenantID  string            `json:"tenant_id"`
    UserID    string            `json:"user_id"`
    Content   string            `json:"content"`
    Type      MemoryType        `json:"type"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
}

type RecallRequest struct {
    TenantID  string            `json:"tenant_id"`
    UserID    string            `json:"user_id"`
    Query     string            `json:"query"`
    Types     []MemoryType      `json:"types,omitempty"`      // filter by type
    TimeRange *TimeRange        `json:"time_range,omitempty"` // temporal filter
    Limit     int               `json:"limit,omitempty"`
}

type RecallResponse struct {
    Results   []MemoryUnit      `json:"results"`
    TotalHits int               `json:"total_hits"`
}
```

### Observe Hook Interface

```go
// Observe domain — services/observe-service/domain/
type HookType string
const (
    HookSessionStart    HookType = "session_start"
    HookSessionEnd      HookType = "session_end"
    HookLLMPrompt       HookType = "llm_prompt"
    HookLLMResponse     HookType = "llm_response"
    HookToolCall        HookType = "tool_call"
    HookToolResult      HookType = "tool_result"
    HookMemoryRead      HookType = "memory_read"
    HookMemoryWrite     HookType = "memory_write"
    HookPlanStep        HookType = "plan_step"
    HookDecision        HookType = "decision"
    HookError           HookType = "error"
    HookCheckpoint      HookType = "checkpoint"
)

type ObserveHook struct {
    TenantID  string            `json:"tenant_id"`
    AgentID   string            `json:"agent_id"`
    SessionID string            `json:"session_id"`
    HookType  HookType          `json:"hook_type"`
    Payload   map[string]any    `json:"payload"`
    Timestamp time.Time         `json:"timestamp"`
}

type ObserveService interface {
    CaptureHook(ctx context.Context, hook *ObserveHook) error
    GetSession(ctx context.Context, sessionID string) (*AgentSession, error)
    ListSessions(ctx context.Context, req *ListSessionsRequest) ([]*AgentSession, error)
    ReplaySession(ctx context.Context, sessionID string) (<-chan *ObserveHook, error)
}
```

### Multi-Tenant Context

```go
// Shared: backend/shared/pkg/tenant/
type TenantContext struct {
    TenantID      string
    UserID        string
    Tier          SubscriptionTier // free / pro / enterprise
    RateLimitRPM  int
    AllowedFeatures []string
}

// Injected by Auth middleware via context key
type contextKey string
const TenantContextKey contextKey = "tenant_ctx"

// Every DB query MUST include TenantID:
// SELECT * FROM memories WHERE tenant_id = $1 AND user_id = $2
```

### InProcessRegistry Interface

```go
// Monolith mode — backend/apps/memory/
type ServiceRegistry interface {
    Register(name string, conn *grpc.ClientConn) error
    Lookup(name string) (*grpc.ClientConn, error)
    ListServices() []string
    HealthCheck(ctx context.Context) map[string]bool
}

// Service names (35+ services)
const (
    ServiceCogneeIngestion    = "cognee-ingestion"
    ServiceCogneeSearch       = "cognee-search"
    ServiceGraphitiIngestion  = "graphiti-ingestion"
    ServiceGraphitiSearch     = "graphiti-search"
    ServiceMemobaseIngestion  = "memobase-ingestion"
    ServiceMemobaseEngine     = "memobase-engine"
    ServiceMemobaseContext    = "memobase-context"
    ServiceZepMemory          = "zep-memory"
    ServiceZepGraph           = "zep-graph"
    ServiceOVFs               = "ov-fs"
    ServiceOVSearch           = "ov-search"
    ServiceSMMemory           = "sm-memory"
    ServiceSMSearch           = "sm-search"
    ServiceSearchHub          = "vnp-search-hub"
    ServiceObserve            = "observe-service"
    ServiceMemoryLifecycle    = "memory-service"
    ServiceOrchestration      = "orchestration-service"
    ServicePipeline           = "pipeline-service"
    // ... 17+ more
)
```

### Consolidation Pipeline Interface

```go
// Consolidation — services/pipeline-service/domain/
type ConsolidationTier int
const (
    TierCompression  ConsolidationTier = 1 // LLM batch compress (70-90% reduction)
    TierSummary      ConsolidationTier = 2 // Session summary (what/why/decision)
    TierProcedure    ConsolidationTier = 3 // Extract reusable procedures
    TierInsight      ConsolidationTier = 4 // Cross-session insights
)

type ConsolidationJob struct {
    SessionID  string
    AgentID    string
    TenantID   string
    Tier       ConsolidationTier
    InputHooks []ObserveHook
    Options    ConsolidationOptions
}

type ConsolidationOptions struct {
    MaxTokensInput  int     // token budget for LLM
    TargetReduction float64 // 0.7-0.9 compression target
    CircuitBreaker  bool    // stop if LLM fails
}
```

### Multi-Agent Coordination

```go
// Orchestration — services/orchestration-service/domain/
type Lease struct {
    LeaseID   string
    AgentID   string
    Resource  string       // resource being locked (user_id, session_id, etc.)
    TenantID  string
    ExpiresAt time.Time
    TTL       time.Duration
}

type Signal struct {
    FromAgent  string
    ToAgent    string
    Type       SignalType   // handoff / alert / update / query
    Payload    map[string]any
    TenantID   string
}

type ActionDAG struct {
    JobID     string
    Steps     []ActionStep
    State     DAGState     // pending / in_progress / completed / failed
    TenantID  string
}
```

---

## Key API Contracts (REST → gRPC mapping)

| REST Endpoint | gRPC Service | Proto method |
|---|---|---|
| `POST /v1/memory/store` | `{engine}-ingestion` | `Ingest(IngestRequest)` |
| `POST /v1/memory/recall` | `vnp-search-hub` | `Search(SearchRequest)` |
| `POST /v1/memory/forget` | All engines | `Delete(DeleteRequest)` |
| `GET /v1/memory/timeline` | `vnp-event` | `ListEvents(EventFilter)` |
| `POST /v1/observe/hooks` | `observe-service` | `CaptureHook(HookRequest)` |
| `GET /v1/memobase/context` | `memobase-context` | `GetContext(ContextRequest)` |
| `POST /v1/orchestration/lease` | `orchestration-service` | `AcquireLease(LeaseRequest)` |
| `POST /v1/consolidate` | `pipeline-service` | `Consolidate(ConsolidationJob)` |

---

## Database Schema Highlights

```sql
-- Every table has TenantID for isolation
CREATE TABLE memories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,          -- Zero-trust: always filter by this
    user_id     TEXT NOT NULL,
    agent_id    TEXT,
    session_id  TEXT,
    type        TEXT NOT NULL,          -- episodic/semantic/...
    content     TEXT NOT NULL,
    embedding   vector(1536),           -- pgvector for semantic search
    metadata    JSONB,
    salience    FLOAT DEFAULT 1.0,      -- Lifecycle decay score
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);

-- Mandatory index for multi-tenancy
CREATE INDEX idx_memories_tenant_user ON memories(tenant_id, user_id);
CREATE INDEX idx_memories_tenant_type ON memories(tenant_id, type);

-- Adaptive memory version chain (Supermemory)
CREATE TABLE sm_memories (
    id          UUID PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    parent_id   UUID REFERENCES sm_memories(id),  -- version chain
    root_id     UUID,
    is_latest   BOOLEAN DEFAULT true,              -- only query where is_latest=true
    forget_after TIMESTAMPTZ,                      -- TTL eviction
    ...
);
```

---

*[← C3 Component](./C3-component.md) | [→ Deployment](./deployment.md)*

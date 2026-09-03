# Solution: SOL-002 — Memory Lifecycle Management

**CR ID:** CR-AM-002  
**Solution ID:** SOL-002  
**Priority:** Critical (Wave 1)  
**Architecture:** EXTEND `services/memory-service/` + PostgreSQL

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `services/memory-service/` đã tồn tại với domain models: `Blob`, `UserContext`, `Profile`, `Event`, `Buffer` (Memobase), `ZepUser`, `ZepSession`, `ZepMemory` (Zep), `SMMemory`, `SMDocument` (Supermemory).
- `IngestUseCase` xử lý blob ingest vào Memobase buffer.
- PostgreSQL là primary store — sẵn sàng để thêm tables mới.
- NATS JetStream available cho events.

Chiến lược: **thêm AgentMemory domain** vào `services/memory-service/` cùng với Memobase/Zep/SM domains — không cần tạo service mới.

---

## 2. Giải pháp

### 2.1. [EXTEND] Thêm AgentMemory domain vào `services/memory-service/`

```
services/memory-service/
├── internal/
│   ├── domain/
│   │   ├── memobase/       # Existing
│   │   ├── zep/            # Existing
│   │   ├── sm/             # Existing
│   │   └── agentmemory/    # [NEW]
│   │       ├── entity.go   # AgentMemory, MemorySlot
│   │       ├── value_object.go  # MemoryType enum
│   │       └── errors.go
│   ├── usecase/
│   │   ├── memobase/       # Existing
│   │   ├── zep/            # Existing
│   │   ├── sm/             # Existing
│   │   └── agentmemory/    # [NEW]
│   │       ├── remember_agent.go     # Jaccard versioning
│   │       ├── recall_agent.go       # Search + filter latest
│   │       ├── forget_agent.go       # Governance delete
│   │       ├── evict.go              # Eviction policy
│   │       ├── auto_forget.go        # TTL sweep
│   │       ├── slots.go              # MemorySlot CRUD
│   │       ├── retention.go          # Retention scoring
│   │       ├── decay.go              # Strength decay scheduler
│   │       └── port/
│   │           ├── input.go          # IAgentMemoryUseCase
│   │           └── output.go         # IMemoryRepo, ISearchClient, IEventPublisher
│   ├── adapter/
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── agentmemory_repo.go  # [NEW]
│   │   │       └── slots_repo.go        # [NEW]
│   │   └── grpc/
│   │       └── agentmemory_handler.go   # [NEW] gRPC handler
└── api/proto/memory/v1/agentmemory.proto  # [NEW]
```

### 2.2. Domain Model — AgentMemory

```go
// services/memory-service/internal/domain/agentmemory/entity.go

type AgentMemory struct {
    ID                   string
    TenantID             string
    Project              string
    Type                 MemoryType
    Title                string
    Content              string
    Concepts             []string     // for Jaccard matching
    Files                []string
    SessionIDs           []string     // sessions where this was sourced
    Strength             float64      // 0.0–1.0, decays over time
    Version              int
    ParentID             string       // previous version ID
    Supersedes           []string     // IDs of superseded memories
    RelatedIDs           []string
    SourceObservationIDs []string
    IsLatest             bool         // false = superseded version
    ForgetAfter          *time.Time   // TTL auto-forget
    AgentID              string
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

type MemoryType string
const (
    MemTypePattern      MemoryType = "pattern"
    MemTypePreference   MemoryType = "preference"
    MemTypeArchitecture MemoryType = "architecture"
    MemTypeBug          MemoryType = "bug"
    MemTypeWorkflow     MemoryType = "workflow"
    MemTypeFact         MemoryType = "fact"
)

type MemorySlot struct {
    Label       string
    Content     string
    Scope       string     // "project" | "global"
    Description string
    SizeLimit   int
    Pinned      bool       // immune to eviction
    ReadOnly    bool
    TenantID    string
    Project     string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 2.3. Jaccard Versioning — `remember_agent.go`

```go
// services/memory-service/internal/usecase/agentmemory/remember_agent.go

const JaccardThreshold = 0.7

type RememberAgentUseCase struct {
    repo        port.IMemoryRepo
    searchClient port.ISearchNotifier
    publisher   port.IEventPublisher
}

func (uc *RememberAgentUseCase) Execute(ctx context.Context, req RememberRequest) (*RememberResult, error) {
    // 1. Load existing isLatest memories of same Type + Project
    existing, err := uc.repo.ListLatestByType(ctx, req.TenantID, req.Project, req.Type)
    
    // 2. Find best Jaccard match
    bestMatch, bestScore := findBestJaccardMatch(req.Concepts, existing)
    
    // 3. Supersede if threshold exceeded
    var superseded []string
    if bestScore >= JaccardThreshold {
        // Mark old memory as not latest
        uc.repo.SetNotLatest(ctx, bestMatch.ID)
        superseded = append(superseded, bestMatch.ID)
        
        // Publish supersede event
        uc.publisher.Publish(ctx, "agentmemory.memory.superseded", SupersedeEvent{
            OldID: bestMatch.ID, NewID: newID,
        })
    }
    
    // 4. Create new memory
    mem := AgentMemory{
        ID:        newID(),
        Type:      req.Type,
        Title:     req.Title,
        Content:   req.Content,
        Concepts:  req.Concepts,
        Files:     req.Files,
        SessionIDs: []string{req.SessionID},
        Strength:  req.Strength, // default 0.7 if not set
        Version:   maxVersion(existing) + 1,
        Supersedes: superseded,
        IsLatest:  true,
        ForgetAfter: req.ForgetAfter,
        AgentID:   req.AgentID,
        TenantID:  req.TenantID,
        Project:   req.Project,
    }
    if err := uc.repo.Save(ctx, mem); err != nil { return nil, err }
    
    // 5. Notify search service to reindex
    go uc.searchClient.IndexMemory(ctx, mem)
    
    // 6. Publish event
    uc.publisher.Publish(ctx, "agentmemory.memory.remembered", RememberedEvent{
        MemoryID: mem.ID, Type: string(mem.Type), Version: mem.Version, Superseded: superseded,
    })
    
    return &RememberResult{MemoryID: mem.ID, Version: mem.Version, Superseded: superseded}, nil
}

// Jaccard coefficient on lowercase concept sets
func jaccardSimilarity(a, b []string) float64 {
    setA := toLowerSet(a)
    setB := toLowerSet(b)
    intersection := 0
    for k := range setA { if setB[k] { intersection++ } }
    union := len(setA) + len(setB) - intersection
    if union == 0 { return 0 }
    return float64(intersection) / float64(union)
}
```

### 2.4. Eviction Policy — `evict.go`

```go
// services/memory-service/internal/usecase/agentmemory/evict.go

type EvictUseCase struct {
    repo      port.IMemoryRepo
    publisher port.IEventPublisher
}

// Eviction score formula:
//   score = importance × recency × frequency
//     importance = memory.Strength
//     recency    = exp(-daysSinceUpdate / 30.0)    // 30-day half-life
//     frequency  = log1p(len(sessionIDs))            // access count proxy

func (uc *EvictUseCase) Execute(ctx context.Context, req EvictRequest) (*EvictResult, error) {
    memories, _ := uc.repo.ListLatestByProject(ctx, req.TenantID, req.Project)
    if len(memories) <= req.MaxMemories { return &EvictResult{}, nil }
    
    // Score each memory
    scored := make([]scoredMemory, len(memories))
    for i, m := range memories {
        daysSince := time.Since(m.UpdatedAt).Hours() / 24
        recency := math.Exp(-daysSince / 30.0)
        frequency := math.Log1p(float64(len(m.SessionIDs)))
        scored[i] = scoredMemory{Memory: m, Score: m.Strength * recency * frequency}
    }
    
    // Sort by score ascending (lowest first = candidates for eviction)
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score < scored[j].Score })
    
    toEvict := scored[:len(memories)-req.MaxMemories]
    var evictedIDs []string
    for _, sm := range toEvict {
        if sm.Memory.Strength >= 1.0 { continue } // Pinned — immune
        evictedIDs = append(evictedIDs, sm.Memory.ID)
        if !req.DryRun {
            uc.repo.Delete(ctx, sm.Memory.ID)
            uc.publisher.Publish(ctx, "agentmemory.memory.expired", ExpiredEvent{
                MemoryID: sm.Memory.ID, Reason: "eviction",
            })
        }
    }
    return &EvictResult{EvictedIDs: evictedIDs, DryRun: req.DryRun}, nil
}
```

### 2.5. TTL Auto-Forget — `auto_forget.go`

```go
// services/memory-service/internal/usecase/agentmemory/auto_forget.go

type AutoForgetUseCase struct {
    repo         port.IMemoryRepo
    searchClient port.ISearchNotifier
    publisher    port.IEventPublisher
}

// Background goroutine: runs every 60 minutes
func (uc *AutoForgetUseCase) StartScheduler(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Minute)
    for {
        select {
        case <-ticker.C:
            uc.sweep(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (uc *AutoForgetUseCase) sweep(ctx context.Context) {
    expired, _ := uc.repo.FindExpired(ctx) // WHERE forget_after < NOW()
    for _, mem := range expired {
        uc.repo.Delete(ctx, mem.ID)
        uc.searchClient.RemoveMemory(ctx, mem.ID)
        uc.publisher.Publish(ctx, "agentmemory.memory.expired", ExpiredEvent{
            MemoryID: mem.ID, Reason: "ttl",
        })
    }
}
```

### 2.6. Strength Decay — `decay.go`

```go
// services/memory-service/internal/usecase/agentmemory/decay.go

// Background goroutine: runs every 2h
// decay_factor = exp(-hoursSinceAccess / (HALF_LIFE_DAYS * 24))
// If strength drops below 0.05 → schedule for eviction

func (d *DecayScheduler) Start(ctx context.Context) {
    ticker := time.NewTicker(2 * time.Hour)
    for {
        select {
        case <-ticker.C:
            memories, _ := d.repo.ListAll(ctx)
            for _, m := range memories {
                hoursSince := time.Since(m.UpdatedAt).Hours()
                factor := math.Exp(-hoursSince / (float64(d.halfLifeDays) * 24))
                newStrength := m.Strength * factor
                d.repo.UpdateStrength(ctx, m.ID, newStrength)
                if newStrength < 0.05 {
                    // Schedule for eviction on next eviction run
                    d.repo.FlagForEviction(ctx, m.ID)
                }
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 2.7. Memory Slots — `slots.go`

```go
// services/memory-service/internal/usecase/agentmemory/slots.go

func (uc *SlotsUseCase) WriteSlot(ctx context.Context, req WriteSlotRequest) error {
    slot, err := uc.repo.GetSlot(ctx, req.TenantID, req.Scope, req.Label)
    
    if err == ErrNotFound {
        // Create new slot
        return uc.repo.CreateSlot(ctx, MemorySlot{
            Label: req.Label, Content: req.Content,
            Scope: req.Scope, TenantID: req.TenantID, Project: req.Project,
        })
    }
    
    if slot.ReadOnly { return ErrReadOnly }
    
    switch req.Mode {
    case "replace":
        slot.Content = req.Content
    case "append":
        slot.Content = slot.Content + "\n" + req.Content
    }
    
    // Enforce size limit
    if slot.SizeLimit > 0 && len(slot.Content) > slot.SizeLimit {
        slot.Content = slot.Content[:slot.SizeLimit]
    }
    
    return uc.repo.UpdateSlot(ctx, slot)
}
```

### 2.8. Retention Scoring — `retention.go`

```go
// services/memory-service/internal/usecase/agentmemory/retention.go

type RetentionScore struct {
    Score             float64
    RecencyFactor     float64
    FrequencyFactor   float64
    ImportanceFactor  float64
    DaysSinceAccess   float64
    RecommendAction   string  // "keep" | "review" | "evict"
}

func (uc *RetentionUseCase) GetScore(ctx context.Context, memID string) (*RetentionScore, error) {
    mem, err := uc.repo.Get(ctx, memID)
    daysSince := time.Since(mem.UpdatedAt).Hours() / 24
    recency := math.Exp(-daysSince / 30.0)
    frequency := math.Log1p(float64(len(mem.SessionIDs)))
    score := mem.Strength * recency * frequency
    
    action := "keep"
    if score < 0.1 { action = "evict" } else if score < 0.3 { action = "review" }
    
    return &RetentionScore{
        Score: score, RecencyFactor: recency, FrequencyFactor: frequency,
        ImportanceFactor: mem.Strength, DaysSinceAccess: daysSince,
        RecommendAction: action,
    }, nil
}
```

### 2.9. gRPC Service Proto

```protobuf
// api/proto/memory/v1/agentmemory.proto

service AgentMemoryService {
  rpc RememberAgent(RememberAgentRequest) returns (RememberAgentResponse);
  rpc ListAgentMemories(ListAgentMemoriesRequest) returns (ListAgentMemoriesResponse);
  rpc GetAgentMemory(GetAgentMemoryRequest) returns (GetAgentMemoryResponse);
  rpc DeleteAgentMemory(DeleteAgentMemoryRequest) returns (DeleteAgentMemoryResponse);
  rpc GetRetentionScore(GetRetentionScoreRequest) returns (GetRetentionScoreResponse);
  rpc EvictMemories(EvictMemoriesRequest) returns (EvictMemoriesResponse);
  rpc AutoForgetSweep(AutoForgetSweepRequest) returns (AutoForgetSweepResponse);

  // Slots
  rpc GetSlot(GetSlotRequest) returns (GetSlotResponse);
  rpc WriteSlot(WriteSlotRequest) returns (WriteSlotResponse);
  rpc DeleteSlot(DeleteSlotRequest) returns (DeleteSlotResponse);
  rpc ListSlots(ListSlotsRequest) returns (ListSlotsResponse);
}
```

### 2.10. PostgreSQL Schema

```sql
-- Migration: 0011_agent_memories.up.sql

CREATE TABLE agent_memories (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    project         TEXT,
    type            TEXT NOT NULL,          -- pattern|preference|architecture|bug|workflow|fact
    title           TEXT NOT NULL,
    content         TEXT NOT NULL,
    concepts        TEXT[] DEFAULT '{}',
    files           TEXT[] DEFAULT '{}',
    session_ids     TEXT[] DEFAULT '{}',
    strength        FLOAT8 NOT NULL DEFAULT 0.7,
    version         INT NOT NULL DEFAULT 1,
    parent_id       UUID REFERENCES agent_memories(id),
    supersedes      UUID[] DEFAULT '{}',
    related_ids     UUID[] DEFAULT '{}',
    source_obs_ids  UUID[] DEFAULT '{}',
    is_latest       BOOLEAN NOT NULL DEFAULT TRUE,
    forget_after    TIMESTAMPTZ,
    agent_id        TEXT,
    flagged_eviction BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_memories_tenant_type_latest ON agent_memories(tenant_id, type, is_latest) WHERE is_latest = TRUE;
CREATE INDEX idx_agent_memories_tenant_project ON agent_memories(tenant_id, project) WHERE is_latest = TRUE;
CREATE INDEX idx_agent_memories_forget_after ON agent_memories(forget_after) WHERE forget_after IS NOT NULL;
CREATE INDEX idx_agent_memories_strength ON agent_memories(tenant_id, strength) WHERE is_latest = TRUE;

CREATE TABLE memory_slots (
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    scope       TEXT NOT NULL,
    label       TEXT NOT NULL,
    content     TEXT,
    description TEXT,
    size_limit  INT DEFAULT 0,
    pinned      BOOLEAN DEFAULT FALSE,
    read_only   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, scope, label)
);
```

### 2.11. Gateway Routes

```go
// Thêm vào gateway/internal/adapter/handler/router.go

// AgentMemory lifecycle routes
r.Post("/v1/memory/agent/remember",         h.ForwardTo("memory-service", "AgentMemoryService/RememberAgent"))
r.Get("/v1/memory/agent/list",              h.ForwardTo("memory-service", "AgentMemoryService/ListAgentMemories"))
r.Get("/v1/memory/agent/{id}",              h.ForwardTo("memory-service", "AgentMemoryService/GetAgentMemory"))
r.Delete("/v1/memory/agent/{id}",           h.ForwardTo("memory-service", "AgentMemoryService/DeleteAgentMemory"))
r.Get("/v1/memory/agent/{id}/retention",    h.ForwardTo("memory-service", "AgentMemoryService/GetRetentionScore"))
r.Post("/v1/memory/agent/evict",            h.ForwardTo("memory-service", "AgentMemoryService/EvictMemories"))
r.Post("/v1/memory/agent/auto-forget",      h.ForwardTo("memory-service", "AgentMemoryService/AutoForgetSweep"))

// Memory Slots routes
r.Get("/v1/memory/slots",                           h.ForwardTo("memory-service", "AgentMemoryService/ListSlots"))
r.Get("/v1/memory/slots/{scope}/{label}",            h.ForwardTo("memory-service", "AgentMemoryService/GetSlot"))
r.Post("/v1/memory/slots/{scope}/{label}",           h.ForwardTo("memory-service", "AgentMemoryService/WriteSlot"))
r.Delete("/v1/memory/slots/{scope}/{label}",         h.ForwardTo("memory-service", "AgentMemoryService/DeleteSlot"))
```

### 2.12. Bootstrap Integration

```go
// apps/memory/internal/bootstrap/memory.go — MODIFY thêm AgentMemory init

func InitMemoryService(reg *bus.InProcessRegistry, db *sql.DB, nats *nats.Conn, cfg *config.Config) {
    // Existing Memobase, Zep, SM init...
    
    // [NEW] AgentMemory init
    memRepo    := postgres.NewAgentMemoryRepo(db)
    slotsRepo  := postgres.NewSlotsRepo(db)
    searchClient := httpclient.NewSearchClient(cfg.ObserveSearch.URL)
    publisher  := natevent.NewPublisher(nats, "agentmemory")
    
    rememberUC := agentmemory.NewRememberAgentUseCase(memRepo, searchClient, publisher)
    evictUC    := agentmemory.NewEvictUseCase(memRepo, publisher)
    autoForgetUC := agentmemory.NewAutoForgetUseCase(memRepo, searchClient, publisher)
    retentionUC  := agentmemory.NewRetentionUseCase(memRepo)
    decayScheduler := agentmemory.NewDecayScheduler(memRepo, cfg.Memory.HalfLifeDays)
    slotsUC    := agentmemory.NewSlotsUseCase(slotsRepo)
    
    // Register gRPC handler
    agentmemorypb.RegisterAgentMemoryServiceServer(grpcServer, grpchandler.NewAgentMemoryHandler(
        rememberUC, evictUC, autoForgetUC, retentionUC, slotsUC,
    ))
    
    // Start background schedulers
    go autoForgetUC.StartScheduler(context.Background())
    go decayScheduler.Start(context.Background())
}
```

---

## 3. Files thay đổi

### [NEW] Files

| File | Mô tả |
|------|-------|
| `services/memory-service/internal/domain/agentmemory/entity.go` | AgentMemory, MemorySlot |
| `services/memory-service/internal/domain/agentmemory/value_object.go` | MemoryType enum |
| `services/memory-service/internal/usecase/agentmemory/remember_agent.go` | Jaccard versioning |
| `services/memory-service/internal/usecase/agentmemory/evict.go` | Eviction formula |
| `services/memory-service/internal/usecase/agentmemory/auto_forget.go` | TTL sweep |
| `services/memory-service/internal/usecase/agentmemory/slots.go` | Slot CRUD |
| `services/memory-service/internal/usecase/agentmemory/retention.go` | Retention scoring |
| `services/memory-service/internal/usecase/agentmemory/decay.go` | Strength decay |
| `services/memory-service/internal/adapter/repository/postgres/agentmemory_repo.go` | DB operations |
| `services/memory-service/internal/adapter/repository/postgres/slots_repo.go` | Slots DB |
| `services/memory-service/internal/adapter/grpc/agentmemory_handler.go` | gRPC handler |
| `api/proto/memory/v1/agentmemory.proto` | Proto contract |
| `db/migrations/0011_agent_memories.up.sql` | DB schema |

### [MODIFY] Files

| File | Thay đổi |
|------|---------|
| `apps/memory/internal/bootstrap/memory.go` | Add AgentMemory init section |
| `gateway/internal/adapter/handler/router.go` | Add `/v1/memory/agent/*` + `/v1/memory/slots/*` routes |
| `apps/memory/configs/config.yaml` | Add `memory.half_life_days: 30` |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-AM-002 | Covered by |
|-----------------|------------|
| POST remember với `["jose","jwt","auth"]` → version:1 | remember_agent.go |
| Jaccard < 0.7 → không supersede | jaccardSimilarity() |
| Jaccard > 0.7 → version:2, superseded:[id1] | findBestJaccardMatch() |
| Cũ có is_latest=false sau supersede | repo.SetNotLatest() |
| forget_after trong quá khứ → bị xóa sau sweep | auto_forget.go |
| evict dry_run=true → list IDs, DB không thay đổi | evict.go DryRun check |
| strength >= 1.0 → không bị evict | pinned check in evict.go |
| slot append mode → nội dung được nối | slots.go append case |
| GET retention → score + recommend_action | retention.go |

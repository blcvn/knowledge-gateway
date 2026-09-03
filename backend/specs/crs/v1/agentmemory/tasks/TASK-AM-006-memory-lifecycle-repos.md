# TASK-AM-006 — Memory Lifecycle: Repos + gRPC Handler

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-006 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/memory-service/internal/adapter/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-002 §2.10, §2.11, §2.12 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-005 |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** observe-service repos: PostgreSQL + BM25 index  
---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/memory-service/internal/adapter/repository/postgres/agentmemory_repo.go` |
| CREATE | `services/memory-service/internal/adapter/repository/postgres/slots_repo.go` |
| CREATE | `services/memory-service/internal/adapter/grpc/agentmemory_handler.go` |
| MODIFY | `apps/memory/internal/bootstrap/memory.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### `adapter/repository/postgres/agentmemory_repo.go`

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type AgentMemoryRepo struct{ db *pgxpool.Pool }

func NewAgentMemoryRepo(db *pgxpool.Pool) *AgentMemoryRepo { return &AgentMemoryRepo{db: db} }

func (r *AgentMemoryRepo) Save(ctx context.Context, m agentmemory.AgentMemory) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO agent_memories
            (id, tenant_id, project, type, title, content, concepts, files, session_ids,
             strength, version, parent_id, supersedes, is_latest, forget_after, agent_id, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
    `, m.ID, m.TenantID, m.Project, string(m.Type), m.Title, m.Content,
        pq.Array(m.Concepts), pq.Array(m.Files), pq.Array(m.SessionIDs),
        m.Strength, m.Version, nilIfEmpty(m.ParentID),
        pq.Array(m.Supersedes), m.IsLatest, m.ForgetAfter, m.AgentID, m.CreatedAt, m.UpdatedAt)
    return err
}

func (r *AgentMemoryRepo) GetByID(ctx context.Context, id string) (*agentmemory.AgentMemory, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, tenant_id, project, type, title, content, concepts, files,
               session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
        FROM agent_memories WHERE id = $1
    `, id)
    return scanMemory(row)
}

func (r *AgentMemoryRepo) ListLatestByType(ctx context.Context, tenantID, project, memType string) ([]agentmemory.AgentMemory, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, type, title, content, concepts, files,
               session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
        FROM agent_memories
        WHERE tenant_id = $1 AND ($2 = '' OR project = $2) AND type = $3 AND is_latest = TRUE
        ORDER BY version DESC
    `, tenantID, project, memType)
    if err != nil { return nil, err }
    return scanMemories(rows)
}

func (r *AgentMemoryRepo) ListLatestByProject(ctx context.Context, tenantID, project string) ([]agentmemory.AgentMemory, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, type, title, content, concepts, files,
               session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
        FROM agent_memories
        WHERE tenant_id = $1 AND ($2 = '' OR project = $2) AND is_latest = TRUE
    `, tenantID, project)
    if err != nil { return nil, err }
    return scanMemories(rows)
}

func (r *AgentMemoryRepo) ListAll(ctx context.Context) ([]agentmemory.AgentMemory, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, type, title, content, concepts, files,
               session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
        FROM agent_memories WHERE is_latest = TRUE
    `)
    if err != nil { return nil, err }
    return scanMemories(rows)
}

func (r *AgentMemoryRepo) SetNotLatest(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `UPDATE agent_memories SET is_latest = FALSE WHERE id = $1`, id)
    return err
}

func (r *AgentMemoryRepo) Delete(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM agent_memories WHERE id = $1`, id)
    return err
}

func (r *AgentMemoryRepo) FindExpired(ctx context.Context) ([]agentmemory.AgentMemory, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, type, title, content, concepts, files,
               session_ids, strength, version, is_latest, forget_after, agent_id, updated_at
        FROM agent_memories WHERE forget_after IS NOT NULL AND forget_after < NOW()
    `)
    if err != nil { return nil, err }
    return scanMemories(rows)
}

func (r *AgentMemoryRepo) UpdateStrength(ctx context.Context, id string, strength float64) error {
    _, err := r.db.Exec(ctx, `UPDATE agent_memories SET strength = $1, updated_at = NOW() WHERE id = $2`, strength, id)
    return err
}

func (r *AgentMemoryRepo) FlagForEviction(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `UPDATE agent_memories SET flagged_eviction = TRUE WHERE id = $1`, id)
    return err
}

func nilIfEmpty(s string) any { if s == "" { return nil }; return s }
```

### `adapter/repository/postgres/slots_repo.go`

```go
package postgres

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type SlotsRepo struct{ db *pgxpool.Pool }
func NewSlotsRepo(db *pgxpool.Pool) *SlotsRepo { return &SlotsRepo{db: db} }

func (r *SlotsRepo) GetSlot(ctx context.Context, tenantID, scope, label string) (*agentmemory.MemorySlot, error) {
    row := r.db.QueryRow(ctx, `
        SELECT tenant_id, project, scope, label, content, description, size_limit, pinned, read_only
        FROM memory_slots WHERE tenant_id = $1 AND scope = $2 AND label = $3
    `, tenantID, scope, label)
    var s agentmemory.MemorySlot
    err := row.Scan(&s.TenantID, &s.Project, &s.Scope, &s.Label, &s.Content,
        &s.Description, &s.SizeLimit, &s.Pinned, &s.ReadOnly)
    if err != nil { return nil, err }
    return &s, nil
}

func (r *SlotsRepo) CreateSlot(ctx context.Context, s agentmemory.MemorySlot) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO memory_slots (tenant_id, project, scope, label, content, description, size_limit, pinned, read_only)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, s.TenantID, s.Project, s.Scope, s.Label, s.Content, s.Description, s.SizeLimit, s.Pinned, s.ReadOnly)
    return err
}

func (r *SlotsRepo) UpdateSlot(ctx context.Context, s agentmemory.MemorySlot) error {
    _, err := r.db.Exec(ctx, `
        UPDATE memory_slots SET content = $1, updated_at = NOW()
        WHERE tenant_id = $2 AND scope = $3 AND label = $4
    `, s.Content, s.TenantID, s.Scope, s.Label)
    return err
}

func (r *SlotsRepo) DeleteSlot(ctx context.Context, tenantID, scope, label string) error {
    _, err := r.db.Exec(ctx, `DELETE FROM memory_slots WHERE tenant_id = $1 AND scope = $2 AND label = $3`, tenantID, scope, label)
    return err
}

func (r *SlotsRepo) ListSlots(ctx context.Context, tenantID, scope string) ([]agentmemory.MemorySlot, error) {
    rows, err := r.db.Query(ctx, `
        SELECT tenant_id, project, scope, label, description, size_limit, pinned, read_only
        FROM memory_slots WHERE tenant_id = $1 AND ($2 = '' OR scope = $2)
    `, tenantID, scope)
    if err != nil { return nil, err }
    defer rows.Close()
    var slots []agentmemory.MemorySlot
    for rows.Next() {
        var s agentmemory.MemorySlot
        rows.Scan(&s.TenantID, &s.Project, &s.Scope, &s.Label, &s.Description, &s.SizeLimit, &s.Pinned, &s.ReadOnly)
        slots = append(slots, s)
    }
    return slots, nil
}
```

### `adapter/grpc/agentmemory_handler.go`

```go
package grpc

import (
    "context"

    memorypb "github.com/vnp-memory/api/proto/memory/v1"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type AgentMemoryHandler struct {
    memorypb.UnimplementedAgentMemoryServiceServer
    rememberUC   *agentmemory.RememberAgentUseCase
    evictUC      *agentmemory.EvictUseCase
    autoForgetUC *agentmemory.AutoForgetUseCase
    retentionUC  *agentmemory.RetentionUseCase
    slotsUC      *agentmemory.SlotsUseCase
    memRepo      port.IMemoryRepo
}

func (h *AgentMemoryHandler) RememberAgent(ctx context.Context, req *memorypb.RememberAgentRequest) (*memorypb.RememberAgentResponse, error) {
    result, err := h.rememberUC.Execute(ctx, agentmemory.RememberRequest{
        TenantID: req.TenantId, Project: req.Project, Type: req.Type,
        Title: req.Title, Content: req.Content, Concepts: req.Concepts,
        Files: req.Files, SessionID: req.SessionId, Strength: req.Strength, AgentID: req.AgentId,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "remember: %v", err) }
    return &memorypb.RememberAgentResponse{
        MemoryId:   result.MemoryID,
        Version:    int32(result.Version),
        Superseded: result.Superseded,
    }, nil
}

func (h *AgentMemoryHandler) ListAgentMemories(ctx context.Context, req *memorypb.ListAgentMemoriesRequest) (*memorypb.ListAgentMemoriesResponse, error) {
    mems, err := h.memRepo.ListLatestByProject(ctx, req.TenantId, req.Project)
    if err != nil { return nil, status.Errorf(codes.Internal, "list: %v", err) }
    return mapListMemoriesResponse(mems), nil
}

func (h *AgentMemoryHandler) EvictMemories(ctx context.Context, req *memorypb.EvictMemoriesRequest) (*memorypb.EvictMemoriesResponse, error) {
    result, err := h.evictUC.Execute(ctx, agentmemory.EvictRequest{
        TenantID: req.TenantId, Project: req.Project,
        MaxMemories: int(req.MaxMemories), DryRun: req.DryRun,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "evict: %v", err) }
    return &memorypb.EvictMemoriesResponse{EvictedIds: result.EvictedIDs, DryRun: result.DryRun}, nil
}

func (h *AgentMemoryHandler) GetRetentionScore(ctx context.Context, req *memorypb.GetRetentionScoreRequest) (*memorypb.GetRetentionScoreResponse, error) {
    score, err := h.retentionUC.GetScore(ctx, req.MemoryId)
    if err != nil { return nil, status.Errorf(codes.NotFound, "memory not found") }
    return &memorypb.GetRetentionScoreResponse{
        Score: score.Score, RecencyFactor: score.RecencyFactor,
        FrequencyFactor: score.FrequencyFactor, ImportanceFactor: score.ImportanceFactor,
        DaysSinceAccess: score.DaysSinceAccess, RecommendAction: score.RecommendAction,
    }, nil
}

func (h *AgentMemoryHandler) WriteSlot(ctx context.Context, req *memorypb.WriteSlotRequest) (*memorypb.WriteSlotResponse, error) {
    if err := h.slotsUC.WriteSlot(ctx, agentmemory.WriteSlotRequest{
        TenantID: req.TenantId, Scope: req.Scope, Label: req.Label,
        Content: req.Content, Mode: req.Mode, Project: req.Project,
    }); err != nil {
        return nil, status.Errorf(codes.Internal, "write slot: %v", err)
    }
    return &memorypb.WriteSlotResponse{Ok: true}, nil
}
```

### MODIFY `bootstrap/memory.go` — AgentMemory init section

```go
// Thêm vào cuối InitMemoryService()

// [NEW] AgentMemory init
memRepo     := postgres.NewAgentMemoryRepo(db)
slotsRepo   := postgres.NewSlotsRepo(db)
searchClient := httpclient.NewSearchClient(cfg.ObserveSearch.URL)
publisher   := natevent.NewPublisher(nats, "agentmemory")

rememberUC    := agentmemory.NewRememberAgentUseCase(memRepo, searchClient, publisher)
evictUC       := agentmemory.NewEvictUseCase(memRepo, publisher)
autoForgetUC  := agentmemory.NewAutoForgetUseCase(memRepo, searchClient, publisher)
retentionUC   := agentmemory.NewRetentionUseCase(memRepo)
decayScheduler := agentmemory.NewDecayScheduler(memRepo, cfg.Memory.HalfLifeDays)
slotsUC       := agentmemory.NewSlotsUseCase(slotsRepo)

agentmemorypb.RegisterAgentMemoryServiceServer(grpcServer, grpchandler.NewAgentMemoryHandler(
    rememberUC, evictUC, autoForgetUC, retentionUC, slotsUC, memRepo,
))

go autoForgetUC.StartScheduler(context.Background())
go decayScheduler.Start(context.Background())
```

### MODIFY `gateway/router.go` — Add AgentMemory routes

```go
r.Post("/v1/memory/agent/remember",         h.ForwardTo("memory-service", "AgentMemoryService/RememberAgent"))
r.Get("/v1/memory/agent/list",              h.ForwardTo("memory-service", "AgentMemoryService/ListAgentMemories"))
r.Get("/v1/memory/agent/{id}",              h.ForwardTo("memory-service", "AgentMemoryService/GetAgentMemory"))
r.Delete("/v1/memory/agent/{id}",           h.ForwardTo("memory-service", "AgentMemoryService/DeleteAgentMemory"))
r.Get("/v1/memory/agent/{id}/retention",    h.ForwardTo("memory-service", "AgentMemoryService/GetRetentionScore"))
r.Post("/v1/memory/agent/evict",            h.ForwardTo("memory-service", "AgentMemoryService/EvictMemories"))
r.Post("/v1/memory/agent/auto-forget",      h.ForwardTo("memory-service", "AgentMemoryService/AutoForgetSweep"))

r.Get("/v1/memory/slots",                        h.ForwardTo("memory-service", "AgentMemoryService/ListSlots"))
r.Get("/v1/memory/slots/{scope}/{label}",         h.ForwardTo("memory-service", "AgentMemoryService/GetSlot"))
r.Post("/v1/memory/slots/{scope}/{label}",        h.ForwardTo("memory-service", "AgentMemoryService/WriteSlot"))
r.Delete("/v1/memory/slots/{scope}/{label}",      h.ForwardTo("memory-service", "AgentMemoryService/DeleteSlot"))
```

---

## Acceptance Criteria

| AC | Check |
|----|-------|
| `POST /v1/memory/agent/remember` → `{memory_id, version, superseded}` | ✅ |
| `GET /v1/memory/agent/{id}/retention` → `{score, recommend_action}` | ✅ |
| `POST /v1/memory/agent/evict?dry_run=true` → IDs, no DB change | ✅ |
| `POST /v1/memory/slots/project/myslot` → write slot | ✅ |
| `POST /v1/memory/slots/project/myslot?mode=append` → content appended | ✅ |

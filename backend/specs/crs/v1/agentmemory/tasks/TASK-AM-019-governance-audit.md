# TASK-AM-019 — Governance & Audit Trail

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-019 |
| **Wave** | 4 (Governance) |
| **Component** | `services/memory-service/internal/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-007 §2.1, §2.2 |
| **Priority** | Medium |
| **Depends On** | TASK-AM-006 |
| **Estimated** | 5h |

---

## Context

Governance Delete + Audit Trail cho AgentMemory. Cascade delete xóa memory khỏi: PostgreSQL, BM25 index, vector index, và graph (cognee/graphiti). Mọi operation đều tạo AuditEntry.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/memory-service/internal/domain/agentmemory/audit.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/governance_delete.go` |
| CREATE | `services/memory-service/internal/usecase/agentmemory/audit_list.go` |
| CREATE | `services/memory-service/internal/adapter/repository/postgres/audit_repo.go` |
| CREATE | `services/memory-service/internal/adapter/grpc/governance_handler.go` |
| MODIFY | `services/memory-service/internal/usecase/agentmemory/remember_agent.go` |
| MODIFY | `services/memory-service/internal/usecase/agentmemory/evict.go` |
| MODIFY | `services/memory-service/internal/usecase/agentmemory/auto_forget.go` |

---

## Implementation

### `domain/agentmemory/audit.go`

```go
package agentmemory

import (
    "time"
    "github.com/google/uuid"
)

type AuditEntry struct {
    ID          string
    Timestamp   time.Time
    Operation   string
    TargetIDs   []string
    PerformedBy string
    Project     string
    TenantID    string
    Details     map[string]any
    Reason      string
}

// Operation constants — 25 types
const (
    AuditObserve           = "observe"
    AuditRemember          = "remember"
    AuditSupersede         = "supersede"
    AuditForget            = "forget"
    AuditGovernanceDelete  = "governance_delete"
    AuditEvict             = "evict"
    AuditAutoForget        = "auto_forget"
    AuditCompress          = "compress"
    AuditSummarize         = "summarize"
    AuditConsolidate       = "consolidate"
    AuditSlotWrite         = "slot_write"
    AuditSlotDelete        = "slot_delete"
    AuditSessionStart      = "session_start"
    AuditSessionEnd        = "session_end"
    AuditSessionDelete     = "session_delete"
    AuditImportTranscript  = "import_transcript"
    AuditSearchQuery       = "search_query"
    AuditContextBuild      = "context_build"
    AuditSignalSend        = "signal_send"
    AuditLeaseAcquire      = "lease_acquire"
    AuditLeaseRelease      = "lease_release"
    AuditCheckpointCreate  = "checkpoint_create"
    AuditCheckpointResolve = "checkpoint_resolve"
    AuditSnapshotCreate    = "snapshot_create"
    AuditDecaySweep        = "decay_sweep"
)

func NewAuditEntry(tenantID, operation string, targetIDs []string) AuditEntry {
    return AuditEntry{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Operation: operation,
        TargetIDs: targetIDs,
        TenantID:  tenantID,
    }
}
```

### `usecase/agentmemory/governance_delete.go`

```go
package agentmemory

import (
    "context"
    "log"

    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
    "github.com/vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type GovernanceDeleteUseCase struct {
    memRepo      port.IMemoryRepo
    searchClient port.ISearchNotifier
    graphClient  port.IGraphClient  // optional: HTTP client to cognee/graphiti
    auditRepo    port.IAuditRepo
    publisher    port.IEventPublisher
}

type GovernanceDeleteRequest struct {
    MemoryID    string
    TenantID    string
    PerformedBy string
    Reason      string
}

func (uc *GovernanceDeleteUseCase) Execute(ctx context.Context, req GovernanceDeleteRequest) error {
    mem, err := uc.memRepo.GetByID(ctx, req.MemoryID)
    if err != nil { return agentmemory.ErrMemoryNotFound }

    // 1. Delete from PostgreSQL (cascade: related observations via FK)
    if err := uc.memRepo.Delete(ctx, req.MemoryID); err != nil { return err }

    // 2. Remove from BM25 index (non-fatal)
    if err := uc.searchClient.RemoveMemory(ctx, req.MemoryID); err != nil {
        log.Printf("[governance] BM25 index removal failed for %s: %v", req.MemoryID, err)
    }

    // 3. Remove from Vector index (non-fatal)
    if err := uc.searchClient.RemoveVector(ctx, req.MemoryID); err != nil {
        log.Printf("[governance] Vector index removal failed for %s: %v", req.MemoryID, err)
    }

    // 4. Remove graph edges (non-fatal; only if graphiti/cognee configured)
    if uc.graphClient != nil {
        if err := uc.graphClient.RemoveBySourceID(ctx, req.MemoryID); err != nil {
            log.Printf("[governance] Graph edge removal failed for %s: %v", req.MemoryID, err)
        }
    }

    // 5. Create audit entry
    entry := agentmemory.NewAuditEntry(req.TenantID, agentmemory.AuditGovernanceDelete, []string{req.MemoryID})
    entry.PerformedBy = req.PerformedBy
    entry.Project     = mem.Project
    entry.Reason      = req.Reason
    entry.Details     = map[string]any{"memory_type": string(mem.Type), "title": mem.Title}
    uc.auditRepo.Save(ctx, entry)

    // 6. Publish NATS event
    uc.publisher.Publish(ctx, "agentmemory.memory.governance_deleted", map[string]any{
        "memory_id": req.MemoryID, "tenant_id": req.TenantID, "performed_by": req.PerformedBy,
    })

    return nil
}
```

### `adapter/repository/postgres/audit_repo.go`

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/memory-service/internal/domain/agentmemory"
)

type AuditRepo struct{ db *pgxpool.Pool }
func NewAuditRepo(db *pgxpool.Pool) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Save(ctx context.Context, entry agentmemory.AuditEntry) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO audit_entries
            (id, tenant_id, timestamp, operation, target_ids, performed_by, project, details, reason)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `, entry.ID, entry.TenantID, entry.Timestamp, entry.Operation,
        pq.Array(entry.TargetIDs), entry.PerformedBy, entry.Project,
        marshalJSON(entry.Details), entry.Reason)
    return err
}

type AuditFilter struct {
    TenantID  string
    Operation string
    Project   string
    FromTime  *time.Time
    ToTime    *time.Time
    Limit     int
    Offset    int
}

func (r *AuditRepo) List(ctx context.Context, filter AuditFilter) ([]agentmemory.AuditEntry, error) {
    if filter.Limit == 0 { filter.Limit = 50 }
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, timestamp, operation, target_ids, performed_by, project, reason
        FROM audit_entries
        WHERE tenant_id = $1
          AND ($2 = '' OR operation = $2)
          AND ($3 = '' OR project = $3)
          AND ($4::TIMESTAMPTZ IS NULL OR timestamp >= $4)
          AND ($5::TIMESTAMPTZ IS NULL OR timestamp <= $5)
        ORDER BY timestamp DESC
        LIMIT $6 OFFSET $7
    `, filter.TenantID, filter.Operation, filter.Project, filter.FromTime, filter.ToTime,
        filter.Limit, filter.Offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var entries []agentmemory.AuditEntry
    for rows.Next() {
        var e agentmemory.AuditEntry
        rows.Scan(&e.ID, &e.TenantID, &e.Timestamp, &e.Operation,
            pq.Array(&e.TargetIDs), &e.PerformedBy, &e.Project, &e.Reason)
        entries = append(entries, e)
    }
    return entries, nil
}

func marshalJSON(v any) []byte {
    if v == nil { return []byte("{}") }
    b, _ := json.Marshal(v)
    return b
}
```

### `port/output.go` addition

```go
// Thêm vào port/output.go
type IAuditRepo interface {
    Save(ctx context.Context, entry agentmemory.AuditEntry) error
    List(ctx context.Context, filter agentmemory.AuditFilter) ([]agentmemory.AuditEntry, error)
}

type IGraphClient interface {
    RemoveBySourceID(ctx context.Context, sourceID string) error
}
```

### MODIFY `remember_agent.go` — Add audit on remember

```go
// Trong RememberAgentUseCase.Execute(), sau khi save:
uc.auditRepo.Save(ctx, agentmemory.AuditEntry{
    ID:        uuid.New().String(),
    Timestamp: time.Now(),
    Operation: agentmemory.AuditRemember,
    TargetIDs: []string{mem.ID},
    TenantID:  req.TenantID,
    Project:   req.Project,
    Details:   map[string]any{"type": req.Type, "version": newVersion},
})
```

### MODIFY `evict.go` — Add audit on evict

```go
// Trong EvictUseCase.Execute(), sau evict:
if !req.DryRun && len(evictedIDs) > 0 {
    uc.auditRepo.Save(ctx, agentmemory.AuditEntry{
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Operation: agentmemory.AuditEvict,
        TargetIDs: evictedIDs,
        TenantID:  req.TenantID,
        Project:   req.Project,
    })
}
```

### `adapter/grpc/governance_handler.go`

```go
package grpc

import (
    "context"

    memorypb "github.com/vnp-memory/api/proto/memory/v1"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type GovernanceHandler struct {
    memorypb.UnimplementedGovernanceServiceServer
    deleteUC  *agentmemory.GovernanceDeleteUseCase
    auditRepo port.IAuditRepo
}

func (h *GovernanceHandler) Delete(ctx context.Context, req *memorypb.GovernanceDeleteRequest) (*memorypb.GovernanceDeleteResponse, error) {
    err := h.deleteUC.Execute(ctx, agentmemory.GovernanceDeleteRequest{
        MemoryID:    req.MemoryId,
        TenantID:    req.TenantId,
        PerformedBy: req.PerformedBy,
        Reason:      req.Reason,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "governance delete: %v", err) }
    return &memorypb.GovernanceDeleteResponse{Deleted: true}, nil
}

func (h *GovernanceHandler) ListAudit(ctx context.Context, req *memorypb.ListAuditRequest) (*memorypb.ListAuditResponse, error) {
    entries, err := h.auditRepo.List(ctx, agentmemory.AuditFilter{
        TenantID:  req.TenantId,
        Operation: req.Operation,
        Project:   req.Project,
        Limit:     int(req.Limit),
        Offset:    int(req.Offset),
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "list audit: %v", err) }
    return mapAuditResponse(entries), nil
}
```

---

## Verification

```bash
# 1. Create memory
memory_id=$(curl -s -X POST http://localhost:8080/v1/memory/agent/remember \
  -H 'Content-Type: application/json' \
  -d '{"type":"fact","title":"Test","content":"Test","concepts":["test"]}' | jq -r '.memory_id')

# 2. Governance delete
curl -s -X DELETE "http://localhost:8080/v1/memory/agent/$memory_id/governance" \
  -d '{"reason":"GDPR request","performed_by":"admin"}' | jq .

# 3. Verify audit entry created
curl -s "http://localhost:8080/v1/memory/audit?operation=governance_delete" | jq .

# 4. Verify memory gone from search
curl -s -X POST http://localhost:8080/v1/observe/search/smart \
  -d '{"query":"Test fact"}' | jq '.results | length'
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `DELETE /v1/memory/agent/{id}/governance` → deleted from PostgreSQL | ✅ |
| Governance delete → BM25 + vector indexes updated | ✅ |
| Governance delete → AuditEntry created with operation=governance_delete | ✅ |
| `GET /v1/memory/audit?operation=governance_delete` → entry returned | ✅ |
| Remember → AuditEntry created with operation=remember | ✅ |
| Evict → AuditEntry created with all evicted_ids | ✅ |
| `GET /v1/memory/audit?project=myproject` → filtered by project | ✅ |

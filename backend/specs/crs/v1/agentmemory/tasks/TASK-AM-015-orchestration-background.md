# TASK-AM-015 — Orchestration Background Sweeper + gRPC Handler + Bootstrap

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-015 |
| **Wave** | 3 (Orchestration) |
| **Component** | `services/orchestration-service/` + `apps/memory/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-004 §2.6, §2.7, §2.8, §2.9 |
| **Priority** | High |
| **Depends On** | TASK-AM-014 |
| **Estimated** | 4h |

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/orchestration-service/internal/adapter/background/sweeper.go` |
| CREATE | `services/orchestration-service/internal/adapter/grpc/handler.go` |
| CREATE | `services/orchestration-service/internal/adapter/repository/postgres/repos.go` |
| CREATE | `apps/memory/internal/bootstrap/orchestration.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### `adapter/background/sweeper.go`

```go
package background

import (
    "context"
    "log"
    "time"

    "github.com/vnp-memory/services/orchestration-service/internal/orchestration"
)

type BackgroundSweeper struct {
    leases      *orchestration.LeaseService
    signals     *orchestration.SignalService
    sentinels   *orchestration.SentinelService
    sketches    *orchestration.SketchService
    checkpoints *orchestration.CheckpointService
}

func NewBackgroundSweeper(
    leases *orchestration.LeaseService,
    signals *orchestration.SignalService,
    sentinels *orchestration.SentinelService,
    sketches *orchestration.SketchService,
    checkpoints *orchestration.CheckpointService,
) *BackgroundSweeper {
    return &BackgroundSweeper{leases, signals, sentinels, sketches, checkpoints}
}

func (s *BackgroundSweeper) Start(ctx context.Context) {
    log.Println("[orchestration] background sweeper started")
    go s.runEvery(ctx, 60*time.Second,  func() { s.leases.SweepExpired(ctx) })
    go s.runEvery(ctx, 300*time.Second, func() { s.signals.DeleteExpired(ctx) })
    go s.runEvery(ctx, 30*time.Second,  func() { s.sentinels.EvaluateAll(ctx) })
    go s.runEvery(ctx, 1*time.Hour,     func() { s.sketches.ReapExpired(ctx) })
    go s.runEvery(ctx, 1*time.Hour,     func() { s.checkpoints.AutoRejectExpired(ctx) })
}

func (s *BackgroundSweeper) runEvery(ctx context.Context, interval time.Duration, fn func()) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C: fn()
        case <-ctx.Done(): return
        }
    }
}
```

### `adapter/grpc/handler.go`

```go
package grpc

import (
    "context"
    "time"

    orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
    "github.com/vnp-memory/services/orchestration-service/internal/orchestration"
    "github.com/vnp-memory/services/orchestration-service/internal/usecase/port"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type OrchestrationHandler struct {
    orchpb.UnimplementedOrchestrationServiceServer
    actions     *orchestration.ActionService
    leases      *orchestration.LeaseService
    signals     *orchestration.SignalService
    checkpoints *orchestration.CheckpointService
    sketches    *orchestration.SketchService
    sentinels   *orchestration.SentinelService
    routines    *orchestration.RoutineService
    actionRepo  port.IActionRepo
    signalRepo  port.ISignalRepo
    crystalRepo port.ICrystalRepo
}

// ── Actions ───────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) CreateAction(ctx context.Context, req *orchpb.CreateActionRequest) (*orchpb.CreateActionResponse, error) {
    action, err := h.actions.Create(ctx, orchestration.CreateActionRequest{
        TenantID: req.TenantId, Project: req.Project, AgentID: req.AgentId,
        Title: req.Title, Description: req.Description, Priority: int(req.Priority),
        Requires: req.Requires, ConflictsWith: req.ConflictsWith, Tags: req.Tags,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "create action: %v", err) }
    return &orchpb.CreateActionResponse{ActionId: action.ID}, nil
}

func (h *OrchestrationHandler) GetAction(ctx context.Context, req *orchpb.GetActionRequest) (*orchpb.GetActionResponse, error) {
    action, err := h.actionRepo.Get(ctx, req.ActionId)
    if err != nil { return nil, status.Errorf(codes.NotFound, "action not found") }
    return &orchpb.GetActionResponse{Action: mapAction(action)}, nil
}

func (h *OrchestrationHandler) ListActions(ctx context.Context, req *orchpb.ListActionsRequest) (*orchpb.ListActionsResponse, error) {
    actions, err := h.actionRepo.List(ctx, req.TenantId, req.Status, int(req.Limit))
    if err != nil { return nil, status.Errorf(codes.Internal, "list actions: %v", err) }
    return mapListActionsResponse(actions), nil
}

func (h *OrchestrationHandler) UpdateAction(ctx context.Context, req *orchpb.UpdateActionRequest) (*orchpb.UpdateActionResponse, error) {
    err := h.actions.UpdateStatus(ctx, req.ActionId, domain.ActionStatus(req.Status), req.Result)
    if err != nil {
        if errors.Is(err, domain.ErrInvalidTransition) {
            return nil, status.Errorf(codes.InvalidArgument, "invalid transition: %v", err)
        }
        return nil, status.Errorf(codes.Internal, "update action: %v", err)
    }
    return &orchpb.UpdateActionResponse{Ok: true}, nil
}

// ── Leases ────────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) AcquireLease(ctx context.Context, req *orchpb.AcquireLeaseRequest) (*orchpb.AcquireLeaseResponse, error) {
    ttl := time.Duration(req.TtlSecs) * time.Second
    if ttl == 0 { ttl = 5 * time.Minute }
    lease, err := h.leases.Acquire(ctx, req.ActionId, req.AgentId, ttl)
    if err != nil {
        var ce domain.ErrLeaseConflictDetail
        if errors.As(err, &ce) {
            return &orchpb.AcquireLeaseResponse{Conflict: true, ConflictingAgent: ce.ActiveLease.AgentID}, nil
        }
        return nil, status.Errorf(codes.Internal, "acquire lease: %v", err)
    }
    return &orchpb.AcquireLeaseResponse{LeaseId: lease.ID}, nil
}

func (h *OrchestrationHandler) RenewLease(ctx context.Context, req *orchpb.RenewLeaseRequest) (*orchpb.RenewLeaseResponse, error) {
    err := h.leases.Renew(ctx, req.LeaseId, time.Duration(req.ExtendSecs)*time.Second)
    if err != nil { return nil, status.Errorf(codes.Internal, "renew lease: %v", err) }
    return &orchpb.RenewLeaseResponse{Ok: true}, nil
}

func (h *OrchestrationHandler) ReleaseLease(ctx context.Context, req *orchpb.ReleaseLeaseRequest) (*orchpb.ReleaseLeaseResponse, error) {
    err := h.leases.Release(ctx, req.LeaseId)
    if err != nil { return nil, status.Errorf(codes.Internal, "release lease: %v", err) }
    return &orchpb.ReleaseLeaseResponse{Ok: true}, nil
}

// ── Signals ───────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) SendSignal(ctx context.Context, req *orchpb.SendSignalRequest) (*orchpb.SendSignalResponse, error) {
    signal, err := h.signals.Send(ctx, req)
    if err != nil { return nil, status.Errorf(codes.Internal, "send signal: %v", err) }
    return &orchpb.SendSignalResponse{SignalId: signal.ID}, nil
}

func (h *OrchestrationHandler) ListSignals(ctx context.Context, req *orchpb.ListSignalsRequest) (*orchpb.ListSignalsResponse, error) {
    signals, err := h.signalRepo.List(ctx, req.TenantId, req.AgentId, req.UnreadOnly)
    if err != nil { return nil, status.Errorf(codes.Internal, "list signals: %v", err) }
    return mapListSignalsResponse(signals), nil
}

// ── Checkpoints ───────────────────────────────────────────────────────────

func (h *OrchestrationHandler) CreateCheckpoint(ctx context.Context, req *orchpb.CreateCheckpointRequest) (*orchpb.CreateCheckpointResponse, error) {
    cp, err := h.checkpoints.Create(ctx, req)
    if err != nil { return nil, status.Errorf(codes.Internal, "create checkpoint: %v", err) }
    return &orchpb.CreateCheckpointResponse{CheckpointId: cp.ID}, nil
}

func (h *OrchestrationHandler) ApproveCheckpoint(ctx context.Context, req *orchpb.ApproveCheckpointRequest) (*orchpb.ApproveCheckpointResponse, error) {
    err := h.checkpoints.Resolve(ctx, req.CheckpointId, "approved", req.ApprovedBy, "")
    if err != nil { return nil, status.Errorf(codes.Internal, "approve: %v", err) }
    return &orchpb.ApproveCheckpointResponse{Ok: true}, nil
}

// ── Sketches & Crystals ───────────────────────────────────────────────────

func (h *OrchestrationHandler) PromoteSketch(ctx context.Context, req *orchpb.PromoteSketchRequest) (*orchpb.PromoteSketchResponse, error) {
    crystal, err := h.sketches.Promote(ctx, req.SketchId)
    if err != nil { return nil, status.Errorf(codes.Internal, "promote sketch: %v", err) }
    return &orchpb.PromoteSketchResponse{CrystalId: crystal.ID}, nil
}

func (h *OrchestrationHandler) GetCrystal(ctx context.Context, req *orchpb.GetCrystalRequest) (*orchpb.GetCrystalResponse, error) {
    crystal, err := h.crystalRepo.Get(ctx, req.CrystalId)
    if err != nil { return nil, status.Errorf(codes.NotFound, "crystal not found") }
    return mapCrystalResponse(crystal), nil
}
```

### `apps/memory/internal/bootstrap/orchestration.go`

```go
package bootstrap

func InitOrchestration(reg *bus.InProcessRegistry, db *pgxpool.Pool, nc *nats.Conn, cfg *config.Config) {
    repos := orchrepo.NewPostgresRepos(db)
    publisher := natevent.NewPublisher(nc, "agentmemory")

    var llm port.ILLMProvider
    if cfg.Bifrost.URL != "" {
        llm = bifrost.NewLLMClient(cfg.Bifrost.URL)
    }

    actionSvc     := orchestration.NewActionService(repos.Actions, publisher)
    leaseSvc      := orchestration.NewLeaseService(repos.Leases, publisher)
    signalSvc     := orchestration.NewSignalService(repos.Signals, publisher)
    checkpointSvc := orchestration.NewCheckpointService(repos.Checkpoints, publisher)
    routineSvc    := orchestration.NewRoutineService(repos.Routines)
    sentinelSvc   := orchestration.NewSentinelService(repos.Sentinels, repos.Actions, publisher)
    sketchSvc     := orchestration.NewSketchService(repos.Sketches, repos.Actions, repos.Crystals, llm)

    handler := grpchandler.NewOrchestrationHandler(
        actionSvc, leaseSvc, signalSvc, checkpointSvc, routineSvc, sentinelSvc, sketchSvc,
        repos.Actions, repos.Signals, repos.Crystals,
    )

    grpcServer := grpc.NewServer()
    orchpb.RegisterOrchestrationServiceServer(grpcServer, handler)
    reg.Register("am-orchestration", grpcServer)

    sweeper := background.NewBackgroundSweeper(leaseSvc, signalSvc, sentinelSvc, sketchSvc, checkpointSvc)
    go sweeper.Start(context.Background())
}
```

### MODIFY `gateway/router.go` — Orchestration routes

```go
// Actions
r.Post("/v1/orchestration/actions",                   h.ForwardTo("am-orchestration", "OrchestrationService/CreateAction"))
r.Get("/v1/orchestration/actions",                    h.ForwardTo("am-orchestration", "OrchestrationService/ListActions"))
r.Get("/v1/orchestration/actions/{id}",               h.ForwardTo("am-orchestration", "OrchestrationService/GetAction"))
r.Patch("/v1/orchestration/actions/{id}",             h.ForwardTo("am-orchestration", "OrchestrationService/UpdateAction"))
r.Delete("/v1/orchestration/actions/{id}",            h.ForwardTo("am-orchestration", "OrchestrationService/DeleteAction"))

// Leases
r.Post("/v1/orchestration/leases/acquire",            h.ForwardTo("am-orchestration", "OrchestrationService/AcquireLease"))
r.Post("/v1/orchestration/leases/renew",              h.ForwardTo("am-orchestration", "OrchestrationService/RenewLease"))
r.Post("/v1/orchestration/leases/release",            h.ForwardTo("am-orchestration", "OrchestrationService/ReleaseLease"))
r.Get("/v1/orchestration/leases/{actionId}",          h.ForwardTo("am-orchestration", "OrchestrationService/GetLease"))

// Signals
r.Post("/v1/orchestration/signals/send",              h.ForwardTo("am-orchestration", "OrchestrationService/SendSignal"))
r.Get("/v1/orchestration/signals",                    h.ForwardTo("am-orchestration", "OrchestrationService/ListSignals"))
r.Post("/v1/orchestration/signals/{id}/read",         h.ForwardTo("am-orchestration", "OrchestrationService/MarkSignalRead"))
r.Delete("/v1/orchestration/signals/{id}",            h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSignal"))

// Checkpoints
r.Post("/v1/orchestration/checkpoints",               h.ForwardTo("am-orchestration", "OrchestrationService/CreateCheckpoint"))
r.Get("/v1/orchestration/checkpoints",                h.ForwardTo("am-orchestration", "OrchestrationService/ListCheckpoints"))
r.Post("/v1/orchestration/checkpoints/{id}/approve",  h.ForwardTo("am-orchestration", "OrchestrationService/ApproveCheckpoint"))
r.Post("/v1/orchestration/checkpoints/{id}/reject",   h.ForwardTo("am-orchestration", "OrchestrationService/RejectCheckpoint"))

// Sentinels
r.Post("/v1/orchestration/sentinels",                 h.ForwardTo("am-orchestration", "OrchestrationService/CreateSentinel"))
r.Get("/v1/orchestration/sentinels",                  h.ForwardTo("am-orchestration", "OrchestrationService/ListSentinels"))
r.Delete("/v1/orchestration/sentinels/{id}",          h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSentinel"))

// Sketches & Crystals
r.Post("/v1/orchestration/sketches",                  h.ForwardTo("am-orchestration", "OrchestrationService/CreateSketch"))
r.Get("/v1/orchestration/sketches",                   h.ForwardTo("am-orchestration", "OrchestrationService/ListSketches"))
r.Post("/v1/orchestration/sketches/{id}/add-action",  h.ForwardTo("am-orchestration", "OrchestrationService/AddActionToSketch"))
r.Post("/v1/orchestration/sketches/{id}/promote",     h.ForwardTo("am-orchestration", "OrchestrationService/PromoteSketch"))
r.Get("/v1/orchestration/crystals",                   h.ForwardTo("am-orchestration", "OrchestrationService/ListCrystals"))
r.Get("/v1/orchestration/crystals/{id}",              h.ForwardTo("am-orchestration", "OrchestrationService/GetCrystal"))

// Routines
r.Post("/v1/orchestration/routines",                  h.ForwardTo("am-orchestration", "OrchestrationService/CreateRoutine"))
r.Get("/v1/orchestration/routines",                   h.ForwardTo("am-orchestration", "OrchestrationService/ListRoutines"))
r.Post("/v1/orchestration/routines/{id}/execute",     h.ForwardTo("am-orchestration", "OrchestrationService/ExecuteRoutine"))
```

---

## Acceptance Criteria

| AC | Check |
|----|-------|
| `POST /v1/orchestration/actions` → `{action_id}` | ✅ |
| `POST /v1/orchestration/leases/acquire` → `{lease_id}` | ✅ |
| Second acquire in TTL → `{conflict: true, conflicting_agent}` | ✅ |
| `POST /v1/orchestration/signals/send` → `{signal_id}` | ✅ |
| `POST /v1/orchestration/sketches/{id}/promote` → `{crystal_id}` | ✅ |
| Background sweeper: leases swept every 60s | ✅ |
| `am-orchestration` registered in InProcessRegistry | ✅ |

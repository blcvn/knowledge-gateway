# Solution: SOL-004 — Multi-Agent Orchestration Layer

**CR ID:** CR-AM-004  
**Solution ID:** SOL-004  
**Priority:** High (Wave 3)  
**Architecture:** NEW `services/orchestration-service/` + PostgreSQL

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- VNP Memory hiện không có coordination layer cho multi-agent.
- `services/pipeline-service/` có `Pipeline`, `Job`, `Queue`, `Worker` — nhưng đây là internal pipeline management, không phải inter-agent orchestration.
- NATS JetStream embedded — sẵn sàng cho signal delivery.
- PostgreSQL sẵn sàng cho persistent state.

**Chiến lược:** Tạo service mới `services/orchestration-service/` — service thứ 38. Dùng **PostgreSQL** cho persistent state (Actions, Leases, Signals, v.v.) thay vì SQLite để đồng nhất với VNP Memory storage strategy.

---

## 2. Giải pháp

### 2.1. [NEW] `services/orchestration-service/`

```
services/orchestration-service/
├── cmd/orchestration/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go           # Action, Lease, Signal, Routine, Checkpoint, Sentinel, Sketch, Crystal
│   │   ├── value_object.go     # ActionStatus, SignalType, CheckpointStatus, SentinelCondition
│   │   └── errors.go           # ErrLeaseConflict, ErrInvalidTransition, ErrExpired
│   ├── orchestration/
│   │   ├── actions.go          # Action CRUD + state machine validation
│   │   ├── leases.go           # Distributed lock (keyed mutex + PostgreSQL)
│   │   ├── signals.go          # Inter-agent messaging
│   │   ├── routines.go         # Workflow template execution
│   │   ├── checkpoints.go      # Approval gate lifecycle
│   │   ├── sentinels.go        # Event watcher + condition evaluation
│   │   └── sketches.go         # Sketch grouping + Crystal promotion
│   ├── usecase/
│   │   ├── port/
│   │   │   ├── input.go        # IOrchestrationUseCase interfaces
│   │   │   └── output.go       # IOrchStore, ILLMProvider, IEventPublisher
│   │   └── ... (one per entity)
│   ├── adapter/
│   │   ├── grpc/handler.go
│   │   ├── repository/
│   │   │   └── postgres/       # All entity repos
│   │   ├── event/publisher.go  # NATS: agentmemory.action.*, agentmemory.signal.*, etc.
│   │   └── background/
│   │       └── sweeper.go      # Background cleanup jobs
└── api/proto/orchestration/v1/orchestration.proto
```

### 2.2. State Machine — Actions

```go
// internal/domain/entity.go

var validTransitions = map[ActionStatus][]ActionStatus{
    ActionPending:   {ActionActive, ActionBlocked, ActionCancelled},
    ActionActive:    {ActionDone, ActionBlocked, ActionCancelled, ActionFailed},
    ActionBlocked:   {ActionActive, ActionCancelled},
    ActionDone:      {},   // terminal
    ActionCancelled: {},   // terminal
    ActionFailed:    {},   // terminal
}

func (current ActionStatus) CanTransitionTo(next ActionStatus) bool {
    allowed := validTransitions[current]
    for _, a := range allowed { if a == next { return true } }
    return false
}
```

### 2.3. Lease Service (Distributed Locking)

```go
// internal/orchestration/leases.go

type LeaseService struct {
    repo     port.ILeaseRepo
    mu       sync.Map          // per-actionId in-process mutex (single-node fast path)
    publisher port.IEventPublisher
}

func (s *LeaseService) Acquire(ctx context.Context, req AcquireLeaseRequest) (*Lease, error) {
    // Check PostgreSQL for existing active lease (multi-node safe)
    existing, _ := s.repo.GetActiveLease(ctx, req.ActionID)
    if existing != nil && time.Now().Before(existing.ExpiresAt) {
        return nil, ErrLeaseConflict{ActiveLease: existing}
    }
    
    lease := Lease{
        ID:         newID(),
        ActionID:   req.ActionID,
        AgentID:    req.AgentID,
        AcquiredAt: time.Now(),
        ExpiresAt:  time.Now().Add(req.TTL),
        Status:     "active",
    }
    if err := s.repo.Save(ctx, lease); err != nil { return nil, err }
    
    s.publisher.Publish(ctx, "agentmemory.lease.acquired", LeaseEvent{LeaseID: lease.ID, ActionID: req.ActionID})
    return &lease, nil
}

func (s *LeaseService) Renew(ctx context.Context, leaseID string, extension time.Duration) error {
    return s.repo.ExtendExpiry(ctx, leaseID, extension)
}

func (s *LeaseService) Release(ctx context.Context, leaseID string) error {
    return s.repo.SetStatus(ctx, leaseID, "released")
}

// Background: runs every 60s to mark expired leases
func (s *LeaseService) SweepExpired(ctx context.Context) {
    expired, _ := s.repo.FindExpired(ctx)
    for _, l := range expired {
        s.repo.SetStatus(ctx, l.ID, "expired")
        s.publisher.Publish(ctx, "agentmemory.lease.expired", LeaseEvent{LeaseID: l.ID, ActionID: l.ActionID})
    }
}
```

### 2.4. Sentinel Condition Evaluation

```go
// internal/orchestration/sentinels.go

type SentinelService struct {
    repo      port.ISentinelRepo
    actionRepo port.IActionRepo
    publisher port.IEventPublisher
}

// Runs every 30s — check all "watching" sentinels
func (s *SentinelService) EvaluateAll(ctx context.Context) {
    sentinels, _ := s.repo.ListWatching(ctx)
    for _, sentinel := range sentinels {
        if s.conditionMet(ctx, sentinel.Condition) {
            s.trigger(ctx, sentinel)
        }
    }
}

func (s *SentinelService) conditionMet(ctx context.Context, cond SentinelCondition) bool {
    switch cond.Type {
    case "action_done":
        action, _ := s.actionRepo.Get(ctx, cond.Target)
        return action != nil && action.Status == ActionDone
    case "signal_received":
        // Check if signal with matching type received by cond.Target
        ...
    case "time":
        // Cron expression — check if current time matches
        return cronMatches(cond.Target, time.Now())
    }
    return false
}

func (s *SentinelService) trigger(ctx context.Context, sentinel Sentinel) {
    s.repo.SetStatus(ctx, sentinel.ID, "triggered")
    if sentinel.ActionID != "" {
        s.actionRepo.TransitionStatus(ctx, sentinel.ActionID, ActionPending, ActionActive)
    }
    if sentinel.SignalTo != "" {
        // Auto-send signal to agent
    }
    s.publisher.Publish(ctx, "agentmemory.sentinel.triggered", SentinelEvent{SentinelID: sentinel.ID})
}
```

### 2.5. Crystal Promotion (Sketch → Crystal)

```go
// internal/orchestration/sketches.go

type SketchService struct {
    repo        port.ISketchRepo
    actionRepo  port.IActionRepo
    crystalRepo port.ICrystalRepo
    llm         port.ILLMProvider
}

func (s *SketchService) Promote(ctx context.Context, sketchID string) (*Crystal, error) {
    sketch, _ := s.repo.Get(ctx, sketchID)
    
    // Collect all action results
    var actionSummaries []string
    for _, aid := range sketch.ActionIDs {
        a, _ := s.actionRepo.Get(ctx, aid)
        if a.Result != "" { actionSummaries = append(actionSummaries, a.Title+": "+a.Result) }
    }
    
    var crystal Crystal
    crystal.SourceActionIDs = sketch.ActionIDs
    
    if s.llm != nil {
        // LLM generate crystal
        prompt := buildCrystalPrompt(actionSummaries)
        resp, err := s.llm.Generate(ctx, prompt)
        if err == nil {
            json.Unmarshal([]byte(resp), &crystal)
        }
    }
    
    // Graceful degrade: synthetic crystal from action titles
    if crystal.Narrative == "" {
        crystal.Narrative = strings.Join(actionSummaries, "; ")
        crystal.KeyOutcomes = extractOutcomes(sketch.ActionIDs, s.actionRepo)
    }
    
    crystal.ID = newID()
    crystal.CreatedAt = time.Now()
    s.crystalRepo.Save(ctx, crystal)
    s.repo.SetStatus(ctx, sketchID, "promoted")
    
    return &crystal, nil
}
```

### 2.6. Background Jobs

```go
// internal/adapter/background/sweeper.go

type BackgroundSweeper struct {
    leases      *orchestration.LeaseService
    signals     *orchestration.SignalService
    sentinels   *orchestration.SentinelService
    sketches    *orchestration.SketchService
    checkpoints *orchestration.CheckpointService
}

func (s *BackgroundSweeper) Start(ctx context.Context) {
    go s.runEvery(ctx, 60*time.Second,  s.leases.SweepExpired)
    go s.runEvery(ctx, 300*time.Second, s.signals.DeleteExpired)
    go s.runEvery(ctx, 30*time.Second,  s.sentinels.EvaluateAll)
    go s.runEvery(ctx, 1*time.Hour,     s.sketches.ReapExpired)
    go s.runEvery(ctx, 1*time.Hour,     s.checkpoints.AutoRejectExpired)
}
```

### 2.7. gRPC Proto

```protobuf
// api/proto/orchestration/v1/orchestration.proto

service OrchestrationService {
  // Actions
  rpc CreateAction(CreateActionRequest) returns (CreateActionResponse);
  rpc GetAction(GetActionRequest) returns (GetActionResponse);
  rpc ListActions(ListActionsRequest) returns (ListActionsResponse);
  rpc UpdateAction(UpdateActionRequest) returns (UpdateActionResponse);
  rpc DeleteAction(DeleteActionRequest) returns (DeleteActionResponse);

  // Leases
  rpc AcquireLease(AcquireLeaseRequest) returns (AcquireLeaseResponse);
  rpc RenewLease(RenewLeaseRequest) returns (RenewLeaseResponse);
  rpc ReleaseLease(ReleaseLeaseRequest) returns (ReleaseLeaseResponse);
  rpc GetLease(GetLeaseRequest) returns (GetLeaseResponse);

  // Signals
  rpc SendSignal(SendSignalRequest) returns (SendSignalResponse);
  rpc ListSignals(ListSignalsRequest) returns (ListSignalsResponse);
  rpc MarkSignalRead(MarkSignalReadRequest) returns (MarkSignalReadResponse);
  rpc DeleteSignal(DeleteSignalRequest) returns (DeleteSignalResponse);

  // Routines
  rpc CreateRoutine(CreateRoutineRequest) returns (CreateRoutineResponse);
  rpc ListRoutines(ListRoutinesRequest) returns (ListRoutinesResponse);
  rpc ExecuteRoutine(ExecuteRoutineRequest) returns (ExecuteRoutineResponse);

  // Checkpoints
  rpc CreateCheckpoint(CreateCheckpointRequest) returns (CreateCheckpointResponse);
  rpc ListCheckpoints(ListCheckpointsRequest) returns (ListCheckpointsResponse);
  rpc ApproveCheckpoint(ApproveCheckpointRequest) returns (ApproveCheckpointResponse);
  rpc RejectCheckpoint(RejectCheckpointRequest) returns (RejectCheckpointResponse);

  // Sentinels
  rpc CreateSentinel(CreateSentinelRequest) returns (CreateSentinelResponse);
  rpc ListSentinels(ListSentinelsRequest) returns (ListSentinelsResponse);
  rpc DeleteSentinel(DeleteSentinelRequest) returns (DeleteSentinelResponse);

  // Sketches & Crystals
  rpc CreateSketch(CreateSketchRequest) returns (CreateSketchResponse);
  rpc ListSketches(ListSketchesRequest) returns (ListSketchesResponse);
  rpc AddActionToSketch(AddActionToSketchRequest) returns (AddActionToSketchResponse);
  rpc PromoteSketch(PromoteSketchRequest) returns (PromoteSketchResponse);
  rpc ListCrystals(ListCrystalsRequest) returns (ListCrystalsResponse);
  rpc GetCrystal(GetCrystalRequest) returns (GetCrystalResponse);
}
```

### 2.8. PostgreSQL Schema

```sql
-- Migration: 0012_orchestration.up.sql

CREATE TABLE agent_actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    project         TEXT,
    agent_id        TEXT,
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    priority        INT NOT NULL DEFAULT 50,
    requires        UUID[] DEFAULT '{}',
    conflicts_with  UUID[] DEFAULT '{}',
    tags            TEXT[] DEFAULT '{}',
    result          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE TABLE agent_leases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id   UUID NOT NULL REFERENCES agent_actions(id),
    agent_id    TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',  -- active | expired | released
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    renewed_at  TIMESTAMPTZ
);

CREATE TABLE agent_signals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    from_agent  TEXT NOT NULL,
    to_agent    TEXT NOT NULL,
    signal_type TEXT NOT NULL,  -- handoff | update | cancel | request | response | alert
    content     TEXT,
    thread_id   TEXT,
    reply_to    UUID REFERENCES agent_signals(id),
    is_read     BOOLEAN DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agent_routines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    name        TEXT NOT NULL,
    description TEXT,
    steps       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agent_checkpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    agent_id    TEXT,
    action_id   UUID REFERENCES agent_actions(id),
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',
    approved_by TEXT,
    rejected_by TEXT,
    reason      TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE agent_sentinels (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    name            TEXT NOT NULL,
    condition       JSONB NOT NULL,  -- {type, target, value}
    action_id       UUID REFERENCES agent_actions(id),
    signal_to       TEXT,
    status          TEXT NOT NULL DEFAULT 'watching',
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at    TIMESTAMPTZ
);

CREATE TABLE agent_sketches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT,
    session_id  TEXT,
    title       TEXT NOT NULL,
    action_ids  UUID[] DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'active',
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agent_crystals (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        TEXT NOT NULL,
    source_action_ids UUID[] DEFAULT '{}',
    narrative        TEXT NOT NULL,
    key_outcomes     TEXT[] DEFAULT '{}',
    files_affected   TEXT[] DEFAULT '{}',
    lessons          TEXT[] DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_actions_tenant_status ON agent_actions(tenant_id, status);
CREATE INDEX idx_agent_leases_action_status ON agent_leases(action_id, status);
CREATE INDEX idx_agent_signals_to_agent_unread ON agent_signals(to_agent, is_read) WHERE NOT is_read;
CREATE INDEX idx_agent_checkpoints_tenant_status ON agent_checkpoints(tenant_id, status);
CREATE INDEX idx_agent_sentinels_status ON agent_sentinels(status) WHERE status = 'watching';
```

### 2.9. Gateway Routes

```go
// Orchestration routes
r.Post("/v1/orchestration/actions",                       h.ForwardTo("am-orchestration", "OrchestrationService/CreateAction"))
r.Get("/v1/orchestration/actions",                        h.ForwardTo("am-orchestration", "OrchestrationService/ListActions"))
r.Get("/v1/orchestration/actions/{id}",                   h.ForwardTo("am-orchestration", "OrchestrationService/GetAction"))
r.Patch("/v1/orchestration/actions/{id}",                 h.ForwardTo("am-orchestration", "OrchestrationService/UpdateAction"))
r.Delete("/v1/orchestration/actions/{id}",                h.ForwardTo("am-orchestration", "OrchestrationService/DeleteAction"))

r.Post("/v1/orchestration/leases/acquire",                h.ForwardTo("am-orchestration", "OrchestrationService/AcquireLease"))
r.Post("/v1/orchestration/leases/renew",                  h.ForwardTo("am-orchestration", "OrchestrationService/RenewLease"))
r.Post("/v1/orchestration/leases/release",                h.ForwardTo("am-orchestration", "OrchestrationService/ReleaseLease"))
r.Get("/v1/orchestration/leases/{actionId}",              h.ForwardTo("am-orchestration", "OrchestrationService/GetLease"))

r.Post("/v1/orchestration/signals/send",                  h.ForwardTo("am-orchestration", "OrchestrationService/SendSignal"))
r.Get("/v1/orchestration/signals",                        h.ForwardTo("am-orchestration", "OrchestrationService/ListSignals"))
r.Post("/v1/orchestration/signals/{id}/read",             h.ForwardTo("am-orchestration", "OrchestrationService/MarkSignalRead"))
r.Delete("/v1/orchestration/signals/{id}",                h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSignal"))

r.Post("/v1/orchestration/routines",                      h.ForwardTo("am-orchestration", "OrchestrationService/CreateRoutine"))
r.Get("/v1/orchestration/routines",                       h.ForwardTo("am-orchestration", "OrchestrationService/ListRoutines"))
r.Post("/v1/orchestration/routines/{id}/execute",         h.ForwardTo("am-orchestration", "OrchestrationService/ExecuteRoutine"))

r.Post("/v1/orchestration/checkpoints",                   h.ForwardTo("am-orchestration", "OrchestrationService/CreateCheckpoint"))
r.Get("/v1/orchestration/checkpoints",                    h.ForwardTo("am-orchestration", "OrchestrationService/ListCheckpoints"))
r.Post("/v1/orchestration/checkpoints/{id}/approve",      h.ForwardTo("am-orchestration", "OrchestrationService/ApproveCheckpoint"))
r.Post("/v1/orchestration/checkpoints/{id}/reject",       h.ForwardTo("am-orchestration", "OrchestrationService/RejectCheckpoint"))

r.Post("/v1/orchestration/sentinels",                     h.ForwardTo("am-orchestration", "OrchestrationService/CreateSentinel"))
r.Get("/v1/orchestration/sentinels",                      h.ForwardTo("am-orchestration", "OrchestrationService/ListSentinels"))
r.Delete("/v1/orchestration/sentinels/{id}",              h.ForwardTo("am-orchestration", "OrchestrationService/DeleteSentinel"))

r.Post("/v1/orchestration/sketches",                      h.ForwardTo("am-orchestration", "OrchestrationService/CreateSketch"))
r.Get("/v1/orchestration/sketches",                       h.ForwardTo("am-orchestration", "OrchestrationService/ListSketches"))
r.Post("/v1/orchestration/sketches/{id}/add-action",      h.ForwardTo("am-orchestration", "OrchestrationService/AddActionToSketch"))
r.Post("/v1/orchestration/sketches/{id}/promote",         h.ForwardTo("am-orchestration", "OrchestrationService/PromoteSketch"))
r.Get("/v1/orchestration/crystals",                       h.ForwardTo("am-orchestration", "OrchestrationService/ListCrystals"))
r.Get("/v1/orchestration/crystals/{id}",                  h.ForwardTo("am-orchestration", "OrchestrationService/GetCrystal"))
```

---

## 3. Acceptance Criteria Mapping

| AC từ CR-AM-004 | Covered by |
|-----------------|------------|
| Acquire lease → active | leases.go Acquire() |
| Second acquire trong TTL → conflict:true | GetActiveLease() check |
| Sau TTL → expired → acquire succeeds | SweepExpired() background |
| SendSignal → recipient GET unread signals | signals.go |
| Invalid transition (done → active) → 400 | validTransitions map |
| PromoteSketch → Crystal created | sketches.go Promote() |
| Sentinel action_done → triggers | sentinels.go EvaluateAll() |
| Checkpoint approve → NATS event | checkpoints.go Approve() |

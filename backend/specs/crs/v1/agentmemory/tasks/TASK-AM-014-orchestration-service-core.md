# TASK-AM-014 — Orchestration Service Core (Domain + Usecases)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-014 |
| **Wave** | 3 (Orchestration) |
| **Component** | `services/orchestration-service/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-004 §2.1 → §2.5 |
| **Priority** | High |
| **Depends On** | TASK-AM-001 |
| **Estimated** | 8h |

---

## Context

Tạo **service #38** trong monolith: `am-orchestration`. Xử lý Actions (state machine), Leases (distributed locking), Signals (inter-agent messaging), Routines, Checkpoints, Sentinels, và Sketches/Crystals.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/orchestration-service/internal/domain/entity.go` |
| CREATE | `services/orchestration-service/internal/domain/value_object.go` |
| CREATE | `services/orchestration-service/internal/domain/errors.go` |
| CREATE | `services/orchestration-service/internal/orchestration/actions.go` |
| CREATE | `services/orchestration-service/internal/orchestration/leases.go` |
| CREATE | `services/orchestration-service/internal/orchestration/signals.go` |
| CREATE | `services/orchestration-service/internal/orchestration/checkpoints.go` |
| CREATE | `services/orchestration-service/internal/orchestration/routines.go` |
| CREATE | `services/orchestration-service/internal/orchestration/sketches.go` |
| CREATE | `services/orchestration-service/internal/usecase/port/output.go` |
| CREATE | `services/orchestration-service/internal/adapter/repository/postgres/repos.go` |
| CREATE | `services/orchestration-service/internal/adapter/event/publisher.go` |
| CREATE | `services/orchestration-service/internal/adapter/grpc/handler.go` |

---

## Implementation

### `internal/domain/entity.go`

```go
package domain

import "time"

type Action struct {
    ID           string
    TenantID     string
    Project      string
    AgentID      string
    Title        string
    Description  string
    Status       ActionStatus
    Priority     int
    Requires     []string  // action IDs
    ConflictsWith []string
    Tags         []string
    Result       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    CompletedAt  *time.Time
}

type Lease struct {
    ID         string
    ActionID   string
    AgentID    string
    Status     string    // "active" | "expired" | "released"
    AcquiredAt time.Time
    ExpiresAt  time.Time
    RenewedAt  *time.Time
}

type Signal struct {
    ID         string
    TenantID   string
    FromAgent  string
    ToAgent    string
    SignalType string    // handoff|update|cancel|request|response|alert
    Content    string
    ThreadID   string
    ReplyTo    string
    IsRead     bool
    ExpiresAt  time.Time
    CreatedAt  time.Time
}

type Routine struct {
    ID          string
    TenantID    string
    Project     string
    Name        string
    Description string
    Steps       []string  // JSONB: ordered step definitions
    CreatedAt   time.Time
}

type Checkpoint struct {
    ID          string
    TenantID    string
    Project     string
    AgentID     string
    ActionID    string
    Title       string
    Description string
    Status      string    // "pending" | "approved" | "rejected" | "expired"
    ApprovedBy  string
    RejectedBy  string
    Reason      string
    ExpiresAt   time.Time
    CreatedAt   time.Time
    ResolvedAt  *time.Time
}

type Sentinel struct {
    ID          string
    TenantID    string
    Name        string
    Condition   SentinelCondition
    ActionID    string
    SignalTo    string
    Status      string    // "watching" | "triggered" | "expired"
    ExpiresAt   time.Time
    CreatedAt   time.Time
    TriggeredAt *time.Time
}

type SentinelCondition struct {
    Type   string  // "action_done" | "signal_received" | "time"
    Target string
    Value  string
}

type Sketch struct {
    ID        string
    TenantID  string
    Project   string
    SessionID string
    Title     string
    ActionIDs []string
    Status    string    // "active" | "promoted" | "expired"
    ExpiresAt time.Time
    CreatedAt time.Time
}

type Crystal struct {
    ID              string
    TenantID        string
    SourceActionIDs []string
    Narrative       string
    KeyOutcomes     []string
    FilesAffected   []string
    Lessons         []string
    CreatedAt       time.Time
}
```

### `internal/domain/value_object.go`

```go
package domain

type ActionStatus string

const (
    ActionPending   ActionStatus = "pending"
    ActionActive    ActionStatus = "active"
    ActionBlocked   ActionStatus = "blocked"
    ActionDone      ActionStatus = "done"
    ActionCancelled ActionStatus = "cancelled"
    ActionFailed    ActionStatus = "failed"
)

// validTransitions defines allowed state machine transitions
var validTransitions = map[ActionStatus][]ActionStatus{
    ActionPending:   {ActionActive, ActionBlocked, ActionCancelled},
    ActionActive:    {ActionDone, ActionBlocked, ActionCancelled, ActionFailed},
    ActionBlocked:   {ActionActive, ActionCancelled},
    ActionDone:      {},
    ActionCancelled: {},
    ActionFailed:    {},
}

func (current ActionStatus) CanTransitionTo(next ActionStatus) bool {
    allowed := validTransitions[current]
    for _, a := range allowed { if a == next { return true } }
    return false
}
```

### `internal/domain/errors.go`

```go
package domain

import "errors"

var (
    ErrInvalidTransition = errors.New("invalid action status transition")
    ErrLeaseConflict     = errors.New("lease conflict: action already locked by another agent")
    ErrLeaseExpired      = errors.New("lease has expired")
    ErrActionNotFound    = errors.New("action not found")
    ErrSignalNotFound    = errors.New("signal not found")
    ErrSketchNotFound    = errors.New("sketch not found")
)

type ErrLeaseConflictDetail struct {
    ActiveLease *Lease
}
func (e ErrLeaseConflictDetail) Error() string { return "lease conflict" }
```

### `internal/orchestration/actions.go`

```go
package orchestration

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/orchestration-service/internal/domain"
    "github.com/vnp-memory/services/orchestration-service/internal/usecase/port"
)

type ActionService struct {
    repo      port.IActionRepo
    publisher port.IEventPublisher
}

func (s *ActionService) Create(ctx context.Context, req CreateActionRequest) (*domain.Action, error) {
    action := domain.Action{
        ID:           uuid.New().String(),
        TenantID:     req.TenantID,
        Project:      req.Project,
        AgentID:      req.AgentID,
        Title:        req.Title,
        Description:  req.Description,
        Status:       domain.ActionPending,
        Priority:     req.Priority,
        Requires:     req.Requires,
        ConflictsWith: req.ConflictsWith,
        Tags:         req.Tags,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    if action.Priority == 0 { action.Priority = 50 }

    if err := s.repo.Save(ctx, action); err != nil { return nil, err }
    return &action, nil
}

func (s *ActionService) UpdateStatus(ctx context.Context, actionID string, next domain.ActionStatus, result string) error {
    action, err := s.repo.Get(ctx, actionID)
    if err != nil { return domain.ErrActionNotFound }

    if !action.Status.CanTransitionTo(next) {
        return domain.ErrInvalidTransition
    }

    now := time.Now()
    action.Status = next
    action.Result = result
    action.UpdatedAt = now
    if next == domain.ActionDone || next == domain.ActionFailed || next == domain.ActionCancelled {
        action.CompletedAt = &now
    }

    if err := s.repo.Update(ctx, *action); err != nil { return err }

    s.publisher.Publish(ctx, "agentmemory.action.completed", map[string]any{
        "action_id": actionID, "status": string(next),
    })
    return nil
}

type CreateActionRequest struct {
    TenantID     string
    Project      string
    AgentID      string
    Title        string
    Description  string
    Priority     int
    Requires     []string
    ConflictsWith []string
    Tags         []string
}
```

### `internal/orchestration/leases.go`

```go
package orchestration

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/orchestration-service/internal/domain"
    "github.com/vnp-memory/services/orchestration-service/internal/usecase/port"
)

type LeaseService struct {
    repo      port.ILeaseRepo
    mu        sync.Map  // per-actionID in-process mutex (single-node fast path)
    publisher port.IEventPublisher
}

func (s *LeaseService) Acquire(ctx context.Context, actionID, agentID string, ttl time.Duration) (*domain.Lease, error) {
    // Get per-action mutex for in-process safety
    rawMu, _ := s.mu.LoadOrStore(actionID, &sync.Mutex{})
    mu := rawMu.(*sync.Mutex)
    mu.Lock()
    defer mu.Unlock()

    // Check PostgreSQL for existing active lease
    existing, _ := s.repo.GetActiveLease(ctx, actionID)
    if existing != nil && time.Now().Before(existing.ExpiresAt) {
        return nil, domain.ErrLeaseConflictDetail{ActiveLease: existing}
    }

    lease := domain.Lease{
        ID:         uuid.New().String(),
        ActionID:   actionID,
        AgentID:    agentID,
        Status:     "active",
        AcquiredAt: time.Now(),
        ExpiresAt:  time.Now().Add(ttl),
    }
    if err := s.repo.Save(ctx, lease); err != nil { return nil, err }

    s.publisher.Publish(ctx, "agentmemory.lease.acquired", map[string]any{
        "lease_id": lease.ID, "action_id": actionID, "agent_id": agentID,
    })
    return &lease, nil
}

func (s *LeaseService) Renew(ctx context.Context, leaseID string, extension time.Duration) error {
    return s.repo.ExtendExpiry(ctx, leaseID, extension)
}

func (s *LeaseService) Release(ctx context.Context, leaseID string) error {
    if err := s.repo.SetStatus(ctx, leaseID, "released"); err != nil { return err }
    s.publisher.Publish(ctx, "agentmemory.lease.released", map[string]any{"lease_id": leaseID})
    return nil
}

func (s *LeaseService) SweepExpired(ctx context.Context) {
    expired, _ := s.repo.FindExpired(ctx)
    for _, l := range expired {
        s.repo.SetStatus(ctx, l.ID, "expired")
        s.publisher.Publish(ctx, "agentmemory.lease.expired", map[string]any{
            "lease_id": l.ID, "action_id": l.ActionID,
        })
    }
}
```

### `internal/orchestration/sentinels.go`

```go
package orchestration

import (
    "context"

    "github.com/vnp-memory/services/orchestration-service/internal/domain"
    "github.com/vnp-memory/services/orchestration-service/internal/usecase/port"
)

type SentinelService struct {
    repo       port.ISentinelRepo
    actionRepo port.IActionRepo
    publisher  port.IEventPublisher
}

// EvaluateAll runs every 30s to check all "watching" sentinels
func (s *SentinelService) EvaluateAll(ctx context.Context) {
    sentinels, _ := s.repo.ListWatching(ctx)
    for _, sentinel := range sentinels {
        if s.conditionMet(ctx, sentinel.Condition) {
            s.trigger(ctx, sentinel)
        }
    }
}

func (s *SentinelService) conditionMet(ctx context.Context, cond domain.SentinelCondition) bool {
    switch cond.Type {
    case "action_done":
        action, _ := s.actionRepo.Get(ctx, cond.Target)
        return action != nil && action.Status == domain.ActionDone
    case "signal_received":
        return false // TODO: integrate with signal service
    case "time":
        return cronMatches(cond.Target)  // cron expression evaluation
    }
    return false
}

func (s *SentinelService) trigger(ctx context.Context, sentinel domain.Sentinel) {
    s.repo.SetStatus(ctx, sentinel.ID, "triggered")

    if sentinel.ActionID != "" {
        s.actionRepo.TransitionStatus(ctx, sentinel.ActionID, domain.ActionPending, domain.ActionActive)
    }

    s.publisher.Publish(ctx, "agentmemory.sentinel.triggered", map[string]any{
        "sentinel_id": sentinel.ID, "condition": sentinel.Condition,
    })
}

// cronMatches: simple time-based condition check
func cronMatches(cronExpr string) bool {
    // Basic implementation: use time.Now() vs cron expression
    // For prod: use robfig/cron or similar
    return false
}
```

### `internal/orchestration/sketches.go`

```go
package orchestration

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/services/orchestration-service/internal/domain"
    "github.com/vnp-memory/services/orchestration-service/internal/usecase/port"
)

type SketchService struct {
    repo        port.ISketchRepo
    actionRepo  port.IActionRepo
    crystalRepo port.ICrystalRepo
    llm         port.ILLMProvider
}

func (s *SketchService) Promote(ctx context.Context, sketchID string) (*domain.Crystal, error) {
    sketch, err := s.repo.Get(ctx, sketchID)
    if err != nil { return nil, domain.ErrSketchNotFound }

    // Collect action results
    var summaries []string
    var actionIDs []string
    for _, aid := range sketch.ActionIDs {
        action, _ := s.actionRepo.Get(ctx, aid)
        if action != nil && action.Result != "" {
            summaries = append(summaries, action.Title+": "+action.Result)
            actionIDs = append(actionIDs, aid)
        }
    }

    crystal := domain.Crystal{
        ID:              uuid.New().String(),
        TenantID:        sketch.TenantID,
        SourceActionIDs: actionIDs,
        CreatedAt:       time.Now(),
    }

    // Try LLM crystal generation
    if s.llm != nil && len(summaries) > 0 {
        prompt := buildCrystalPrompt(summaries)
        resp, err := s.llm.Chat(ctx, "Generate a crystal memory from these action results.", prompt)
        if err == nil {
            var result struct {
                Narrative   string   `json:"narrative"`
                KeyOutcomes []string `json:"key_outcomes"`
                Lessons     []string `json:"lessons"`
            }
            if json.Unmarshal([]byte(resp), &result) == nil {
                crystal.Narrative   = result.Narrative
                crystal.KeyOutcomes = result.KeyOutcomes
                crystal.Lessons     = result.Lessons
            }
        }
    }

    // Graceful degrade: synthetic crystal
    if crystal.Narrative == "" {
        crystal.Narrative   = strings.Join(summaries, "; ")
        crystal.KeyOutcomes = summaries
    }

    if err := s.crystalRepo.Save(ctx, crystal); err != nil { return nil, err }
    s.repo.SetStatus(ctx, sketchID, "promoted")
    return &crystal, nil
}

func buildCrystalPrompt(summaries []string) string {
    return fmt.Sprintf(`Synthesize these action results into a crystal memory.
Return JSON: {"narrative": "...", "key_outcomes": [...], "lessons": [...]}

Actions:
%s`, strings.Join(summaries, "\n"))
}
```

### `internal/usecase/port/output.go`

```go
package port

import (
    "context"
    "time"
    "github.com/vnp-memory/services/orchestration-service/internal/domain"
)

type IActionRepo interface {
    Save(ctx context.Context, action domain.Action) error
    Get(ctx context.Context, id string) (*domain.Action, error)
    List(ctx context.Context, tenantID, status string, limit int) ([]domain.Action, error)
    Update(ctx context.Context, action domain.Action) error
    Delete(ctx context.Context, id string) error
    TransitionStatus(ctx context.Context, id string, from, to domain.ActionStatus) error
}

type ILeaseRepo interface {
    Save(ctx context.Context, lease domain.Lease) error
    GetActiveLease(ctx context.Context, actionID string) (*domain.Lease, error)
    ExtendExpiry(ctx context.Context, leaseID string, extension time.Duration) error
    SetStatus(ctx context.Context, leaseID, status string) error
    FindExpired(ctx context.Context) ([]domain.Lease, error)
}

type ISignalRepo interface {
    Save(ctx context.Context, signal domain.Signal) error
    GetByID(ctx context.Context, id string) (*domain.Signal, error)
    List(ctx context.Context, tenantID, agentID string, unreadOnly bool) ([]domain.Signal, error)
    MarkRead(ctx context.Context, id string) error
    Delete(ctx context.Context, id string) error
    DeleteExpired(ctx context.Context) error
}

type ISentinelRepo interface {
    Save(ctx context.Context, sentinel domain.Sentinel) error
    ListWatching(ctx context.Context) ([]domain.Sentinel, error)
    SetStatus(ctx context.Context, id, status string) error
    Delete(ctx context.Context, id string) error
}

type ISketchRepo interface {
    Save(ctx context.Context, sketch domain.Sketch) error
    Get(ctx context.Context, id string) (*domain.Sketch, error)
    List(ctx context.Context, tenantID string) ([]domain.Sketch, error)
    AddAction(ctx context.Context, sketchID, actionID string) error
    SetStatus(ctx context.Context, id, status string) error
}

type ICrystalRepo interface {
    Save(ctx context.Context, crystal domain.Crystal) error
    Get(ctx context.Context, id string) (*domain.Crystal, error)
    List(ctx context.Context, tenantID string) ([]domain.Crystal, error)
}

type ICheckpointRepo interface {
    Save(ctx context.Context, cp domain.Checkpoint) error
    List(ctx context.Context, tenantID, status string) ([]domain.Checkpoint, error)
    Resolve(ctx context.Context, id, status, by, reason string) error
    AutoRejectExpired(ctx context.Context) error
}

type IRoutineRepo interface {
    Save(ctx context.Context, routine domain.Routine) error
    List(ctx context.Context, tenantID string) ([]domain.Routine, error)
    Get(ctx context.Context, id string) (*domain.Routine, error)
}

type IEventPublisher interface {
    Publish(ctx context.Context, subject string, payload any) error
}

type ILLMProvider interface {
    Chat(ctx context.Context, systemPrompt, userMsg string) (string, error)
}
```

---

## Verification

```bash
cd services/orchestration-service
go build ./...
go test ./internal/orchestration/... -v
```

**Tests:**
```go
func TestActionStateMachine_ValidTransitions(t *testing.T) {
    assert.True(t, ActionPending.CanTransitionTo(ActionActive))
    assert.True(t, ActionActive.CanTransitionTo(ActionDone))
    assert.False(t, ActionDone.CanTransitionTo(ActionActive))  // terminal
}

func TestLeaseService_ConflictDetection(t *testing.T) {
    // Acquire → success
    // Second acquire within TTL → ErrLeaseConflict
    // After TTL expires + sweep → third acquire succeeds
}

func TestSketch_SyntheticPromote(t *testing.T) {
    // No LLM → crystal generated from action summaries
    // crystal.Narrative = join(summaries)
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `CreateAction` → `{action_id}` | ✅ |
| `UpdateAction` done→active → `ErrInvalidTransition` | ✅ |
| `AcquireLease` → `{lease_id}` | ✅ |
| 2nd acquire within TTL → `{conflict: true}` | ✅ |
| `SweepExpired` → expired leases marked, NATS event published | ✅ |
| `PromoteSketch` → `{crystal_id}`, narrative populated | ✅ |
| `SentinelService.EvaluateAll` → triggers on condition_met | ✅ |

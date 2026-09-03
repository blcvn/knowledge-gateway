# Change Request: CR-AM-004 — Multi-Agent Orchestration Layer

**CR ID:** CR-AM-004  
**Component:** `services/orchestration-service` [NEW SERVICE]  
**Priority:** High  
**Status:** ✅ Implemented  
**Reference:** agentmemory PRD §6.4, SRS FR-MULTI-001..005, FR-ORCH-001..005  
**Spec:** `references/agentmemory/specs/services/orchestration-service/spec.md`

---

## 1. Mô tả

Xây dựng **Orchestration Service** — coordination layer cho multi-agent scenarios, cung cấp:
1. **Actions** — Task graph với priority, dependencies, state machine.
2. **Leases** — Distributed locking (prevent concurrent write conflicts).
3. **Signals** — Inter-agent messaging (handoff, update, alert).
4. **Routines** — Reusable workflow templates với step ordering.
5. **Checkpoints** — Human-in-the-loop approval gates.
6. **Sentinels** — Event watchers triggering on conditions.
7. **Sketches & Crystals** — Ephemeral action groups → permanent records.

---

## 2. Vấn đề hiện tại

VNP Memory hiện không có cơ chế nào để:
- Ngăn chặn race condition khi nhiều agents (Claude Code + Cursor) cùng ghi memory.
- Cho phép agents thông báo cho nhau khi một task hoàn thành (handoff).
- Quản lý task dependency graph (requires, conflicts_with).
- Tạo approval workflow (human-in-the-loop checkpoint).

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/orchestration-service/`

**Port:** `8085`  
**Binary:** `cmd/orchestration/main.go`

**Cấu trúc:**
```
services/orchestration-service/
├── cmd/orchestration/main.go
├── internal/
│   ├── domain/
│   │   ├── entity.go       # Action, Lease, Signal, Routine, Checkpoint, Sentinel, Sketch, Crystal
│   │   ├── value_object.go # ActionStatus, SignalType, CheckpointStatus, SentinelCondition
│   │   └── errors.go       # ErrLeaseConflict, ErrInvalidTransition
│   ├── orchestration/
│   │   ├── actions.go      # Action CRUD + state machine transitions
│   │   ├── leases.go       # Distributed lock (per-actionId keyed mutex)
│   │   ├── signals.go      # Inter-agent messaging
│   │   ├── routines.go     # Workflow templates + execution
│   │   ├── checkpoints.go  # Approval gates
│   │   ├── sentinels.go    # Event watchers
│   │   └── sketches.go     # Ephemeral groups + crystal promotion
│   ├── usecase/
│   │   └── port/output.go  # KVStore, LLMProvider, EventPublisher
│   └── adapter/
│       ├── http/handler.go
│       └── background/
│           └── sweeper.go  # Background jobs
```

### 3.2. Domain Models

```go
// internal/domain/entity.go

type ActionStatus string
const (
    ActionPending   ActionStatus = "pending"
    ActionActive    ActionStatus = "active"
    ActionBlocked   ActionStatus = "blocked"
    ActionDone      ActionStatus = "done"
    ActionCancelled ActionStatus = "cancelled"
    ActionFailed    ActionStatus = "failed"
)

type Action struct {
    ID            string
    Title         string
    Description   string
    Status        ActionStatus
    Priority      int         // 0-100
    AgentID       string
    Project       string
    TenantID      string
    Requires      []string    // actionIDs that must complete first
    ConflictsWith []string    // cannot run simultaneously
    Tags          []string
    Result        string      // completion result/summary
    CreatedAt     time.Time
    UpdatedAt     time.Time
    CompletedAt   *time.Time
}

// State Machine:
// pending → active | blocked | cancelled
// active  → done | blocked | cancelled | failed
// blocked → active | cancelled
// done, cancelled, failed → terminal

type Lease struct {
    ID         string
    ActionID   string
    AgentID    string
    AcquiredAt time.Time
    ExpiresAt  time.Time
    RenewedAt  *time.Time
    Status     string    // "active" | "expired" | "released"
}

type Signal struct {
    ID        string
    From      string    // sender agentId
    To        string    // recipient agentId
    Type      string    // "handoff" | "update" | "cancel" | "request" | "response" | "alert"
    Content   string
    ThreadID  string    // for conversation threading
    ReplyTo   string    // signal being replied to
    Read      bool
    ExpiresAt time.Time
    CreatedAt time.Time
}

type Routine struct {
    ID          string
    Name        string
    Description string
    Steps       []RoutineStep
    Project     string
    TenantID    string
    CreatedAt   time.Time
}

type RoutineStep struct {
    ID         string
    Title      string
    DependsOn  []string  // step IDs
    ActionType string    // "observe" | "remember" | "search" | "signal" | "custom"
    Config     map[string]any
}

type Checkpoint struct {
    ID          string
    Title       string
    Description string
    Status      CheckpointStatus  // pending | approved | rejected
    AgentID     string
    Project     string
    TenantID    string
    ActionID    string
    ApprovedBy  string
    RejectedBy  string
    Reason      string
    ExpiresAt   time.Time
    CreatedAt   time.Time
    ResolvedAt  *time.Time
}

type Sentinel struct {
    ID          string
    Name        string
    Condition   SentinelCondition
    ActionID    string    // trigger this action when condition fires
    SignalTo    string    // or send signal to this agent
    Status      string    // "watching" | "triggered" | "expired"
    CreatedAt   time.Time
    ExpiresAt   time.Time
    TriggeredAt *time.Time
}

type SentinelCondition struct {
    Type   string  // "action_done" | "signal_received" | "time" | "kv_change"
    Target string  // actionID, agentID, cron expression, or KV namespace
    Value  string  // expected value (for kv_change)
}

type Sketch struct {
    ID        string
    Title     string
    ActionIDs []string
    Status    string    // "active" | "promoted" | "expired"
    SessionID string
    Project   string
    TenantID  string
    ExpiresAt time.Time
    CreatedAt time.Time
}

type Crystal struct {
    ID              string
    SourceActionIDs []string
    Narrative       string
    KeyOutcomes     []string
    FilesAffected   []string
    Lessons         []string
    CreatedAt       time.Time
}
```

### 3.3. Lease Service (Distributed Locking)

```go
// internal/orchestration/leases.go
// Per-actionId keyed mutex for atomic check-and-set
// Sweep expired leases every 60s (background goroutine)
// No external distributed lock (Redis) required — in-process mutex sufficient
// for single-node deployment; for multi-node: upgrade to Redis SETNX
```

### 3.4. Action State Machine Validation

```go
var validTransitions = map[ActionStatus][]ActionStatus{
    ActionPending:   {ActionActive, ActionBlocked, ActionCancelled},
    ActionActive:    {ActionDone, ActionBlocked, ActionCancelled, ActionFailed},
    ActionBlocked:   {ActionActive, ActionCancelled},
    ActionDone:      {},  // terminal
    ActionCancelled: {},  // terminal
    ActionFailed:    {},  // terminal
}
```

### 3.5. Crystal Promotion (LLM-generated record)

```go
// When sketch is promoted:
// 1. Collect all action results
// 2. LLM: generate {narrative, key_outcomes, files_affected, lessons}
// 3. Graceful degrade: if no LLM → synthetic crystal from action titles
// 4. Save Crystal, mark Sketch as "promoted"
```

### 3.6. Background Jobs

| Job | Interval | Nhiệm vụ |
|---|---|---|
| Lease sweeper | 60s | Mark expired leases as "expired" |
| Signal sweeper | 300s | Delete expired signals |
| Sentinel checker | 30s | Evaluate sentinel conditions |
| Sketch reaper | 3600s | Mark expired sketches as "expired" |
| Checkpoint reaper | 3600s | Auto-reject expired pending checkpoints |

### 3.7. API Endpoints

```
# Actions
POST   /v1/orchestration/actions
GET    /v1/orchestration/actions?status=pending&project=...
GET    /v1/orchestration/actions/{id}
PATCH  /v1/orchestration/actions/{id}
DELETE /v1/orchestration/actions/{id}

# Leases
POST   /v1/orchestration/leases/acquire
POST   /v1/orchestration/leases/renew
POST   /v1/orchestration/leases/release
GET    /v1/orchestration/leases/{actionId}

# Signals
POST   /v1/orchestration/signals/send
GET    /v1/orchestration/signals?to={agentId}&unread=true
POST   /v1/orchestration/signals/{id}/read
DELETE /v1/orchestration/signals/{id}

# Routines
POST   /v1/orchestration/routines
GET    /v1/orchestration/routines
POST   /v1/orchestration/routines/{id}/execute

# Checkpoints
POST   /v1/orchestration/checkpoints
GET    /v1/orchestration/checkpoints?status=pending
POST   /v1/orchestration/checkpoints/{id}/approve
POST   /v1/orchestration/checkpoints/{id}/reject

# Sentinels
POST   /v1/orchestration/sentinels
GET    /v1/orchestration/sentinels?status=watching
DELETE /v1/orchestration/sentinels/{id}

# Sketches & Crystals
POST   /v1/orchestration/sketches
GET    /v1/orchestration/sketches
POST   /v1/orchestration/sketches/{id}/add-action
POST   /v1/orchestration/sketches/{id}/promote
GET    /v1/orchestration/crystals
GET    /v1/orchestration/crystals/{id}
```

### 3.8. NATS Events

| Subject | Direction | Payload |
|---|---|---|
| `agentmemory.action.completed` | Publish | `{action_id, agent_id, project}` |
| `agentmemory.lease.acquired` | Publish | `{lease_id, action_id, agent_id}` |
| `agentmemory.lease.expired` | Publish | `{lease_id, action_id}` |
| `agentmemory.signal.sent` | Publish | `{signal_id, from, to, type}` |
| `agentmemory.checkpoint.resolved` | Publish | `{checkpoint_id, status, resolved_by}` |

### 3.9. Database Changes

**[NEW]** PostgreSQL tables: `agent_actions`, `agent_leases`, `agent_signals`, `agent_routines`, `agent_checkpoints`, `agent_sentinels`, `agent_sketches`, `agent_crystals`

---

## 4. Acceptance Criteria

- [x] `POST /v1/orchestration/leases/acquire` với `action_id: "act_1"` → returns lease with `status: "active"`.
- [x] Second acquire call với cùng `action_id` trong TTL → returns `{conflict: true}` (no error).
- [x] Sau khi lease TTL hết hạn (mock time) → status = `"expired"`, acquire succeeds cho agent khác.
- [x] `POST /v1/orchestration/signals/send` với `type: "handoff"` → recipient agent có thể `GET /signals?to={agentId}&unread=true`.
- [x] `PATCH /v1/orchestration/actions/{id}` với invalid transition (done → active) → returns 400.
- [x] `POST /v1/orchestration/sketches/{id}/promote` → Crystal được tạo với `narrative` + `key_outcomes`.
- [x] Sentinel với `condition: {type: "action_done", target: "act_1"}`: khi act_1 hoàn thành → sentinel triggers + signal sent.
- [x] Checkpoint approval workflow: create → pending → approve → `status: "approved"`, NATS event published.

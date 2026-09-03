# Solution: SOL-ORCH-001 — Checkpoints (Human Approval Gates) & Sentinels (Event Watchers)

**CR:** CR-ORCH-001
**TDD refs:** `architecture/12-agentmemory-services.md`, `models/obs-service.md`
**Version:** v3/orchestration

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** Checkpoint + Sentinel domain + SentinelService implemented; checkpoint HTTP handler pending
---

## 1. Architecture Analysis

From TDD `agentmemory-services.md`, `orchestration-service` already implements:
- Distributed leases (Redis SET NX) → solved by CR-ENT-001/SOL-ENT-001
- Signal routing (NATS) → CR-ENT-001 covers this

This solution adds: **Checkpoints** (human approval blocking), **Sentinels** (NATS event watchers), **Sketches/Crystals** (ephemeral memory).

---

## 2. Checkpoint Domain

```go
// services/orchestration-service/internal/domain/checkpoint.go [NEW]
type CheckpointStatus string
const (
    CheckpointPending  CheckpointStatus = "pending"
    CheckpointApproved CheckpointStatus = "approved"
    CheckpointRejected CheckpointStatus = "rejected"
    CheckpointExpired  CheckpointStatus = "expired"
)

type Checkpoint struct {
    ID             string           `json:"id"`
    ActionID       string           `json:"action_id"`
    TenantID       string           `json:"tenant_id"`
    AgentID        string           `json:"agent_id"`
    Type           string           `json:"type"`           // approval|review|confirmation
    Message        string           `json:"message"`
    Status         CheckpointStatus `json:"status"`
    TimeoutMinutes int              `json:"timeout_minutes"` // default: 60
    CreatedAt      time.Time        `json:"created_at"`
    ResolvedAt     *time.Time       `json:"resolved_at,omitempty"`
    ResolvedBy     string           `json:"resolved_by,omitempty"`
}
```

---

## 3. Checkpoint Usecase

```go
// services/orchestration-service/internal/usecase/checkpoint.go [NEW]

type CheckpointUseCase struct {
    repo    port.CheckpointRepository
    pub     port.EventPublisher
    nats    *nats.Conn
}

func (u *CheckpointUseCase) Create(ctx context.Context, req *CreateCheckpointRequest) (*Checkpoint, error) {
    cp := &Checkpoint{
        ID: uuid.NewString(), ActionID: req.ActionID,
        TenantID: req.TenantID, AgentID: req.AgentID,
        Type: req.Type, Message: req.Message,
        Status: CheckpointPending,
        TimeoutMinutes: 60,
    }
    if req.TimeoutMinutes > 0 { cp.TimeoutMinutes = req.TimeoutMinutes }

    if err := u.repo.Create(ctx, cp); err != nil { return nil, err }

    // Publish notification for Console / webhook
    u.pub.Publish(ctx, "agent.checkpoint.created", map[string]any{
        "checkpoint_id": cp.ID, "tenant_id": cp.TenantID,
        "agent_id": cp.AgentID, "message": cp.Message,
    })

    // Schedule auto-expiry
    go func() {
        time.Sleep(time.Duration(cp.TimeoutMinutes) * time.Minute)
        u.expireCheckpoint(cp.ID, cp.TenantID)
    }()

    return cp, nil
}

func (u *CheckpointUseCase) Resolve(ctx context.Context, id, tenantID, resolvedBy string, approved bool) error {
    cp, err := u.repo.Get(ctx, id, tenantID)
    if err != nil { return err }
    if cp.Status != CheckpointPending { return ErrCheckpointAlreadyResolved }

    now := time.Now()
    cp.Status = CheckpointApproved
    if !approved { cp.Status = CheckpointRejected }
    cp.ResolvedAt = &now; cp.ResolvedBy = resolvedBy

    if err := u.repo.Update(ctx, cp); err != nil { return err }

    // Notify agent: publish to NATS subject the agent is polling
    subject := fmt.Sprintf("agent.checkpoint.resolved.%s", id)
    data, _ := json.Marshal(map[string]any{
        "checkpoint_id": id, "status": string(cp.Status), "resolved_by": resolvedBy,
    })
    return u.nats.Publish(subject, data)
}

// Agent polls this until resolved
func (u *CheckpointUseCase) GetStatus(ctx context.Context, id, tenantID string) (*Checkpoint, error) {
    return u.repo.Get(ctx, id, tenantID)
}

func (u *CheckpointUseCase) expireCheckpoint(id, tenantID string) {
    ctx := context.Background()
    cp, err := u.repo.Get(ctx, id, tenantID)
    if err != nil || cp.Status != CheckpointPending { return }
    now := time.Now()
    cp.Status = CheckpointExpired; cp.ResolvedAt = &now
    u.repo.Update(ctx, cp)
    u.nats.Publish(fmt.Sprintf("agent.checkpoint.resolved.%s", id),
        []byte(`{"status":"expired"}`))
}
```

---

## 4. Sentinel Domain & Usecase

```go
// services/orchestration-service/internal/domain/sentinel.go [NEW]
type SentinelTrigger struct {
    Type      string `json:"type"`      // event|memory_count|time|error_rate
    Condition string `json:"condition"` // human-readable expression
    Subject   string `json:"subject"`   // NATS subject pattern (for event type)
}

type SentinelAction struct {
    Type    string         `json:"type"`    // notify|signal|webhook|pause_agent
    Payload map[string]any `json:"payload"`
}

type Sentinel struct {
    ID              string          `json:"id"`
    TenantID        string          `json:"tenant_id"`
    AgentID         string          `json:"agent_id"`
    Trigger         SentinelTrigger `json:"trigger"`
    Action          SentinelAction  `json:"action"`
    Status          string          `json:"status"` // active|triggered|disabled
    LastTriggeredAt *time.Time      `json:"last_triggered_at,omitempty"`
    CreatedAt       time.Time       `json:"created_at"`
}

// services/orchestration-service/internal/usecase/sentinel.go [NEW]
type SentinelUseCase struct {
    repo    port.SentinelRepository
    nats    *nats.Conn
    pub     port.EventPublisher
    subs    map[string]*nats.Subscription
    mu      sync.Mutex
}

func (u *SentinelUseCase) Create(ctx context.Context, s *Sentinel) (*Sentinel, error) {
    s.ID = uuid.NewString(); s.Status = "active"
    if err := u.repo.Create(ctx, s); err != nil { return nil, err }

    // For event-type sentinels: subscribe to NATS subject immediately
    if s.Trigger.Type == "event" && s.Trigger.Subject != "" {
        u.mu.Lock()
        sub, err := u.nats.Subscribe(s.Trigger.Subject, func(msg *nats.Msg) {
            go u.trigger(ctx, s, msg.Data)
        })
        if err == nil { u.subs[s.ID] = sub }
        u.mu.Unlock()
    }
    return s, nil
}

func (u *SentinelUseCase) trigger(ctx context.Context, s *Sentinel, payload []byte) {
    now := time.Now()
    s.LastTriggeredAt = &now
    u.repo.Update(ctx, s)

    switch s.Action.Type {
    case "notify":
        u.pub.Publish(ctx, "agent.sentinel.triggered", map[string]any{
            "sentinel_id": s.ID, "agent_id": s.AgentID, "payload": string(payload),
        })
    case "signal":
        subject := fmt.Sprintf("agent.signal.%s.%s", s.TenantID, s.AgentID)
        u.nats.Publish(subject, payload)
    case "pause_agent":
        u.pub.Publish(ctx, "agent.pause_requested", map[string]string{"agent_id": s.AgentID})
    }
}
```

---

## 5. Sketch & Crystal (Ephemeral Memory)

```go
// services/orchestration-service/internal/usecase/sketch.go [NEW]
type Sketch struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    AgentID   string    `json:"agent_id"`
    Content   string    `json:"content"`
    ExpiresAt time.Time `json:"expires_at"` // 24h from creation
    CreatedAt time.Time `json:"created_at"`
}

func (u *SketchUseCase) Create(ctx context.Context, req *CreateSketchRequest) (*Sketch, error) {
    sk := &Sketch{
        ID: uuid.NewString(), TenantID: req.TenantID,
        AgentID: req.AgentID, Content: req.Content,
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }
    return sk, u.repo.Create(ctx, sk)
}

// Crystallize — promote sketch to permanent memory
func (u *SketchUseCase) Crystallize(ctx context.Context, sketchID, tenantID string, memType string) error {
    sk, err := u.repo.Get(ctx, sketchID, tenantID)
    if err != nil { return err }

    // Call memory store to make permanent
    conn := u.registry.Get("gateway") // or directly call engine
    // ... dispatch to appropriate engine based on memType
    svcName := domain.EngineService(memType)
    conn = u.registry.Get(svcName)
    client := ingestpb.NewIngestServiceClient(conn)
    _, err = client.Ingest(ctx, &ingestpb.IngestRequest{
        Content: sk.Content, TenantId: tenantID,
        Type: memType,
    })
    if err != nil { return err }

    return u.repo.Delete(ctx, sketchID)
}
```

---

## 6. HTTP Handler

```go
// gateway/adapter/handler/orchestration_handler.go [MODIFY — add checkpoint/sentinel/sketch]

// POST /v1/orchestration/checkpoints
func (h *OrchHandler) CreateCheckpoint(w http.ResponseWriter, r *http.Request) {
    var req domain.CreateCheckpointRequest
    json.NewDecoder(r.Body).Decode(&req)
    req.TenantID = tenant.FromContext(r.Context())
    cp, err := h.checkpointUC.Create(r.Context(), &req)
    if err != nil { writeError(w, 500, "create_failed", err.Error()); return }
    writeJSON(w, 202, cp)
}

// GET /v1/orchestration/checkpoints/{id} — agent polls
func (h *OrchHandler) GetCheckpoint(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    cp, err := h.checkpointUC.GetStatus(r.Context(), id, tenantID)
    if err != nil { writeError(w, 404, "not_found", ""); return }
    writeJSON(w, 200, cp)
}

// POST /v1/orchestration/checkpoints/{id}/approve
func (h *OrchHandler) ApproveCheckpoint(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    actorID  := auth.UserIDFromContext(r.Context())
    h.checkpointUC.Resolve(r.Context(), id, tenantID, actorID, true)
    writeJSON(w, 200, map[string]string{"status": "approved"})
}

// POST /v1/orchestration/checkpoints/{id}/reject
func (h *OrchHandler) RejectCheckpoint(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    tenantID := tenant.FromContext(r.Context())
    actorID  := auth.UserIDFromContext(r.Context())
    h.checkpointUC.Resolve(r.Context(), id, tenantID, actorID, false)
    writeJSON(w, 200, map[string]string{"status": "rejected"})
}
```

---

## 7. DB Migration

```sql
-- deployment/dev/migrations/0047_checkpoints_sentinels.sql
CREATE TABLE agent_checkpoints (
    id UUID PRIMARY KEY,
    action_id TEXT, tenant_id TEXT NOT NULL, agent_id TEXT,
    type TEXT DEFAULT 'approval', message TEXT,
    status TEXT DEFAULT 'pending',
    timeout_minutes INT DEFAULT 60,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ, resolved_by TEXT
);

CREATE TABLE agent_sentinels (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL, agent_id TEXT,
    trigger_type TEXT, trigger_condition TEXT, trigger_subject TEXT,
    action_type TEXT, action_payload JSONB,
    status TEXT DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE agent_sketches (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL, agent_id TEXT,
    content TEXT, expires_at TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Auto cleanup expired sketches
CREATE INDEX idx_sketches_expires ON agent_sketches(expires_at);
```

---

## 8. File Changes

| File | Action |
|---|---|
| `services/orchestration-service/internal/domain/checkpoint.go` | **[NEW]** |
| `services/orchestration-service/internal/domain/sentinel.go` | **[NEW]** |
| `services/orchestration-service/internal/usecase/checkpoint.go` | **[NEW]** |
| `services/orchestration-service/internal/usecase/sentinel.go` | **[NEW]** |
| `services/orchestration-service/internal/usecase/sketch.go` | **[NEW]** |
| `gateway/adapter/handler/orchestration_handler.go` | **[MODIFY]** checkpoint/sentinel/sketch endpoints |
| `gateway/adapter/handler/router.go` | **[MODIFY]** new routes |
| `deployment/dev/migrations/0047_checkpoints_sentinels.sql` | **[NEW]** |

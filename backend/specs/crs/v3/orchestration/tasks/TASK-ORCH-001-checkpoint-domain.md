# TASK-ORCH-001 — Checkpoint Domain Model & DB Migration

| Field | Value |
|---|---|
| **Task ID** | TASK-ORCH-001 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-ORCH-001](../solutions/SOL-ORCH-001-Checkpoints-Sentinels.md) §2 |
| **Component** | `services/orchestration-service/internal/domain/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo domain models `Checkpoint`, `Sentinel`, `Sketch` và DB migration.

---

## Công việc cụ thể

### 1. Tạo `services/orchestration-service/internal/domain/checkpoint.go` [NEW]

```go
package domain

import "time"

type CheckpointStatus string

const (
    CheckpointPending  CheckpointStatus = "pending"
    CheckpointApproved CheckpointStatus = "approved"
    CheckpointRejected CheckpointStatus = "rejected"
    CheckpointExpired  CheckpointStatus = "expired"
)

type Checkpoint struct {
    ID             string           `json:"id"`
    ActionID       string           `json:"action_id,omitempty"`
    TenantID       string           `json:"tenant_id"`
    AgentID        string           `json:"agent_id"`
    Type           string           `json:"type"`            // approval|review|confirmation
    Message        string           `json:"message"`
    Status         CheckpointStatus `json:"status"`
    TimeoutMinutes int              `json:"timeout_minutes"`
    CreatedAt      time.Time        `json:"created_at"`
    ResolvedAt     *time.Time       `json:"resolved_at,omitempty"`
    ResolvedBy     string           `json:"resolved_by,omitempty"`
}

type Sentinel struct {
    ID              string     `json:"id"`
    TenantID        string     `json:"tenant_id"`
    AgentID         string     `json:"agent_id"`
    TriggerType     string     `json:"trigger_type"`      // event|memory_count|time|error_rate
    TriggerCondition string    `json:"trigger_condition"`
    TriggerSubject  string     `json:"trigger_subject,omitempty"` // NATS subject for event type
    ActionType      string     `json:"action_type"`       // notify|signal|webhook|pause_agent
    ActionPayload   map[string]any `json:"action_payload,omitempty"`
    Status          string     `json:"status"`            // active|triggered|disabled
    LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}

type Sketch struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    AgentID   string    `json:"agent_id"`
    Content   string    `json:"content"`
    ExpiresAt time.Time `json:"expires_at"` // 24h from creation
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. DB Migration `deployment/dev/migrations/0047_checkpoints_sentinels.sql` [NEW]

```sql
CREATE TABLE agent_checkpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id       TEXT,
    tenant_id       TEXT NOT NULL,
    agent_id        TEXT,
    type            TEXT DEFAULT 'approval',
    message         TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    timeout_minutes INT DEFAULT 60,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    resolved_at     TIMESTAMPTZ,
    resolved_by     TEXT
);
CREATE INDEX idx_checkpoints_tenant ON agent_checkpoints(tenant_id, status);

CREATE TABLE agent_sentinels (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         TEXT NOT NULL,
    agent_id          TEXT,
    trigger_type      TEXT NOT NULL,
    trigger_condition TEXT,
    trigger_subject   TEXT,
    action_type       TEXT NOT NULL,
    action_payload    JSONB,
    status            TEXT DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE agent_sketches (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  TEXT NOT NULL,
    agent_id   TEXT,
    content    TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_sketches_expires ON agent_sketches(expires_at);
```

---

## Acceptance Criteria

- [ ] Domain structs compile
- [ ] Migration runs without error: `goose up`
- [ ] `agent_checkpoints`, `agent_sentinels`, `agent_sketches` tables exist

## Files

```
services/orchestration-service/internal/domain/checkpoint.go   [NEW]
deployment/dev/migrations/0047_checkpoints_sentinels.sql        [NEW]
```

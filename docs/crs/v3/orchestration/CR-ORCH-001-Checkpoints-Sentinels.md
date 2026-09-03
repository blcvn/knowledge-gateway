# Change Request: CR-ORCH-001 — Checkpoints (Human Approval) & Sentinels (Event Watchers)

**CR ID:** CR-ORCH-001
**Component:** `backend/services/orchestration-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v3 / Orchestration
**Feature:** [F11](../../../features/11-multi-agent-orchestration/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-05 | Agent Developer | Không thể pause agent và chờ human approval |
| PP-P1-06 | Agent Developer | Không có automated event watchers |

---

## 2. Checkpoints (Human Approval Gates)

```
Agent reaches checkpoint:
  1. POST /v1/orchestration/checkpoints (agent creates gate)
  2. Agent pauses → polls checkpoint status
  3. Human notified (webhook / console notification)
  4. POST /v1/orchestration/checkpoints/{id}/approve OR /reject
  5. Agent resumes (approve) OR rolls back (reject)

Checkpoint model:
  {
    id, action_id, tenant_id, agent_id,
    type: "approval|review|confirmation",
    message: "About to delete 1000 memories. Approve?",
    status: "pending|approved|rejected|expired",
    timeout_minutes: 60,  // auto-reject after 60min
    created_at, resolved_at, resolved_by
  }
```

---

## 3. Sentinels (Event Watchers)

```
Sentinel: background watcher that triggers action when condition met.

Sentinel model:
  {
    id, tenant_id, agent_id,
    trigger: {
      type: "event|memory_count|time|error_rate",
      condition: "error_count > 10 in 5min",
      nats_subject: "agent.*.error"  // for event type
    },
    action: {
      type: "notify|signal|webhook|pause_agent",
      payload: {...}
    },
    status: "active|triggered|disabled"
  }

Examples:
  - "If cognee-ingestion error rate > 20% → send alert signal"
  - "If user hasn't stored memory in 7 days → send nudge"
  - "If session has > 100 observations → trigger consolidation"
```

---

## 4. Sketches & Crystals

```
Sketch: ephemeral draft memory (L0)
  - Agent writes temporary notes during reasoning
  - Auto-expires: TTL 24h
  - Never consolidated to long-term

Crystal: crystallized sketch → permanent memory
  - Agent calls POST /v1/orchestration/crystals
  - Sketch promoted to semantic/episodic memory
```

---

## 5. API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/orchestration/checkpoints` | Create approval gate |
| `GET` | `/v1/orchestration/checkpoints/{id}` | Check status (agent polls) |
| `POST` | `/v1/orchestration/checkpoints/{id}/approve` | Human approve |
| `POST` | `/v1/orchestration/checkpoints/{id}/reject` | Human reject |
| `POST` | `/v1/orchestration/sentinels` | Create sentinel |
| `GET` | `/v1/orchestration/sentinels` | List active sentinels |
| `DELETE` | `/v1/orchestration/sentinels/{id}` | Remove sentinel |
| `POST` | `/v1/orchestration/routines` | Create workflow template |
| `POST` | `/v1/orchestration/sketches` | Create ephemeral sketch |
| `POST` | `/v1/orchestration/crystals` | Crystallize sketch |

---

## 6. DB Schema

```sql
CREATE TABLE agent_checkpoints (
    id UUID PRIMARY KEY,
    action_id UUID, tenant_id TEXT NOT NULL,
    agent_id TEXT, type TEXT, message TEXT,
    status TEXT DEFAULT 'pending',
    timeout_minutes INT DEFAULT 60,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_at TIMESTAMPTZ, resolved_by TEXT
);

CREATE TABLE agent_sentinels (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL, agent_id TEXT,
    trigger_type TEXT, trigger_condition JSONB,
    action_type TEXT, action_payload JSONB,
    status TEXT DEFAULT 'active',
    last_triggered_at TIMESTAMPTZ
);

CREATE TABLE agent_sketches (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL, agent_id TEXT,
    content TEXT, expires_at TIMESTAMPTZ
);
```

---

## 7. Acceptance Criteria

- [ ] Checkpoint created → agent can poll status
- [ ] Human approval via API unblocks agent within 1s
- [ ] Timeout → auto-reject (NATS event)
- [ ] Sentinel triggers within 30s of condition met
- [ ] Sketch auto-expires after TTL
- [ ] Crystal → memory stored in correct engine type

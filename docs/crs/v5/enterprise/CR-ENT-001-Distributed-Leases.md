# Change Request: CR-ENT-001 — Distributed Lease System

**CR ID:** CR-ENT-001
**Component:** `backend/services/orchestration-service`
**Priority:** 🟡 High
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S8 — Multi-Agent Coordination](../../../bussiness/solutions/S8-multi-agent-coordination.md)
**Features:** [F11](../../../features/11-multi-agent-orchestration/README.md)
**ADR:** [ADR-011](../../../adr/ADR-011-distributed-leases.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-08 | AI Agent Developer | Multi-agent race conditions — Agent A và B cùng update memory → conflict |

**Scenario:** 2 agents cùng update user profile → last-write-wins → data loss.
**After:** Lease system → chỉ 1 agent giữ lock tại 1 thời điểm.

---

## 2. Lease + Signal Architecture

```
Agent A: POST /v1/orchestration/lease  { resource: "user_profile:u1", ttl: 30s }
→ { lease_id: "l_abc", acquired: true, expires_at: "..." }

Agent B: POST /v1/orchestration/lease  { resource: "user_profile:u1" }
→ { acquired: false, held_by: "agent_a", expires_at: "..." }  # retry later

Agent A (done): DELETE /v1/orchestration/lease/l_abc
→ Lease released, NATS event: orchestration.lease.released

Signal system:
  Agent A → Signal → Agent B: { type: "handoff", payload: {...} }
```

---

## 3. API Contract

```http
# Acquire lease
POST /v1/orchestration/lease
{
  "agent_id": "agent_a",
  "resource": "user_profile:u1",
  "ttl_seconds": 30
}
→ 200: { "lease_id": "l_abc", "acquired": true, "expires_at": "..." }
→ 409: { "acquired": false, "held_by": "agent_b", "expires_at": "..." }

# Release lease
DELETE /v1/orchestration/leases/{lease_id}
→ 200: { "released": true }

# Send signal between agents
POST /v1/orchestration/signals
{
  "from_agent": "agent_a",
  "to_agent": "agent_b",
  "type": "handoff",  // handoff | alert | update | query
  "payload": { "context": "Task done, continuing from step 3" }
}

# Get pending signals
GET /v1/orchestration/signals?agent_id=agent_b
```

---

## 4. Thay đổi đề xuất

### 4.1 PostgreSQL Lease Table

```sql
CREATE TABLE leases (
    lease_id    UUID PRIMARY KEY,
    resource    TEXT NOT NULL,
    tenant_id   TEXT NOT NULL,
    agent_id    TEXT NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (resource, tenant_id)  -- one lease per resource per tenant
);

-- Acquire with optimistic insert (steal if expired)
INSERT INTO leases (lease_id, resource, tenant_id, agent_id, expires_at)
VALUES ($1, $2, $3, $4, NOW() + $5::interval)
ON CONFLICT (resource, tenant_id) DO UPDATE
    SET agent_id = $4, expires_at = NOW() + $5::interval, lease_id = $1
    WHERE leases.expires_at < NOW();  -- only steal if expired
```

---

## 5. Acceptance Criteria

- [ ] Lease acquire: `< 10ms` response
- [ ] TTL auto-expire: nếu agent crash, lease expires sau TTL
- [ ] Only lease holder có thể release
- [ ] Signal delivery qua NATS (at-least-once)
- [ ] `GET /v1/orchestration/leases` liệt kê active leases (admin)
- [ ] Sentinel watchers: event-driven condition triggers (không polling)

# ADR-011 — Distributed Leases cho Multi-Agent Coordination

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-05 |
| **Deciders** | Platform + ML Team |
| **Feature** | F11 (Multi-Agent Orchestration) |

---

## Context

Multi-agent systems có vấn đề **race conditions**: nhiều agents cùng update 1 memory resource → inconsistent state, duplicate work, conflicting updates.

Ví dụ cụ thể:
- Agent A và Agent B cùng đọc user profile → cùng update → last-write-wins → Agent A's update bị mất
- 2 agents cùng trigger memory flush → duplicate processing
- Agent đang write session summary → Agent B read partial data

---

## Decision

**Distributed Lease mechanism với PostgreSQL + NATS-backed Signals.**

```go
// Acquire exclusive lease trước khi modify shared resource
type LeaseRequest struct {
    Resource  string        // "user_profile:u1", "session:s1", "memory:m1"
    AgentID   string
    TenantID  string
    TTL       time.Duration // max hold time (e.g., 30s)
}

// Lease stored in PostgreSQL với expires_at
func (s *LeaseStore) Acquire(ctx context.Context, req *LeaseRequest) (*Lease, error) {
    // Optimistic insert: INSERT ... ON CONFLICT DO NOTHING
    // If conflict: check if expired → steal if expired, else return ErrLeaseHeld
    result, err := s.db.ExecContext(ctx, `
        INSERT INTO leases (lease_id, resource, agent_id, tenant_id, expires_at)
        VALUES ($1, $2, $3, $4, NOW() + $5::interval)
        ON CONFLICT (resource, tenant_id) DO UPDATE
            SET agent_id = $3, expires_at = NOW() + $5::interval
            WHERE leases.expires_at < NOW()  -- only steal if expired
    `, uuid.New(), req.Resource, req.AgentID, req.TenantID, req.TTL)
    ...
}

// Signal system (NATS-backed): agent-to-agent communication
type SignalType string
const (
    SignalHandoff SignalType = "handoff" // "I'm done, you can take over"
    SignalAlert   SignalType = "alert"   // "Something went wrong"
    SignalUpdate  SignalType = "update"  // "I've changed the shared state"
    SignalQuery   SignalType = "query"   // "What is your current state?"
)
```

**Sketch → Crystal pattern cho safe collaborative writing:**

```
Sketch (draft, mutable):     Working memory (agent can write freely)
Crystal (committed, immutable): Permanent memory (needs lease + validation)
Transition: Lease → validate → commit → release
```

---

## Consequences

**Positive:**
- **Zero race conditions** cho shared resource access
- TTL prevents deadlock (lease auto-expires nếu agent crash)
- Signals enable agent handoff và coordination without polling
- Sentinels: event-driven condition triggers (no polling required)

**Negative:**
- Lease acquisition adds latency (~5ms per acquire)
- Agents must handle `ErrLeaseHeld` gracefully (retry logic)
- Complex for simple single-agent scenarios (over-engineering if only 1 agent)

**Mitigations:**
- Lease is optional: `orchestration.WithLease(false)` cho non-critical writes
- Exponential backoff retry built into SDK
- Sentinel (not polling): `WatchFor("memory.updated", condition)` instead of loop

---

## Alternatives Considered

### A1 — Database row-level locking (SELECT FOR UPDATE)
- **Rejected:** Không work cross-service; deadlock risk; no TTL auto-release; không support cross-database (Neo4j)

### A2 — Redis SETNX (distributed mutex)
- **Considered:** Simpler; but PostgreSQL already deployed; Redis SETNX TTL management tricky (clock drift)
- **Rejected:** Thêm Redis dependency cho feature đã có PostgreSQL solution

### A3 — Optimistic concurrency (version counter)
- **Partially used:** Jaccard versioning trong Memory Lifecycle (ADR-related); but optimistic không đủ cho exclusive writes

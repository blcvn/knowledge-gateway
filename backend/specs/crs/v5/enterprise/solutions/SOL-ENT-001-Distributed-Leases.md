# SOL-ENT-001 — Solution: Distributed Lease System

| Field | Value |
|---|---|
| **Solution ID** | SOL-ENT-001 |
| **CR** | [CR-ENT-001](../../../../docs/crs/v5/enterprise/CR-ENT-001-Distributed-Leases.md) |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) §orchestration-service |
| **Status** | Open |
| **Priority** | 🟡 High |

---

## 1. Giải pháp

`orchestration-service` đã có lease domain model. Cần:
1. Redis-backed distributed mutex với TTL
2. NATS-backed signal routing
3. Action DAG state machine

### 1.1 `services/orchestration-service/internal/usecase/lease.go` [MODIFY]

```go
type LeaseManager struct {
    redis  *redis.Client
    pub    port.EventPublisher
}

const leasePrefix = "vnp:lease:"

// AcquireLease — Redis SET NX với TTL
func (m *LeaseManager) AcquireLease(ctx context.Context, req *AcquireLeaseRequest) (*Lease, error) {
    key := leasePrefix + req.TenantID + ":" + req.ResourceID
    leaseID := uuid.NewString()
    ttl := req.TTL
    if ttl == 0 { ttl = 30 * time.Second }

    ok, err := m.redis.SetNX(ctx, key, leaseID, ttl).Result()
    if err != nil { return nil, err }
    if !ok { return nil, ErrLeaseConflict }

    lease := &Lease{
        ID: leaseID, ResourceID: req.ResourceID,
        TenantID: req.TenantID, AgentID: req.AgentID,
        ExpiresAt: time.Now().Add(ttl),
    }
    m.pub.Publish(ctx, "agent.lease.acquired", lease)
    return lease, nil
}

// ReleaseLease — verify leaseID trước khi delete (Lua script để atomic)
func (m *LeaseManager) ReleaseLease(ctx context.Context, leaseID, resourceID, tenantID string) error {
    key := leasePrefix + tenantID + ":" + resourceID
    script := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("DEL", KEYS[1])
        else
            return 0
        end`
    result, _ := m.redis.Eval(ctx, script, []string{key}, leaseID).Int()
    if result == 0 { return ErrNotLeaseOwner }
    m.pub.Publish(ctx, "agent.lease.released", map[string]string{"lease_id": leaseID})
    return nil
}

// RenewLease — extend TTL nếu còn là owner
func (m *LeaseManager) RenewLease(ctx context.Context, leaseID, resourceID, tenantID string, ttl time.Duration) error {
    key := leasePrefix + tenantID + ":" + resourceID
    script := `
        if redis.call("GET", KEYS[1]) == ARGV[1] then
            return redis.call("EXPIRE", KEYS[1], ARGV[2])
        else
            return 0
        end`
    result, _ := m.redis.Eval(ctx, script, []string{key}, leaseID, int(ttl.Seconds())).Int()
    if result == 0 { return ErrNotLeaseOwner }
    return nil
}
```

### 1.2 Signal Router (`signal.go`)

```go
type SignalRouter struct {
    nats *nats.Conn
}

// SendSignal — handoff | alert | update | query
func (r *SignalRouter) SendSignal(ctx context.Context, req *SendSignalRequest) error {
    subject := fmt.Sprintf("agent.signal.%s.%s", req.TenantID, req.TargetAgentID)
    data, _ := json.Marshal(Signal{
        Type: req.SignalType, Payload: req.Payload,
        From: req.FromAgentID, Timestamp: time.Now(),
    })
    return r.nats.PublishMsg(&nats.Msg{Subject: subject, Data: data})
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/orchestration-service/internal/usecase/lease.go` | MODIFY — Redis NX + Lua |
| `services/orchestration-service/internal/usecase/signal.go` | NEW |
| `services/orchestration-service/internal/adapter/grpc/handler.go` | MODIFY |
| `deployment/dev/migrations/0XX_leases.sql` | VERIFY — table exists |

---

## 3. Acceptance Criteria

- [ ] Lease acquire: atomic (Redis NX), p99 < 5ms
- [ ] Lease conflict returns `409 Conflict`
- [ ] Lease release only by owner (Lua atomic check)
- [ ] Signal routing via NATS < 10ms
- [ ] Lease TTL auto-expires (no orphan leases)

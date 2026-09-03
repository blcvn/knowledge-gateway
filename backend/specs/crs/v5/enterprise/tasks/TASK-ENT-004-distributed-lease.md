# TASK-ENT-004 — Distributed Lease System (Redis NX + Lua)

| Field | Value |
|---|---|
| **Task ID** | TASK-ENT-004 |
| **Wave** | 2 |
| **Solution** | [SOL-ENT-001](../solutions/SOL-ENT-001-Distributed-Leases.md) §1.1 |
| **Component** | `services/orchestration-service/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 5h |

---

## Mục tiêu

Implement Redis-backed distributed lease: AcquireLease, ReleaseLease, RenewLease.

---

## Công việc cụ thể

### `services/orchestration-service/internal/usecase/lease.go` [MODIFY]

```go
const leasePrefix = "vnp:lease:"

type LeaseManager struct {
    redis *redis.Client
    pub   port.EventPublisher
}

// AcquireLease — Redis SET NX (atomic, no race condition)
func (m *LeaseManager) AcquireLease(ctx context.Context, req *AcquireLeaseRequest) (*Lease, error) {
    ttl := req.TTL
    if ttl == 0 { ttl = 30 * time.Second }
    key := leasePrefix + req.TenantID + ":" + req.ResourceID
    leaseID := uuid.NewString()

    ok, err := m.redis.SetNX(ctx, key, leaseID, ttl).Result()
    if err != nil { return nil, fmt.Errorf("redis error: %w", err) }
    if !ok { return nil, ErrLeaseConflict }

    lease := &Lease{
        ID: leaseID, ResourceID: req.ResourceID,
        TenantID: req.TenantID, AgentID: req.AgentID,
        ExpiresAt: time.Now().Add(ttl),
    }
    m.pub.Publish(ctx, "agent.lease.acquired", lease)
    return lease, nil
}

// ReleaseLease — Lua script: verify owner then DEL (atomic)
var releaseScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end`)

func (m *LeaseManager) ReleaseLease(ctx context.Context, tenantID, resourceID, leaseID string) error {
    key := leasePrefix + tenantID + ":" + resourceID
    result, err := releaseScript.Run(ctx, m.redis, []string{key}, leaseID).Int()
    if err != nil { return err }
    if result == 0 { return ErrNotLeaseOwner }
    m.pub.Publish(ctx, "agent.lease.released", map[string]string{"lease_id": leaseID})
    return nil
}

// RenewLease — Lua script: verify owner then EXPIRE
var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("EXPIRE", KEYS[1], ARGV[2])
else
    return 0
end`)

func (m *LeaseManager) RenewLease(ctx context.Context, tenantID, resourceID, leaseID string, ttl time.Duration) error {
    key := leasePrefix + tenantID + ":" + resourceID
    result, err := renewScript.Run(ctx, m.redis, []string{key}, leaseID, int(ttl.Seconds())).Int()
    if err != nil { return err }
    if result == 0 { return ErrNotLeaseOwner }
    return nil
}

var (
    ErrLeaseConflict = errors.New("resource already locked")
    ErrNotLeaseOwner = errors.New("lease not owned by caller")
)
```

### Unit tests

```go
func TestAcquireLease_Success(t *testing.T) { ... }
func TestAcquireLease_Conflict_Returns409Error(t *testing.T) { ... }
func TestReleaseLease_NotOwner_ReturnsError(t *testing.T) { ... }
func TestReleaseLease_Atomic_NoConcurrentRelease(t *testing.T) { ... }
func TestRenewLease_ExtendsExpiry(t *testing.T) { ... }
```

---

## Acceptance Criteria

- [ ] AcquireLease: atomic via Redis SET NX
- [ ] ConflictError on second acquire of same resource
- [ ] ReleaseLease: Lua script verifies ownership
- [ ] RenewLease: only owner can extend TTL
- [ ] Lease auto-expires (no orphan leases after crash)
- [ ] NATS events published for acquire/release

## Files

```
services/orchestration-service/internal/usecase/lease.go       [MODIFY]
services/orchestration-service/internal/usecase/lease_test.go  [NEW]
```

---

**Ghi chú audit:** orchestration-service/internal/orchestration/leases.go: LeaseService.Acquire() (sync.Map + PostgreSQL) + Renew() + Release() + SweepExpired()

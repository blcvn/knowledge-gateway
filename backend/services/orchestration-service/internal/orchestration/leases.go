package orchestration

import (
    "context"
    "sync"
    "time"

    "github.com/google/uuid"
    "vnp-memory/services/orchestration-service/internal/domain"
    "vnp-memory/services/orchestration-service/internal/usecase/port"
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

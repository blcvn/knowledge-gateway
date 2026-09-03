package port

import (
    "context"
    "time"
    "vnp-memory/services/orchestration-service/internal/domain"
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

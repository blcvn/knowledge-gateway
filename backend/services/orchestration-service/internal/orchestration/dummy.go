package orchestration

import (
    "context"
    "vnp-memory/services/orchestration-service/internal/domain"
    orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
)

// SignalService is a no-op stub used when NATS is not configured.
// For production, replace with NATSSignalRouter (signal.go).
// SOL-ENT-001 / TASK-ENT-005
type SignalService struct{}
func (s *SignalService) ReapExpired(ctx context.Context)   {}
func (s *SignalService) DeleteExpired(ctx context.Context) {}
func (s *SignalService) Send(ctx context.Context, req *orchpb.SendSignalRequest) (*domain.Signal, error) {
    // Stub: returns empty signal without NATS.
    // Use NATSSignalRouter for real inter-agent communication.
    return &domain.Signal{ID: "noop"}, nil
}

type CheckpointService struct{}
func (c *CheckpointService) ReapExpired(ctx context.Context) {}
func (c *CheckpointService) AutoRejectExpired(ctx context.Context) {}
func (c *CheckpointService) Create(ctx context.Context, req *orchpb.CreateCheckpointRequest) (*domain.Checkpoint, error) { return &domain.Checkpoint{}, nil }
func (c *CheckpointService) Resolve(ctx context.Context, id, status, by, reason string) error { return nil }

type RoutineService struct{}
func (r *RoutineService) ReapExpired(ctx context.Context) {}

func (s *SketchService) ReapExpired(ctx context.Context) {}

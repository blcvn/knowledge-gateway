package orchestration

import (
    "context"
    "vnp-memory/services/orchestration-service/internal/domain"
    orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
)

type SignalService struct{}
func (s *SignalService) ReapExpired(ctx context.Context) {}
func (s *SignalService) DeleteExpired(ctx context.Context) {}
func (s *SignalService) Send(ctx context.Context, req *orchpb.SendSignalRequest) (*domain.Signal, error) { return &domain.Signal{}, nil }

type CheckpointService struct{}
func (c *CheckpointService) ReapExpired(ctx context.Context) {}
func (c *CheckpointService) AutoRejectExpired(ctx context.Context) {}
func (c *CheckpointService) Create(ctx context.Context, req *orchpb.CreateCheckpointRequest) (*domain.Checkpoint, error) { return &domain.Checkpoint{}, nil }
func (c *CheckpointService) Resolve(ctx context.Context, id, status, by, reason string) error { return nil }

type RoutineService struct{}
func (r *RoutineService) ReapExpired(ctx context.Context) {}

func (s *SketchService) ReapExpired(ctx context.Context) {}

package grpc

import (
	"context"
	"errors"
	"time"

	"vnp-memory/services/orchestration-service/internal/domain"
	"vnp-memory/services/orchestration-service/internal/orchestration"
	"vnp-memory/services/orchestration-service/internal/usecase/port"

	orchpb "github.com/vnp-memory/api/proto/orchestration/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type OrchestrationHandler struct {
	orchpb.UnimplementedOrchestrationServiceServer
	actions     *orchestration.ActionService
	leases      *orchestration.LeaseService
	signals     *orchestration.SignalService
	checkpoints *orchestration.CheckpointService
	sketches    *orchestration.SketchService
	sentinels   *orchestration.SentinelService
	routines    *orchestration.RoutineService
	actionRepo  port.IActionRepo
	signalRepo  port.ISignalRepo
	crystalRepo port.ICrystalRepo
}

// ── Actions ───────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) CreateAction(ctx context.Context, req *orchpb.CreateActionRequest) (*orchpb.CreateActionResponse, error) {
	action, err := h.actions.Create(ctx, orchestration.CreateActionRequest{
		TenantID: req.TenantId, Project: req.Project, AgentID: req.AgentId,
		Title: req.Title, Description: req.Description, Priority: int(req.Priority),
		Requires: req.Requires, ConflictsWith: req.ConflictsWith, Tags: req.Tags,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create action: %v", err)
	}
	return &orchpb.CreateActionResponse{ActionId: action.ID}, nil
}

func (h *OrchestrationHandler) GetAction(ctx context.Context, req *orchpb.GetActionRequest) (*orchpb.GetActionResponse, error) {
	action, err := h.actionRepo.Get(ctx, req.ActionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "action not found")
	}
	return &orchpb.GetActionResponse{Action: mapAction(*action)}, nil
}

func (h *OrchestrationHandler) ListActions(ctx context.Context, req *orchpb.ListActionsRequest) (*orchpb.ListActionsResponse, error) {
	actions, err := h.actionRepo.List(ctx, req.TenantId, req.Status, int(req.Limit))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list actions: %v", err)
	}
	return mapListActionsResponse(actions), nil
}

func (h *OrchestrationHandler) UpdateAction(ctx context.Context, req *orchpb.UpdateActionRequest) (*orchpb.UpdateActionResponse, error) {
	err := h.actions.UpdateStatus(ctx, req.ActionId, domain.ActionStatus(req.Status), req.Result)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid transition: %v", err)
		}
		return nil, status.Errorf(codes.Internal, "update action: %v", err)
	}
	return &orchpb.UpdateActionResponse{Ok: true}, nil
}

// ── Leases ────────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) AcquireLease(ctx context.Context, req *orchpb.AcquireLeaseRequest) (*orchpb.AcquireLeaseResponse, error) {
	ttl := time.Duration(req.TtlSecs) * time.Second
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	lease, err := h.leases.Acquire(ctx, req.ActionId, req.AgentId, ttl)
	if err != nil {
		var ce domain.ErrLeaseConflictDetail
		if errors.As(err, &ce) {
			return &orchpb.AcquireLeaseResponse{Conflict: true, ConflictingAgent: ce.ActiveLease.AgentID}, nil
		}
		return nil, status.Errorf(codes.Internal, "acquire lease: %v", err)
	}
	return &orchpb.AcquireLeaseResponse{LeaseId: lease.ID}, nil
}

func (h *OrchestrationHandler) RenewLease(ctx context.Context, req *orchpb.RenewLeaseRequest) (*orchpb.RenewLeaseResponse, error) {
	err := h.leases.Renew(ctx, req.LeaseId, time.Duration(req.ExtendSecs)*time.Second)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "renew lease: %v", err)
	}
	return &orchpb.RenewLeaseResponse{Ok: true}, nil
}

func (h *OrchestrationHandler) ReleaseLease(ctx context.Context, req *orchpb.ReleaseLeaseRequest) (*orchpb.ReleaseLeaseResponse, error) {
	err := h.leases.Release(ctx, req.LeaseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "release lease: %v", err)
	}
	return &orchpb.ReleaseLeaseResponse{Ok: true}, nil
}

// ── Signals ───────────────────────────────────────────────────────────────

func (h *OrchestrationHandler) SendSignal(ctx context.Context, req *orchpb.SendSignalRequest) (*orchpb.SendSignalResponse, error) {
	signal, err := h.signals.Send(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send signal: %v", err)
	}
	return &orchpb.SendSignalResponse{SignalId: signal.ID}, nil
}

func (h *OrchestrationHandler) ListSignals(ctx context.Context, req *orchpb.ListSignalsRequest) (*orchpb.ListSignalsResponse, error) {
	signals, err := h.signalRepo.List(ctx, req.TenantId, req.AgentId, req.UnreadOnly)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list signals: %v", err)
	}
	return mapListSignalsResponse(signals), nil
}

// ── Checkpoints ───────────────────────────────────────────────────────────

func (h *OrchestrationHandler) CreateCheckpoint(ctx context.Context, req *orchpb.CreateCheckpointRequest) (*orchpb.CreateCheckpointResponse, error) {
	cp, err := h.checkpoints.Create(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create checkpoint: %v", err)
	}
	return &orchpb.CreateCheckpointResponse{CheckpointId: cp.ID}, nil
}

func (h *OrchestrationHandler) ApproveCheckpoint(ctx context.Context, req *orchpb.ApproveCheckpointRequest) (*orchpb.ApproveCheckpointResponse, error) {
	err := h.checkpoints.Resolve(ctx, req.CheckpointId, "approved", req.ApprovedBy, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "approve: %v", err)
	}
	return &orchpb.ApproveCheckpointResponse{Ok: true}, nil
}

// ── Sketches & Crystals ───────────────────────────────────────────────────

func (h *OrchestrationHandler) PromoteSketch(ctx context.Context, req *orchpb.PromoteSketchRequest) (*orchpb.PromoteSketchResponse, error) {
	crystal, err := h.sketches.Promote(ctx, req.SketchId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "promote sketch: %v", err)
	}
	return &orchpb.PromoteSketchResponse{CrystalId: crystal.ID}, nil
}

func (h *OrchestrationHandler) GetCrystal(ctx context.Context, req *orchpb.GetCrystalRequest) (*orchpb.GetCrystalResponse, error) {
	crystal, err := h.crystalRepo.Get(ctx, req.CrystalId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "crystal not found")
	}
	return mapCrystalResponse(crystal), nil
}

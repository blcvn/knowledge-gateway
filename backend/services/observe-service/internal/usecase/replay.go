package usecase

import (
	"context"

	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/replay"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type ReplayUseCase struct {
	sessionRepo port.ISessionRepo
	obsRepo     port.IObservationRepo
}

type ListReplaySessionsRequest struct {
	TenantID string
	Project  string
	Limit    int
	Offset   int
}

type ListReplaySessionsResponse struct {
	Sessions []domain.Session
	Total    int
}

func (uc *ReplayUseCase) ListSessions(ctx context.Context, req ListReplaySessionsRequest) (*ListReplaySessionsResponse, error) {
	sessions, err := uc.sessionRepo.List(ctx, req.TenantID, "completed", req.Project, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}
	return &ListReplaySessionsResponse{Sessions: sessions, Total: len(sessions)}, nil
}

type LoadReplayRequest struct {
	SessionID string
	TenantID  string
	HookTypes []string
	FromIndex int
	ToIndex   int
}

type LoadReplayResponse struct {
	Timeline replay.Timeline
}

func (uc *ReplayUseCase) LoadTimeline(ctx context.Context, req LoadReplayRequest) (*LoadReplayResponse, error) {
	session, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	obs, err := uc.obsRepo.ListCompressed(ctx, req.SessionID, 500, 0)
	if err != nil {
		return nil, err
	}

	timeline := replay.BuildTimeline(req.SessionID, session.Project, obs)
	if len(req.HookTypes) > 0 || req.FromIndex > 0 || req.ToIndex > 0 {
		timeline = timeline.Filter(req.HookTypes, req.FromIndex, req.ToIndex)
	}

	return &LoadReplayResponse{Timeline: timeline}, nil
}

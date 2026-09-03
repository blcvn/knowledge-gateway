package usecase

import (
	"context"
	"time"

	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type EndSessionUseCase struct {
	sessionRepo port.ISessionRepo
	obsRepo     port.IObservationRepo
	publisher   port.IEventPublisher
}

type EndSessionRequest struct {
	SessionID string
	TenantID  string
}
type EndSessionResponse struct {
	SessionID        string
	Status           string
	ObservationCount int
}

func (uc *EndSessionUseCase) Execute(ctx context.Context, req EndSessionRequest) (*EndSessionResponse, error) {
	session, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, domain.ErrSessionNotFound
	}
	if session.Status == "completed" {
		return nil, domain.ErrSessionEnded
	}

	now := time.Now()
	session.Status = "completed"
	session.EndedAt = &now

	if err := uc.sessionRepo.UpdateStatus(ctx, req.SessionID, "completed"); err != nil {
		return nil, err
	}

	count, _ := uc.sessionRepo.GetObsCount(ctx, req.SessionID)

	uc.publisher.Publish(ctx, "agentmemory.session.ended", map[string]any{
		"session_id":        req.SessionID,
		"tenant_id":         req.TenantID,
		"observation_count": count,
		"ended_at":          now,
	})

	return &EndSessionResponse{
		SessionID:        req.SessionID,
		Status:           "completed",
		ObservationCount: count,
	}, nil
}

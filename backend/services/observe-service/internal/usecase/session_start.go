package usecase

import (
	"context"

	"github.com/vnp-memory/services/observe-service/internal/domain"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type CreateSessionUseCase struct {
	sessionRepo port.ISessionRepo
	publisher   port.IEventPublisher
}

type CreateSessionRequest struct {
	TenantID    string
	Project     string
	CWD         string
	Model       string
	AgentID     string
	FirstPrompt string
}

type CreateSessionResponse struct {
	SessionID string
	Status    string
}

func (uc *CreateSessionUseCase) Execute(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
	session := domain.NewSession(req.TenantID, req.Project, req.CWD, req.Model, req.AgentID)
	session.FirstPrompt = req.FirstPrompt

	if err := uc.sessionRepo.Save(ctx, session); err != nil {
		return nil, err
	}

	uc.publisher.Publish(ctx, "agentmemory.session.started", map[string]any{
		"session_id": session.ID,
		"tenant_id":  session.TenantID,
		"project":    session.Project,
		"agent_id":   session.AgentID,
		"started_at": session.StartedAt,
	})

	return &CreateSessionResponse{SessionID: session.ID, Status: "active"}, nil
}

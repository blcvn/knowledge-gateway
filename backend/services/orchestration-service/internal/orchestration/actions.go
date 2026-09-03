package orchestration

import (
    "context"
    "time"

    "github.com/google/uuid"
    "vnp-memory/services/orchestration-service/internal/domain"
    "vnp-memory/services/orchestration-service/internal/usecase/port"
)

type ActionService struct {
    repo      port.IActionRepo
    publisher port.IEventPublisher
}

func (s *ActionService) Create(ctx context.Context, req CreateActionRequest) (*domain.Action, error) {
    action := domain.Action{
        ID:           uuid.New().String(),
        TenantID:     req.TenantID,
        Project:      req.Project,
        AgentID:      req.AgentID,
        Title:        req.Title,
        Description:  req.Description,
        Status:       domain.ActionPending,
        Priority:     req.Priority,
        Requires:     req.Requires,
        ConflictsWith: req.ConflictsWith,
        Tags:         req.Tags,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    if action.Priority == 0 { action.Priority = 50 }

    if err := s.repo.Save(ctx, action); err != nil { return nil, err }
    return &action, nil
}

func (s *ActionService) UpdateStatus(ctx context.Context, actionID string, next domain.ActionStatus, result string) error {
    action, err := s.repo.Get(ctx, actionID)
    if err != nil { return domain.ErrActionNotFound }

    if !action.Status.CanTransitionTo(next) {
        return domain.ErrInvalidTransition
    }

    now := time.Now()
    action.Status = next
    action.Result = result
    action.UpdatedAt = now
    if next == domain.ActionDone || next == domain.ActionFailed || next == domain.ActionCancelled {
        action.CompletedAt = &now
    }

    if err := s.repo.Update(ctx, *action); err != nil { return err }

    s.publisher.Publish(ctx, "agentmemory.action.completed", map[string]any{
        "action_id": actionID, "status": string(next),
    })
    return nil
}

type CreateActionRequest struct {
    TenantID     string
    Project      string
    AgentID      string
    Title        string
    Description  string
    Priority     int
    Requires     []string
    ConflictsWith []string
    Tags         []string
}

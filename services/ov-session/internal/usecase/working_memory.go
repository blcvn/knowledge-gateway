package usecase

import (
	"context"
	"time"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
	"github.com/vnp-memory/services/ov-session/internal/domain/repository"
)

type WorkingMemoryUseCase interface {
	GetWorkingMemory(ctx context.Context, sessionID string) (*model.WorkingMemory, error)
	UpdateWorkingMemory(ctx context.Context, wm *model.WorkingMemory) (*model.WorkingMemory, error)
}

type wmUseCaseImpl struct {
	sessionRepo repository.SessionRepository
}

func NewWorkingMemoryUseCase(sessionRepo repository.SessionRepository) WorkingMemoryUseCase {
	return &wmUseCaseImpl{
		sessionRepo: sessionRepo,
	}
}

func (uc *wmUseCaseImpl) GetWorkingMemory(ctx context.Context, sessionID string) (*model.WorkingMemory, error) {
	return uc.sessionRepo.GetWorkingMemory(ctx, sessionID)
}

func (uc *wmUseCaseImpl) UpdateWorkingMemory(ctx context.Context, wm *model.WorkingMemory) (*model.WorkingMemory, error) {
	wm.UpdatedAt = time.Now()
	err := uc.sessionRepo.UpdateWorkingMemory(ctx, wm)
	if err != nil {
		return nil, err
	}
	return wm, nil
}

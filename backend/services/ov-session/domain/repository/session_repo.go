package repository

import (
	"context"

	"github.com/vnp-memory/services/ov-session/domain/model"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	Update(ctx context.Context, session *model.Session) error

	// Working Memory
	GetWorkingMemory(ctx context.Context, sessionID string) (*model.WorkingMemory, error)
	UpdateWorkingMemory(ctx context.Context, wm *model.WorkingMemory) error

	// Memories
	SaveMemory(ctx context.Context, memory *model.CandidateMemory) error
}

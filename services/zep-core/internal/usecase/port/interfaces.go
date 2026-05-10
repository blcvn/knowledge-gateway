package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/zep-core/internal/domain/memory"
	"github.com/vnp-community/vnp-memory/services/zep-core/internal/domain/thread"
	"github.com/vnp-community/vnp-memory/services/zep-core/internal/domain/user"
)

type UserUseCase interface {
	CreateUser(ctx context.Context, tenantID uuid.UUID, userID, email string) (*user.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*user.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type ThreadUseCase interface {
	CreateThread(ctx context.Context, tenantID uuid.UUID, userID string) (*thread.Thread, error)
	GetThread(ctx context.Context, id uuid.UUID) (*thread.Thread, error)
	EndThread(ctx context.Context, id uuid.UUID) error
	UpsertSession(ctx context.Context, threadID uuid.UUID) (*thread.Session, error)
}

type MemoryUseCase interface {
	PutMemory(ctx context.Context, threadID uuid.UUID, msgs []memory.Message) error
	GetContext(ctx context.Context, threadID uuid.UUID, maxTokens int) (*memory.ContextAssembly, error)
}

type UserRepository interface {
	Create(ctx context.Context, u *user.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ThreadRepository interface {
	Create(ctx context.Context, t *thread.Thread) error
	FindByID(ctx context.Context, id uuid.UUID) (*thread.Thread, error)
	CreateSession(ctx context.Context, s *thread.Session) error
}

type MessageRepository interface {
	BatchCreate(ctx context.Context, msgs []memory.Message) error
	FindByThread(ctx context.Context, threadID uuid.UUID, limit int) ([]memory.Message, error)
}

type EventPublisher interface {
	PublishMemoryPut(ctx context.Context, tenantID, threadID uuid.UUID) error
}

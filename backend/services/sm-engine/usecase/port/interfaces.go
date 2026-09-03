package port

import (
	"context"
	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/sm-engine/domain/document"
	"github.com/vnp-community/vnp-memory/services/sm-engine/domain/memory"
	"github.com/vnp-community/vnp-memory/services/sm-engine/domain/profile"
)

type DocumentUseCase interface {
	SaveDocument(ctx context.Context, tenantID, userID uuid.UUID, title, url, content string) (*document.Document, error)
	GetDocument(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) error
}

type MemoryUseCase interface {
	CreateMemory(ctx context.Context, tenantID, userID uuid.UUID, content, category string) (*memory.Memory, error)
	GetMemory(ctx context.Context, id uuid.UUID) (*memory.Memory, error)
	SearchMemories(ctx context.Context, tenantID, userID uuid.UUID, query string, limit int) ([]memory.Memory, error)
}

type ProfileUseCase interface {
	GetProfile(ctx context.Context, tenantID, userID uuid.UUID) (*profile.Profile, error)
	UpdatePreferences(ctx context.Context, tenantID, userID uuid.UUID, prefs []profile.StaticPreference) error
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *document.Document) error
	FindByID(ctx context.Context, id uuid.UUID) (*document.Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type MemoryRepository interface {
	Create(ctx context.Context, mem *memory.Memory) error
	FindByID(ctx context.Context, id uuid.UUID) (*memory.Memory, error)
	SearchByUser(ctx context.Context, tenantID, userID uuid.UUID, query string, limit int) ([]memory.Memory, error)
}

type ProfileRepository interface {
	FindByUser(ctx context.Context, tenantID, userID uuid.UUID) (*profile.Profile, error)
	Upsert(ctx context.Context, p *profile.Profile) error
}

type EventPublisher interface {
	PublishDocumentSaved(ctx context.Context, tenantID, docID uuid.UUID) error
}

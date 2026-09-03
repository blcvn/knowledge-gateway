// Package port defines output interfaces for memory-service usecases.
package port

import (
	"context"

	"vnp-memory/services/memory-service/internal/domain/memobase"
	"vnp-memory/services/memory-service/internal/domain/sm"
	"vnp-memory/services/memory-service/internal/domain/zep"
)

// ── Memobase Repositories ──────────────────────────────────────────────────

// BlobRepository persists memory blobs with vector embeddings.
type BlobRepository interface {
	Create(ctx context.Context, blob *memobase.Blob) error
	List(ctx context.Context, userID, tenantID string, limit int) ([]*memobase.Blob, error)
	GetBufferSize(ctx context.Context, userID string) (int, error)
	SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*memobase.Blob, error)
}

// ProfileRepository persists user profiles.
type ProfileRepository interface {
	Upsert(ctx context.Context, userID, tenantID string, p *memobase.Profile) error
	GetByUser(ctx context.Context, userID, tenantID string) ([]*memobase.Profile, error)
}

// EventRepository persists user events (memobase).
type EventRepository interface {
	Create(ctx context.Context, evt *memobase.Event) error
	GetByUser(ctx context.Context, userID string, limit int) ([]*memobase.Event, error)
}

// MemoryEngine processes blob buffers (scoring + summarization).
type MemoryEngine interface {
	ProcessBuffer(ctx context.Context, userID string) error
	Summarize(blobs []*memobase.Blob) string
}

// EmbeddingService generates text embeddings.
type EmbeddingService interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// EventPublisher publishes domain events to NATS.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

// ── Zep Client ─────────────────────────────────────────────────────────────

// ZepClient wraps the Zep Cloud SDK or HTTP API.
type ZepClient interface {
	CreateUser(ctx context.Context, userID, email, firstName, lastName string, meta map[string]any) (*zep.ZepUser, error)
	GetUser(ctx context.Context, userID string) (*zep.ZepUser, error)
	UpdateUser(ctx context.Context, userID string, updates map[string]any) (*zep.ZepUser, error)
	PutMemory(ctx context.Context, sessionID string, mem *zep.ZepMemory) error
	GetMemory(ctx context.Context, sessionID string) (*zep.ZepMemory, error)
	GraphSearch(ctx context.Context, userID, query string, limit int) ([]*zep.GraphFact, error)
	SessionSearch(ctx context.Context, sessionID, query string, limit int) ([]*zep.ZepMessage, error)
	AddFact(ctx context.Context, userID string, fact *zep.GraphFact) error
}

// ── Supermemory Repositories ───────────────────────────────────────────────

// SMMemoryRepository persists SM memories.
type SMMemoryRepository interface {
	Create(ctx context.Context, mem *sm.SMMemory) error
	List(ctx context.Context, tenantID string, limit int) ([]*sm.SMMemory, error)
	SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*sm.SMMemory, error)
}

// SMDocumentRepository persists SM documents.
type SMDocumentRepository interface {
	Create(ctx context.Context, doc *sm.SMDocument) error
	FindByID(ctx context.Context, id string) (*sm.SMDocument, error)
}

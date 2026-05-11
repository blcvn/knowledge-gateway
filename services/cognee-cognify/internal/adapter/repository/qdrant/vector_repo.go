package qdrant

import (
	"context"

	"github.com/google/uuid"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// Client is a minimal interface for qdrant operations needed here
type Client interface {
	Upsert(ctx context.Context, collectionName string, id uuid.UUID, vector []float32, payload map[string]interface{}) error
}

type vectorRepository struct {
	client           Client
	chunkCollection  string
	entityCollection string
}

// NewVectorRepository creates a new Qdrant-backed VectorRepository.
func NewVectorRepository(client Client, chunkCol, entityCol string) port.VectorRepository {
	return &vectorRepository{
		client:           client,
		chunkCollection:  chunkCol,
		entityCollection: entityCol,
	}
}

func (r *vectorRepository) UpsertChunkEmbedding(ctx context.Context, tenantID string, chunkID uuid.UUID, text string, embedding []float32) error {
	payload := map[string]interface{}{
		"tenant_id": tenantID,
		"text":      text,
		"type":      "chunk",
	}
	return r.client.Upsert(ctx, r.chunkCollection, chunkID, embedding, payload)
}

func (r *vectorRepository) UpsertEntityEmbedding(ctx context.Context, tenantID string, entityID uuid.UUID, text string, embedding []float32) error {
	payload := map[string]interface{}{
		"tenant_id": tenantID,
		"text":      text,
		"type":      "entity",
	}
	return r.client.Upsert(ctx, r.entityCollection, entityID, embedding, payload)
}

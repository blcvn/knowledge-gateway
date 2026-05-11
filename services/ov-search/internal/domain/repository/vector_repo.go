package repository

import (
	"context"

	"vnp-memory/ov-search/internal/domain/model"
)

type VectorRepository interface {
	Upsert(ctx context.Context, vector model.EmbeddingVector, payload model.UpsertPayload) error
	Search(ctx context.Context, queryVector []float32, sparseVector []float32, accountID string, maxResults int) ([]model.SearchResult, error)
	Delete(ctx context.Context, path string, accountID string) error
}

package persistence

// Fallback for Qdrant using pgvector

import (
	"context"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
)

type pgVectorRepo struct {
	// mock db
}

func NewPGVectorRepo(dsn string) repository.VectorRepository {
	return &pgVectorRepo{}
}

func (r *pgVectorRepo) Upsert(ctx context.Context, vector model.EmbeddingVector, payload model.UpsertPayload) error {
	return nil
}

func (r *pgVectorRepo) Search(ctx context.Context, queryVector []float32, sparseVector []float32, accountID string, maxResults int) ([]model.SearchResult, error) {
	return []model.SearchResult{}, nil
}

func (r *pgVectorRepo) Delete(ctx context.Context, path string, accountID string) error {
	return nil
}

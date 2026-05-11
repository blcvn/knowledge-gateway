package qdrant

import (
	"context"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type vectorSearcher struct {
	// Qdrant client connection would go here
}

func NewVectorSearcher() port.VectorSearcher {
	return &vectorSearcher{}
}

func (s *vectorSearcher) SearchSimilar(ctx context.Context, embedding []float32, topK int, tenantID, datasetID string) ([]domain.SearchResult, error) {
	// Placeholder: Execute actual Qdrant vector search
	return []domain.SearchResult{}, nil
}

func (s *vectorSearcher) SearchChunks(ctx context.Context, query string, topK int, tenantID, datasetID string) ([]domain.SearchResult, error) {
	// Placeholder: Keyword/raw text search over Qdrant payload
	return []domain.SearchResult{}, nil
}

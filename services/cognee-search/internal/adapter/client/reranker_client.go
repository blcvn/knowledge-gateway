package client

import (
	"context"

	"vnp-memory/services/cognee-search/internal/domain"
	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type rerankerClient struct {
	modelID string
}

func NewRerankerClient(modelID string) port.Reranker {
	return &rerankerClient{modelID: modelID}
}

func (c *rerankerClient) Rerank(ctx context.Context, query string, results []domain.SearchResult, topK int) []domain.SearchResult {
	// In production, this calls a cross-encoder model to re-score the items.
	// For now, return as is.
	return results
}

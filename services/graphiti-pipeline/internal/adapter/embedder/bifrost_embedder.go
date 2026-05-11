package embedder

import (
	"context"

	"graphiti-pipeline/internal/domain/knowledge"
	"graphiti-pipeline/internal/usecase/port"
)

type BifrostEmbedder struct {
	// Configuration
}

func NewBifrostEmbedder() port.EmbedderClient {
	return &BifrostEmbedder{}
}

func (e *BifrostEmbedder) GenerateEmbedding(ctx context.Context, req knowledge.EmbeddingRequest) (knowledge.EmbeddingVector, error) {
	// Call Bifrost embedder API
	return nil, nil
}

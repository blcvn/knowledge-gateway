package client

import (
	"context"

	"vnp-memory/ov-search/internal/usecase/port"
)

type bifrostClient struct {
	addr string
}

func NewBifrostClient(addr string) port.EmbedderPort {
	return &bifrostClient{addr: addr}
}

func (c *bifrostClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, []float32, error) {
	// Mock call to Bifrost for dense and sparse embeddings
	dense := make([]float32, 1536)
	sparse := make([]float32, 1536)
	return dense, sparse, nil
}

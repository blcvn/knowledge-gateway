package client

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type embedderClient struct {
	provider string
}

// NewEmbedderClient creates a new embedding generation client.
func NewEmbedderClient(provider string) port.EmbedderClient {
	return &embedderClient{provider: provider}
}

func (c *embedderClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	// Implement HTTP call to OpenAI/Jina/Ollama
	return nil, nil
}

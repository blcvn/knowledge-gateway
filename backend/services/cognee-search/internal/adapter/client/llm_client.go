package client

import (
	"context"

	"vnp-memory/services/cognee-search/internal/usecase/port"
)

type llmClient struct {
	// Client connection to Bifrost/OpenAI
}

func NewLLMClient() port.LLMClient {
	return &llmClient{}
}

func (c *llmClient) Generate(ctx context.Context, prompt string) (string, error) {
	return "Mocked LLM generation based on context", nil
}

func (c *llmClient) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

package client

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type bifrostClient struct {
	url string
}

// NewBifrostClient creates a new LLM client using the Bifrost Gateway.
func NewBifrostClient(url string) port.LLMClient {
	return &bifrostClient{url: url}
}

func (c *bifrostClient) GenerateCompletion(ctx context.Context, prompt string, model string, maxTokens int) (string, error) {
	// Implement HTTP call to Bifrost
	return "", nil
}

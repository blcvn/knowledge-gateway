package embedder

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// EmbedderClient — generates vector embeddings for text
type EmbedderClient interface {
	Create(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}

// OpenAIEmbedder uses OpenAI's text-embedding-3-small (1536 dims)
type OpenAIEmbedder struct {
	client *openai.Client
	model  string
	dims   int
}

func NewOpenAIEmbedder(apiKey, model string) *OpenAIEmbedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	return &OpenAIEmbedder{
		client: openai.NewClient(apiKey),
		model:  model,
		dims:   1536,
	}
}

func (e *OpenAIEmbedder) Dimensions() int { return e.dims }

func (e *OpenAIEmbedder) Create(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return make([]float32, e.dims), nil
	}

	// Truncate to ~8192 tokens (API limit — rough char approximation)
	if len(text) > 32000 {
		text = text[:32000]
	}

	resp, err := e.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(e.model),
	})
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return resp.Data[0].Embedding, nil
}

package port

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/model"
)

// LLMClient is the driven port for LLM interactions (e.g., Bifrost).
type LLMClient interface {
	GenerateCompletion(ctx context.Context, prompt string, model string, maxTokens int) (string, error)
}

// EmbedderClient is the driven port for generating vector embeddings.
type EmbedderClient interface {
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// EventPublisher is the driven port for publishing integration events.
type EventPublisher interface {
	PublishEngineCompleted(ctx context.Context, result model.PipelineResult) error
	PublishProfileChanged(ctx context.Context, userID, projectID string) error
	PublishEventCreated(ctx context.Context, event model.UserEvent) error
}

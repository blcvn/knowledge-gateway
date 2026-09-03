package port

import (
	"context"
	"vnp-memory/services/graphiti-knowledge/domain"
)

type LLMClient interface {
	Complete(ctx context.Context, prompt string, model string) (string, domain.TokenUsage, error)
}

type EmbedderClient interface {
	Embed(ctx context.Context, text string, model string) (domain.EmbeddingVector, error)
	EmbedBatch(ctx context.Context, texts []string, model string) ([]domain.EmbeddingVector, error)
}

type GraphReader interface {
	FindSimilarEntities(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEntity, error)
	GetEntityByName(ctx context.Context, name string, groupID string) (*domain.ExtractedEntity, error)
	FindSimilarEdges(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEdge, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, data interface{}) error
}

type PromptRegistry interface {
	Render(templateID string, vars map[string]interface{}) (string, error)
	GetModel(templateID string) string
	List() []domain.PromptTemplate
}

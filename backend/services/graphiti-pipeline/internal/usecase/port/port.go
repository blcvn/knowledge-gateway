package port

import (
	"context"

	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/domain/knowledge"
)

type EventPublisher interface {
	PublishEpisodeIngested(ctx context.Context, event ingestion.EpisodeIngested) error
	PublishEpisodeFailed(ctx context.Context, event ingestion.EpisodeFailed) error
}

type StoreClient interface {
	SaveBulk(ctx context.Context, req SaveBulkRequest) error
	RollbackBulk(ctx context.Context, episodeID string) error
}

type SaveBulkRequest struct {
	Episode  ingestion.Episode
	Entities []knowledge.ExtractedEntity
	Edges    []knowledge.ExtractedEdge
}

type LLMClient interface {
	ExtractEntities(ctx context.Context, content string) ([]knowledge.ExtractedEntity, error)
	ResolveEntities(ctx context.Context, existing []knowledge.ExtractedEntity, newEntities []knowledge.ExtractedEntity) ([]knowledge.Resolution, error)
	ExtractEdges(ctx context.Context, content string, entities []knowledge.ExtractedEntity) ([]knowledge.ExtractedEdge, error)
	ResolveEdges(ctx context.Context, existing []knowledge.ExtractedEdge, newEdges []knowledge.ExtractedEdge) ([]knowledge.ExtractedEdge, error)
	UpdateCommunity(ctx context.Context, community knowledge.CommunityNode) (knowledge.CommunityNode, error)
}

type EmbedderClient interface {
	GenerateEmbedding(ctx context.Context, req knowledge.EmbeddingRequest) (knowledge.EmbeddingVector, error)
}

type EpisodeRepository interface {
	Save(ctx context.Context, episode ingestion.Episode) error
	CheckDuplicate(ctx context.Context, hash ingestion.ContentHash) (bool, error)
}

type SagaRepository interface {
	Save(ctx context.Context, saga ingestion.Saga) error
	UpdateState(ctx context.Context, sagaID string, state ingestion.SagaState, step ingestion.PipelineStep) error
}

type GroupLock interface {
	Acquire(ctx context.Context, groupID ingestion.GroupID) (func(), error)
}

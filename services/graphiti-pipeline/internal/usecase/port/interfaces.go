// Package port defines input/output port interfaces for graphiti-pipeline.
//
// Consolidated from: graphiti-ingestion + graphiti-knowledge
// Key optimization: Knowledge extraction RPCs called locally by saga orchestrator.
package port

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/graphiti-pipeline/internal/domain/ingestion"
	"github.com/vnp-community/vnp-memory/services/graphiti-pipeline/internal/domain/knowledge"
)

// --- Input Ports ---

// IngestionUseCase handles episode ingestion with saga orchestration.
type IngestionUseCase interface {
	IngestEpisode(ctx context.Context, tenantID uuid.UUID, groupID, content, speaker, source string, validAt time.Time) (*ingestion.Episode, error)
	GetEpisode(ctx context.Context, id uuid.UUID) (*ingestion.Episode, error)
	GetSagaState(ctx context.Context, episodeID uuid.UUID) (*ingestion.SagaState, error)
}

// KnowledgeUseCase handles entity/edge extraction and community detection.
type KnowledgeUseCase interface {
	ExtractEntities(ctx context.Context, tenantID uuid.UUID, content string, validAt time.Time) ([]knowledge.Entity, error)
	ExtractEdges(ctx context.Context, tenantID uuid.UUID, entities []knowledge.Entity, content string) ([]knowledge.Edge, error)
	DetectCommunities(ctx context.Context, tenantID uuid.UUID) ([]knowledge.Community, error)
	InvalidateEntity(ctx context.Context, entityID uuid.UUID, invalidAt time.Time) error
}

// --- Output Ports ---

// EpisodeRepository persists episodes and saga state.
type EpisodeRepository interface {
	CreateEpisode(ctx context.Context, ep *ingestion.Episode) error
	FindEpisodeByID(ctx context.Context, id uuid.UUID) (*ingestion.Episode, error)

	CreateSaga(ctx context.Context, saga *ingestion.SagaState) error
	FindSagaByEpisode(ctx context.Context, episodeID uuid.UUID) (*ingestion.SagaState, error)
	UpdateSagaStep(ctx context.Context, sagaID uuid.UUID, step, status string) error
}

// EntityRepository persists knowledge entities with bi-temporal support.
type EntityRepository interface {
	Upsert(ctx context.Context, entity *knowledge.Entity) error
	FindByID(ctx context.Context, id uuid.UUID) (*knowledge.Entity, error)
	FindByName(ctx context.Context, tenantID uuid.UUID, name string) (*knowledge.Entity, error)
	Invalidate(ctx context.Context, id uuid.UUID, invalidAt time.Time) error
}

// EdgeRepository persists knowledge edges.
type EdgeRepository interface {
	Upsert(ctx context.Context, edge *knowledge.Edge) error
	FindBySource(ctx context.Context, sourceID uuid.UUID) ([]knowledge.Edge, error)
	FindByTarget(ctx context.Context, targetID uuid.UUID) ([]knowledge.Edge, error)
}

// CommunityRepository persists detected communities.
type CommunityRepository interface {
	Upsert(ctx context.Context, community *knowledge.Community) error
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]knowledge.Community, error)
}

// LLMExtractor calls the LLM for entity/edge extraction.
type LLMExtractor interface {
	ExtractEntitiesFromText(ctx context.Context, text string) ([]knowledge.Entity, error)
	ExtractEdgesFromText(ctx context.Context, text string, entities []knowledge.Entity) ([]knowledge.Edge, error)
	SummarizeCommunity(ctx context.Context, entities []knowledge.Entity) (string, error)
}

// EventPublisher publishes pipeline events to NATS.
type EventPublisher interface {
	PublishEpisodeIngested(ctx context.Context, tenantID, episodeID uuid.UUID) error
	PublishPipelineCompleted(ctx context.Context, tenantID, episodeID uuid.UUID) error
}

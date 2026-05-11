package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
)

// JobRepository persists CognifyJob state for resume-on-failure.
type JobRepository interface {
	Create(ctx context.Context, job *domain.CognifyJob) error
	GetByID(ctx context.Context, tenantID string, id uuid.UUID) (*domain.CognifyJob, error)
	Update(ctx context.Context, job *domain.CognifyJob) error
	ListByDataset(ctx context.Context, tenantID string, datasetID uuid.UUID) ([]*domain.CognifyJob, error)
}

// DataItemReader reads ingested data items for cognification (calls cognee-ingestion or local DB).
type DataItemReader interface {
	// GetTextByDataset returns all raw text from a dataset's data items.
	GetTextByDataset(ctx context.Context, tenantID string, datasetID uuid.UUID) ([]TextItem, error)
}

// TextItem represents extracted text from an ingested data item.
type TextItem struct {
	ID   string
	Text string
}

// LLMClient abstracts the LLM provider (via Bifrost gateway).
type LLMClient interface {
	// Complete sends a prompt and returns the completion text.
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)

	// CompleteStructured sends a prompt and parses the response into the target struct.
	CompleteStructured(ctx context.Context, systemPrompt, userPrompt string, target any) error
}

// EmbedderClient generates vector embeddings.
type EmbedderClient interface {
	// Embed returns embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle returns embedding for a single text.
	EmbedSingle(ctx context.Context, text string) ([]float32, error)
}

// GraphRepository abstracts the graph database (Neo4j) for entity/edge persistence.
type GraphRepository interface {
	UpsertEntity(ctx context.Context, tenantID string, entity *domain.Entity) (string, error)
	UpsertRelationship(ctx context.Context, tenantID string, rel *domain.Relationship) error
	UpsertCommunity(ctx context.Context, tenantID string, community *domain.Community) error
	GetEntityByName(ctx context.Context, tenantID, name string) (*domain.Entity, error)
}

// VectorRepository abstracts the vector store (pgvector/Qdrant) for embedding persistence.
type VectorRepository interface {
	UpsertChunkEmbedding(ctx context.Context, tenantID string, chunkID uuid.UUID, text string, embedding []float32) error
	UpsertEntityEmbedding(ctx context.Context, tenantID string, entityID uuid.UUID, text string, embedding []float32) error
}

// EventPublisher publishes pipeline events to NATS.
type EventPublisher interface {
	PublishPipelineCompleted(ctx context.Context, event domain.PipelineCompletedEvent) error
	PublishStageAdvanced(ctx context.Context, event domain.StageAdvancedEvent) error
}

// OntologyRepository persists ontology definitions.
type OntologyRepository interface {
	Create(ctx context.Context, ont *domain.Ontology) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Ontology, error)
}

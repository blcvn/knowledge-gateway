package port

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-cognify/internal/domain"
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
	// [NEW] CE-006: generic publish for memify lifecycle events
	PublishPipelineEvent(ctx context.Context, subject string, payload map[string]any) error
}

// MemifyGraphRepository extends GraphRepository with Memify-specific read/write methods.
// Separate interface to keep DI explicit and avoid interface pollution.
type MemifyGraphRepository interface {
	// GetDatasetGraph loads all nodes and edges for a dataset (read-only, used by Memify step 1)
	GetDatasetGraph(ctx context.Context, datasetID uuid.UUID, tenantID string) ([]domain.GraphNode, []domain.GraphEdge, error)
	// UpsertGraphDiff adds only new nodes/edges from diff — no deletes (Memify step 4)
	UpsertGraphDiff(ctx context.Context, datasetID uuid.UUID, tenantID string, diff domain.GraphDiff) error
}

// MemifyVectorRepository handles vector upserts for newly derived triplets.
type MemifyVectorRepository interface {
	// UpsertTripletPoint inserts/updates a triplet embedding point in the vector store.
	UpsertTripletPoint(ctx context.Context, collection, pointID string, vec []float32, payload map[string]any) error
}

// PipelineRunRepository persists and queries async pipeline job status.
// Backed by cognee_pipeline_runs table (migration 0045).
type PipelineRunRepository interface {
	Save(ctx context.Context, run domain.PipelineRun) error
	GetByID(ctx context.Context, id string) (*domain.PipelineRun, error)
	SetStatus(ctx context.Context, id string, status string) error
	SetStatusWithError(ctx context.Context, id string, status string, errMsg string) error
	SetStatusWithResult(ctx context.Context, id string, status string, newNodes, newEdges int) error
}

// OntologyRepository persists ontology definitions.
type OntologyRepository interface {
	Create(ctx context.Context, ont *domain.Ontology) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Ontology, error)
}

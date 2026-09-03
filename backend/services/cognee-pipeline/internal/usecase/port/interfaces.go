// Package port defines input/output port interfaces for cognee-pipeline.
//
// Consolidated from: cognee-ingestion + cognee-cognify
// Key optimization: CognifyUseCase is called locally by IngestionUseCase
// instead of cross-service gRPC (saves ~15ms per pipeline run).
package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/domain/cognify"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/domain/ingestion"
)

// --- Input Ports ---

// IngestionUseCase handles dataset and data item management.
type IngestionUseCase interface {
	CreateDataset(ctx context.Context, tenantID uuid.UUID, name, desc string) (*ingestion.Dataset, error)
	AddDataItem(ctx context.Context, datasetID uuid.UUID, sourceType, sourceURI, mimeType string) (*ingestion.DataItem, error)
	GetDataset(ctx context.Context, id uuid.UUID) (*ingestion.Dataset, error)
	ListDatasets(ctx context.Context, tenantID uuid.UUID) ([]*ingestion.Dataset, error)
	DeleteDataset(ctx context.Context, id uuid.UUID) error
}

// CognifyUseCase handles the 7-stage cognification pipeline.
type CognifyUseCase interface {
	StartCognify(ctx context.Context, tenantID, datasetID uuid.UUID) (*cognify.CognifyJob, error)
	GetJobStatus(ctx context.Context, jobID uuid.UUID) (*cognify.CognifyJob, error)
	CancelJob(ctx context.Context, jobID uuid.UUID) error

	// Ontology management
	CreateOntology(ctx context.Context, tenantID uuid.UUID, name string, categories []cognify.OntologyCategory) (*cognify.Ontology, error)
	GetOntology(ctx context.Context, id uuid.UUID) (*cognify.Ontology, error)
}

// --- Output Ports ---

// DatasetRepository persists datasets and data items.
type DatasetRepository interface {
	CreateDataset(ctx context.Context, ds *ingestion.Dataset) error
	FindDatasetByID(ctx context.Context, id uuid.UUID) (*ingestion.Dataset, error)
	ListDatasetsByTenant(ctx context.Context, tenantID uuid.UUID) ([]*ingestion.Dataset, error)
	DeleteDataset(ctx context.Context, id uuid.UUID) error

	CreateDataItem(ctx context.Context, item *ingestion.DataItem) error
	CountItemsByDataset(ctx context.Context, datasetID uuid.UUID) (int, error)
}

// CognifyJobRepository persists cognify job state.
type CognifyJobRepository interface {
	Create(ctx context.Context, job *cognify.CognifyJob) error
	FindByID(ctx context.Context, id uuid.UUID) (*cognify.CognifyJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status cognify.JobStatus, stage cognify.PipelineStage, progress float64) error
}

// OntologyRepository persists ontology definitions.
type OntologyRepository interface {
	Create(ctx context.Context, ont *cognify.Ontology) error
	FindByID(ctx context.Context, id uuid.UUID) (*cognify.Ontology, error)
}

// GraphStore abstracts the graph database for entity/edge persistence.
type GraphStore interface {
	UpsertEntity(ctx context.Context, tenantID uuid.UUID, name, entityType, desc string) (string, error)
	UpsertEdge(ctx context.Context, tenantID uuid.UUID, sourceID, targetID, relation string, weight float64) error
}

// EmbeddingService generates vector embeddings via LLM.
type EmbeddingService interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// EventPublisher publishes pipeline events to NATS.
type EventPublisher interface {
	PublishIngested(ctx context.Context, tenantID, datasetID uuid.UUID) error
	PublishCognifyCompleted(ctx context.Context, tenantID, datasetID uuid.UUID) error
}

package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/dto"
)

// CognifyUseCase is the main input port for triggering cognification.
type CognifyUseCase interface {
	Execute(ctx context.Context, req dto.TriggerCognifyRequest) (*dto.CognifyJobResult, error)
}

// JobManager handles cognify job lifecycle queries.
type JobManager interface {
	GetJob(ctx context.Context, tenantID string, jobID uuid.UUID) (*domain.CognifyJob, error)
	ListJobs(ctx context.Context, tenantID string, datasetID uuid.UUID) ([]*domain.CognifyJob, error)
	CancelJob(ctx context.Context, tenantID string, jobID uuid.UUID) error
}

// OntologyManager handles ontology CRUD.
type OntologyManager interface {
	Create(ctx context.Context, tenantID, name string, categories []domain.OntologyCategory) (*domain.Ontology, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.Ontology, error)
}

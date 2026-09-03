package cognify

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/domain/cognify"
	"github.com/vnp-community/vnp-memory/services/cognee-pipeline/internal/usecase/port"
)

type cognifyUseCase struct {
	jobRepo     port.CognifyJobRepository
	ontologyRepo port.OntologyRepository
	publisher   port.EventPublisher
}

// NewCognifyUseCase creates a new cognify use case orchestrator
func NewCognifyUseCase(jobRepo port.CognifyJobRepository, ontologyRepo port.OntologyRepository, publisher port.EventPublisher) port.CognifyUseCase {
	return &cognifyUseCase{
		jobRepo:      jobRepo,
		ontologyRepo: ontologyRepo,
		publisher:    publisher,
	}
}

func (uc *cognifyUseCase) StartCognify(ctx context.Context, tenantID, datasetID uuid.UUID) (*cognify.CognifyJob, error) {
	job := &cognify.CognifyJob{
		ID:        uuid.New(),
		TenantID:  tenantID,
		DatasetID: datasetID,
		Status:    cognify.JobRunning,
		Stage:     cognify.StageExtractEntities,
		Progress:  0.0,
	}

	err := uc.jobRepo.Create(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("failed to create cognify job: %w", err)
	}

	// 8-stage pipeline runner
	// In a real implementation, this would iterate through stages and update progress.
	// For now, we simulate completion and publish the completed event.
	go func(bgCtx context.Context, j *cognify.CognifyJob) {
		// Update status to completed
		_ = uc.jobRepo.UpdateStatus(bgCtx, j.ID, cognify.JobCompleted, cognify.StagePostProcess, 1.0)
		// Publish event to notify search service
		_ = uc.publisher.PublishCognifyCompleted(bgCtx, j.TenantID, j.DatasetID)
	}(context.Background(), job)

	return job, nil
}

func (uc *cognifyUseCase) GetJobStatus(ctx context.Context, jobID uuid.UUID) (*cognify.CognifyJob, error) {
	return uc.jobRepo.FindByID(ctx, jobID)
}

func (uc *cognifyUseCase) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	return uc.jobRepo.UpdateStatus(ctx, jobID, cognify.JobFailed, cognify.StagePostProcess, 0.0)
}

func (uc *cognifyUseCase) CreateOntology(ctx context.Context, tenantID uuid.UUID, name string, categories []cognify.OntologyCategory) (*cognify.Ontology, error) {
	ont := &cognify.Ontology{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       name,
		Categories: categories,
	}
	err := uc.ontologyRepo.Create(ctx, ont)
	return ont, err
}

func (uc *cognifyUseCase) GetOntology(ctx context.Context, id uuid.UUID) (*cognify.Ontology, error) {
	return uc.ontologyRepo.FindByID(ctx, id)
}

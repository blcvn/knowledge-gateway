package postgres

import (
	"context"

	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type SagaRepo struct {
	// db *sql.DB
}

func NewSagaRepo() port.SagaRepository {
	return &SagaRepo{}
}

func (r *SagaRepo) Save(ctx context.Context, saga ingestion.Saga) error {
	// Save initial saga state
	return nil
}

func (r *SagaRepo) UpdateState(ctx context.Context, sagaID string, state ingestion.SagaState, step ingestion.PipelineStep) error {
	// Atomic state transition
	return nil
}

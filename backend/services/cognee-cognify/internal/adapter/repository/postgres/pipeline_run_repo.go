// Package postgres implements PipelineRunRepository backed by cognee_pipeline_runs table.
// TASK-CE-006: Memify UseCase — Pipeline Run tracking
// Migration: db/migrations/0045_cognee_pipeline_runs.up.sql
package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/cognee-cognify/internal/domain"
)

// PipelineRunRepo implements port.PipelineRunRepository using PostgreSQL.
type PipelineRunRepo struct {
	db *pgxpool.Pool
}

// NewPipelineRunRepo constructs a PipelineRunRepo.
func NewPipelineRunRepo(db *pgxpool.Pool) *PipelineRunRepo {
	return &PipelineRunRepo{db: db}
}

// Save inserts a new pipeline run record.
func (r *PipelineRunRepo) Save(ctx context.Context, run domain.PipelineRun) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cognee_pipeline_runs
		    (id, dataset_id, tenant_id, type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, run.ID, run.DatasetID, run.TenantID, run.Type, run.Status, time.Now())
	return err
}

// GetByID retrieves a pipeline run by its ID.
func (r *PipelineRunRepo) GetByID(ctx context.Context, id string) (*domain.PipelineRun, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, dataset_id, tenant_id, type, status, new_nodes, new_edges, error, created_at, updated_at
		FROM cognee_pipeline_runs
		WHERE id = $1
	`, id)

	var run domain.PipelineRun
	var errStr *string
	err := row.Scan(
		&run.ID, &run.DatasetID, &run.TenantID, &run.Type, &run.Status,
		&run.NewNodes, &run.NewEdges, &errStr, &run.CreatedAt, &run.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if errStr != nil {
		run.Error = *errStr
	}
	return &run, nil
}

// SetStatus updates the status of a pipeline run.
func (r *PipelineRunRepo) SetStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cognee_pipeline_runs SET status = $1, updated_at = NOW() WHERE id = $2`,
		status, id,
	)
	return err
}

// SetStatusWithError marks a pipeline run as FAILED with an error message.
func (r *PipelineRunRepo) SetStatusWithError(ctx context.Context, id, status, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE cognee_pipeline_runs SET status = $1, error = $2, updated_at = NOW() WHERE id = $3`,
		status, errMsg, id,
	)
	return err
}

// SetStatusWithResult marks a pipeline run as COMPLETED with node/edge counts.
func (r *PipelineRunRepo) SetStatusWithResult(ctx context.Context, id, status string, newNodes, newEdges int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE cognee_pipeline_runs
		SET status = $1, new_nodes = $2, new_edges = $3, updated_at = NOW()
		WHERE id = $4
	`, status, newNodes, newEdges, id)
	return err
}

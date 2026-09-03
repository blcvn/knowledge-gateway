package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/repository"
)

type bufferRepository struct {
	db *pgxpool.Pool
}

func NewBufferZoneRepository(db *pgxpool.Pool) repository.BufferZoneRepository {
	return &bufferRepository{db: db}
}

func (r *bufferRepository) Insert(ctx context.Context, buffer *model.BufferZone) error {
	query := `
		INSERT INTO buffer_zones (id, user_id, project_id, blob_id, blob_type, token_size, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query, buffer.ID, buffer.UserID, buffer.ProjectID, buffer.BlobID, buffer.BlobType, buffer.TokenSize, buffer.Status, buffer.CreatedAt, buffer.UpdatedAt)
	return err
}

func (r *bufferRepository) GetTotalIdleTokens(ctx context.Context, projectID, userID string, blobType model.BlobType) (int, error) {
	query := `
		SELECT COALESCE(SUM(token_size), 0) FROM buffer_zones
		WHERE project_id = $1 AND user_id = $2 AND blob_type = $3 AND status = 'idle'
	`
	var total int
	err := r.db.QueryRow(ctx, query, projectID, userID, blobType).Scan(&total)
	return total, err
}

func (r *bufferRepository) GetStatusAggregation(ctx context.Context, projectID, userID string) (*repository.BufferStatusAggregation, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'idle') AS idle_count,
			COUNT(*) FILTER (WHERE status = 'processing') AS processing_count,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed_count,
			COALESCE(SUM(token_size) FILTER (WHERE status = 'idle'), 0) AS total_tokens
		FROM buffer_zones
		WHERE project_id = $1 AND user_id = $2
	`
	var agg repository.BufferStatusAggregation
	err := r.db.QueryRow(ctx, query, projectID, userID).Scan(&agg.IdleCount, &agg.ProcessingCount, &agg.FailedCount, &agg.TotalTokens)
	if err != nil {
		return nil, err
	}
	return &agg, nil
}

func (r *bufferRepository) UpdateStatusForIdle(ctx context.Context, projectID, userID string, blobType model.BlobType, targetStatus model.BufferStatus) ([]string, error) {
	query := `
		UPDATE buffer_zones
		SET status = $1, updated_at = NOW()
		WHERE project_id = $2 AND user_id = $3 AND blob_type = $4 AND status = 'idle'
		RETURNING id
	`
	rows, err := r.db.Query(ctx, query, targetStatus, projectID, userID, blobType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *bufferRepository) GetIdleSince(ctx context.Context, since time.Time) ([]*model.BufferZone, error) {
	query := `
		SELECT id, user_id, project_id, blob_id, blob_type, token_size, status, created_at, updated_at
		FROM buffer_zones
		WHERE status = 'idle' AND created_at <= $1
	`
	rows, err := r.db.Query(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*model.BufferZone
	for rows.Next() {
		var b model.BufferZone
		if err := rows.Scan(&b.ID, &b.UserID, &b.ProjectID, &b.BlobID, &b.BlobType, &b.TokenSize, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, &b)
	}
	return results, rows.Err()
}

func (r *bufferRepository) UpdateStatus(ctx context.Context, projectID, bufferID string, status model.BufferStatus) error {
	query := `UPDATE buffer_zones SET status = $1, updated_at = NOW() WHERE id = $2 AND project_id = $3`
	_, err := r.db.Exec(ctx, query, status, bufferID, projectID)
	return err
}

func (r *bufferRepository) DeleteByBlobID(ctx context.Context, projectID, blobID string) error {
	query := `DELETE FROM buffer_zones WHERE blob_id = $1 AND project_id = $2`
	_, err := r.db.Exec(ctx, query, blobID, projectID)
	return err
}

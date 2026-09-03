package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
)

type BufferRepo struct {
	db *pgxpool.Pool
}

func NewBufferRepo(db *pgxpool.Pool) *BufferRepo {
	return &BufferRepo{db: db}
}

func (r *BufferRepo) FindOrCreate(ctx context.Context, tenantID, userID uuid.UUID) (*ingestion.BufferZone, error) {
	q := `SELECT id, state, token_count, threshold, blob_ids, last_flushed 
	      FROM buffer_zones WHERE tenant_id = $1 AND user_id = $2`
	
	var buf ingestion.BufferZone
	buf.TenantID = tenantID
	buf.UserID = userID

	err := r.db.QueryRow(ctx, q, tenantID, userID).Scan(
		&buf.ID, &buf.State, &buf.TokenCount, &buf.Threshold, &buf.BlobIDs, &buf.LastFlushed,
	)

	if err == pgx.ErrNoRows {
		buf.ID = uuid.New()
		buf.State = ingestion.BufferIdle
		buf.Threshold = ingestion.DefaultTokenThreshold
		buf.BlobIDs = []uuid.UUID{}
		
		insertQ := `INSERT INTO buffer_zones (id, tenant_id, user_id, state, token_count, threshold, blob_ids)
		            VALUES ($1, $2, $3, $4, $5, $6, $7)`
		_, err := r.db.Exec(ctx, insertQ, buf.ID, buf.TenantID, buf.UserID, buf.State, buf.TokenCount, buf.Threshold, buf.BlobIDs)
		if err != nil {
			return nil, fmt.Errorf("insert buffer: %w", err)
		}
		return &buf, nil
	} else if err != nil {
		return nil, fmt.Errorf("query buffer: %w", err)
	}

	if buf.BlobIDs == nil {
		buf.BlobIDs = []uuid.UUID{}
	}
	return &buf, nil
}

func (r *BufferRepo) Update(ctx context.Context, buf *ingestion.BufferZone) error {
	q := `UPDATE buffer_zones SET state = $1, token_count = $2, threshold = $3, blob_ids = $4, last_flushed = $5, updated_at = $6
	      WHERE id = $7 AND tenant_id = $8`
	_, err := r.db.Exec(ctx, q, buf.State, buf.TokenCount, buf.Threshold, buf.BlobIDs, buf.LastFlushed, time.Now(), buf.ID, buf.TenantID)
	if err != nil {
		return fmt.Errorf("update buffer: %w", err)
	}
	return nil
}

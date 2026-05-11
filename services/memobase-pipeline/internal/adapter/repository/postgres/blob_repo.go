package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
)

type BlobRepo struct {
	db *pgxpool.Pool
}

func NewBlobRepo(db *pgxpool.Pool) *BlobRepo {
	return &BlobRepo{db: db}
}

func (r *BlobRepo) Create(ctx context.Context, blob *ingestion.Blob) error {
	q := `INSERT INTO blobs (id, tenant_id, user_id, content, type, tokens, metadata, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.Exec(ctx, q, blob.ID, blob.TenantID, blob.UserID, blob.Content, blob.Type, blob.Tokens, blob.Metadata, blob.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert blob: %w", err)
	}
	return nil
}

func (r *BlobRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]ingestion.Blob, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := `SELECT id, tenant_id, user_id, content, type, tokens, metadata, created_at
	      FROM blobs WHERE id = ANY($1)`
	
	rows, err := r.db.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("query blobs: %w", err)
	}
	defer rows.Close()

	var blobs []ingestion.Blob
	for rows.Next() {
		var b ingestion.Blob
		if err := rows.Scan(&b.ID, &b.TenantID, &b.UserID, &b.Content, &b.Type, &b.Tokens, &b.Metadata, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan blob: %w", err)
		}
		blobs = append(blobs, b)
	}
	return blobs, nil
}

func (r *BlobRepo) DeleteByIDs(ctx context.Context, ids []uuid.UUID) error {
	q := `DELETE FROM blobs WHERE id = ANY($1)`
	_, err := r.db.Exec(ctx, q, ids)
	if err != nil {
		return fmt.Errorf("delete blobs: %w", err)
	}
	return nil
}

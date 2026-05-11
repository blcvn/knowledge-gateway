package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/repository"
)

type blobRepository struct {
	db *pgxpool.Pool
}

func NewBlobRepository(db *pgxpool.Pool) repository.BlobRepository {
	return &blobRepository{db: db}
}

func (r *blobRepository) Insert(ctx context.Context, blob *model.GeneralBlob) error {
	query := `
		INSERT INTO general_blobs (id, user_id, project_id, blob_type, blob_data, add_fields, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query, blob.ID, blob.UserID, blob.ProjectID, blob.BlobType, blob.BlobData, blob.AddFields, blob.CreatedAt, blob.UpdatedAt)
	return err
}

func (r *blobRepository) FindByID(ctx context.Context, projectID, blobID string) (*model.GeneralBlob, error) {
	query := `
		SELECT id, user_id, project_id, blob_type, blob_data, add_fields, created_at, updated_at
		FROM general_blobs
		WHERE id = $1 AND project_id = $2
	`
	row := r.db.QueryRow(ctx, query, blobID, projectID)
	
	var b model.GeneralBlob
	err := row.Scan(&b.ID, &b.UserID, &b.ProjectID, &b.BlobType, &b.BlobData, &b.AddFields, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *blobRepository) Delete(ctx context.Context, projectID, blobID string) error {
	query := `DELETE FROM general_blobs WHERE id = $1 AND project_id = $2`
	_, err := r.db.Exec(ctx, query, blobID, projectID)
	return err
}

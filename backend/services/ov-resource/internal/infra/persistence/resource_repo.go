package persistence

import (
	"context"
	"database/sql"
	"time"

	"openviking.com/ov-resource/internal/domain/model"
)

type resourceRepo struct {
	db *sql.DB
}

func NewResourceRepository(db *sql.DB) *resourceRepo {
	return &resourceRepo{db: db}
}

func (r *resourceRepo) Create(ctx context.Context, resource *model.Resource) error {
	query := `
		INSERT INTO ov_resources (
			id, account_id, source_path, target_path, filename, mime_type, parser_type, 
			chunk_count, total_tokens, content_hash, status, error_message, parse_duration_ms, 
			ingested_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`
	_, err := r.db.ExecContext(ctx, query,
		resource.ID, resource.AccountID, resource.SourcePath, resource.TargetPath, resource.Filename,
		resource.MimeType, resource.ParserType, resource.ChunkCount, resource.TotalTokens, resource.ContentHash,
		resource.Status, resource.ErrorMessage, resource.ParseDurationMs, resource.IngestedAt, time.Now(),
	)
	return err
}

func (r *resourceRepo) Update(ctx context.Context, resource *model.Resource) error {
	query := `
		UPDATE ov_resources SET
			status = $1, chunk_count = $2, total_tokens = $3, parse_duration_ms = $4,
			ingested_at = $5, error_message = $6
		WHERE id = $7 AND account_id = $8
	`
	_, err := r.db.ExecContext(ctx, query,
		resource.Status, resource.ChunkCount, resource.TotalTokens, resource.ParseDurationMs,
		resource.IngestedAt, resource.ErrorMessage, resource.ID, resource.AccountID,
	)
	return err
}

func (r *resourceRepo) GetByID(ctx context.Context, id, accountID string) (*model.Resource, error) {
	query := `SELECT id, account_id, source_path, target_path, filename, mime_type, parser_type, 
		chunk_count, total_tokens, content_hash, status, error_message, parse_duration_ms, 
		ingested_at, created_at FROM ov_resources WHERE id = $1 AND account_id = $2`
	
	res := &model.Resource{}
	var ingestedAt sql.NullTime
	var errMsg sql.NullString
	err := r.db.QueryRowContext(ctx, query, id, accountID).Scan(
		&res.ID, &res.AccountID, &res.SourcePath, &res.TargetPath, &res.Filename,
		&res.MimeType, &res.ParserType, &res.ChunkCount, &res.TotalTokens, &res.ContentHash,
		&res.Status, &errMsg, &res.ParseDurationMs, &ingestedAt, &res.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if ingestedAt.Valid {
		res.IngestedAt = ingestedAt.Time
	}
	if errMsg.Valid {
		res.ErrorMessage = errMsg.String
	}
	return res, nil
}

func (r *resourceRepo) GetByHash(ctx context.Context, accountID, hash string) (*model.Resource, error) {
	query := `SELECT id, account_id, source_path, target_path, filename, mime_type, parser_type, 
		chunk_count, total_tokens, content_hash, status, error_message, parse_duration_ms, 
		ingested_at, created_at FROM ov_resources WHERE account_id = $1 AND content_hash = $2`
	
	res := &model.Resource{}
	var ingestedAt sql.NullTime
	var errMsg sql.NullString
	err := r.db.QueryRowContext(ctx, query, accountID, hash).Scan(
		&res.ID, &res.AccountID, &res.SourcePath, &res.TargetPath, &res.Filename,
		&res.MimeType, &res.ParserType, &res.ChunkCount, &res.TotalTokens, &res.ContentHash,
		&res.Status, &errMsg, &res.ParseDurationMs, &ingestedAt, &res.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if ingestedAt.Valid {
		res.IngestedAt = ingestedAt.Time
	}
	if errMsg.Valid {
		res.ErrorMessage = errMsg.String
	}
	return res, nil
}

func (r *resourceRepo) UpdateStatus(ctx context.Context, id, accountID string, status model.ResourceStatus, errorMessage string) error {
	query := `UPDATE ov_resources SET status = $1, error_message = $2 WHERE id = $3 AND account_id = $4`
	_, err := r.db.ExecContext(ctx, query, status, errorMessage, id, accountID)
	return err
}

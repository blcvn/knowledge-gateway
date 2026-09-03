package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/cognee-ingestion/internal/domain"
)

// DataItemRepo implements port.DataItemRepository using PostgreSQL.
type DataItemRepo struct {
	pool *pgxpool.Pool
}

// NewDataItemRepo creates a new PostgreSQL-backed DataItemRepository.
func NewDataItemRepo(pool *pgxpool.Pool) *DataItemRepo {
	return &DataItemRepo{pool: pool}
}

// Create persists a new data item.
func (r *DataItemRepo) Create(ctx context.Context, item *domain.DataItem) error {
	meta, err := json.Marshal(item.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO data_items (id, dataset_id, tenant_id, source, filename, mime_type, raw_text, storage_path, size_bytes, file_hash, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		item.ID, item.DatasetID, item.TenantID, string(item.Source),
		item.Filename, string(item.MimeType), item.RawText, item.StoragePath,
		item.SizeBytes, item.FileHash, meta, item.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert data item: %w", err)
	}
	return nil
}

// ListByDataset returns all data items for a given dataset.
func (r *DataItemRepo) ListByDataset(ctx context.Context, datasetID uuid.UUID) ([]*domain.DataItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, dataset_id, tenant_id, source, filename, mime_type, raw_text, storage_path, size_bytes, file_hash, metadata, created_at
		FROM data_items
		WHERE dataset_id = $1
		ORDER BY created_at DESC`,
		datasetID,
	)
	if err != nil {
		return nil, fmt.Errorf("query data items: %w", err)
	}
	defer rows.Close()

	return scanItems(rows)
}

// DeleteByDataset removes all data items for a dataset.
func (r *DataItemRepo) DeleteByDataset(ctx context.Context, datasetID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM data_items WHERE dataset_id = $1`, datasetID)
	if err != nil {
		return fmt.Errorf("delete data items: %w", err)
	}
	return nil
}

// ExistsByHash checks if a data item with the given hash exists in the dataset.
func (r *DataItemRepo) ExistsByHash(ctx context.Context, datasetID uuid.UUID, fileHash string) (bool, error) {
	if fileHash == "" {
		return false, nil
	}
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM data_items WHERE dataset_id = $1 AND file_hash = $2)`,
		datasetID, fileHash,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check hash exists: %w", err)
	}
	return exists, nil
}

// GetByID retrieves a single data item by ID.
func (r *DataItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DataItem, error) {
	item := &domain.DataItem{}
	var source, mimeType string
	var meta []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, dataset_id, tenant_id, source, filename, mime_type, raw_text, storage_path, size_bytes, file_hash, metadata, created_at
		FROM data_items
		WHERE id = $1`,
		id,
	).Scan(
		&item.ID, &item.DatasetID, &item.TenantID, &source,
		&item.Filename, &mimeType, &item.RawText, &item.StoragePath,
		&item.SizeBytes, &item.FileHash, &meta, &item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query data item: %w", err)
	}

	item.Source = domain.DataSource(source)
	item.MimeType = domain.MimeType(mimeType)
	if meta != nil {
		_ = json.Unmarshal(meta, &item.Metadata)
	}
	return item, nil
}

func scanItems(rows pgx.Rows) ([]*domain.DataItem, error) {
	var items []*domain.DataItem
	for rows.Next() {
		item := &domain.DataItem{}
		var source, mimeType string
		var meta []byte
		if err := rows.Scan(
			&item.ID, &item.DatasetID, &item.TenantID, &source,
			&item.Filename, &mimeType, &item.RawText, &item.StoragePath,
			&item.SizeBytes, &item.FileHash, &meta, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan data item: %w", err)
		}
		item.Source = domain.DataSource(source)
		item.MimeType = domain.MimeType(mimeType)
		if meta != nil {
			_ = json.Unmarshal(meta, &item.Metadata)
		}
		items = append(items, item)
	}
	return items, nil
}

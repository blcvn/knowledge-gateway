// Package postgres implements DatasetRepository and DataItemRepository
// using PostgreSQL with pgx driver.
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

// DatasetRepo implements port.DatasetRepository using PostgreSQL.
type DatasetRepo struct {
	pool *pgxpool.Pool
}

// NewDatasetRepo creates a new PostgreSQL-backed DatasetRepository.
func NewDatasetRepo(pool *pgxpool.Pool) *DatasetRepo {
	return &DatasetRepo{pool: pool}
}

// Create persists a new dataset.
func (r *DatasetRepo) Create(ctx context.Context, ds *domain.Dataset) error {
	meta, err := json.Marshal(ds.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO datasets (id, tenant_id, name, description, status, file_count, total_size_bytes, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		ds.ID, ds.TenantID, ds.Name, ds.Description, string(ds.Status),
		ds.FileCount, ds.TotalSizeBytes, meta, ds.CreatedAt, ds.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert dataset: %w", err)
	}
	return nil
}

// GetByID retrieves a dataset by ID, scoped to the given tenant.
func (r *DatasetRepo) GetByID(ctx context.Context, tenantID string, id uuid.UUID) (*domain.Dataset, error) {
	ds := &domain.Dataset{}
	var status string
	var meta []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, description, status, file_count, total_size_bytes, metadata, created_at, updated_at
		FROM datasets
		WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(
		&ds.ID, &ds.TenantID, &ds.Name, &ds.Description, &status,
		&ds.FileCount, &ds.TotalSizeBytes, &meta, &ds.CreatedAt, &ds.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query dataset: %w", err)
	}

	ds.Status = domain.DatasetStatus(status)
	if meta != nil {
		_ = json.Unmarshal(meta, &ds.Metadata)
	}
	return ds, nil
}

// List returns datasets for a tenant with cursor-based pagination.
func (r *DatasetRepo) List(ctx context.Context, tenantID string, cursor string, limit int) ([]*domain.Dataset, string, error) {
	var rows pgx.Rows
	var err error

	if cursor == "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, name, description, status, file_count, total_size_bytes, metadata, created_at, updated_at
			FROM datasets
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT $2`,
			tenantID, limit+1,
		)
	} else {
		cursorID, parseErr := uuid.Parse(cursor)
		if parseErr != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", parseErr)
		}
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, name, description, status, file_count, total_size_bytes, metadata, created_at, updated_at
			FROM datasets
			WHERE tenant_id = $1 AND id < $2
			ORDER BY created_at DESC
			LIMIT $3`,
			tenantID, cursorID, limit+1,
		)
	}
	if err != nil {
		return nil, "", fmt.Errorf("query datasets: %w", err)
	}
	defer rows.Close()

	var datasets []*domain.Dataset
	for rows.Next() {
		ds := &domain.Dataset{}
		var status string
		var meta []byte
		if err := rows.Scan(
			&ds.ID, &ds.TenantID, &ds.Name, &ds.Description, &status,
			&ds.FileCount, &ds.TotalSizeBytes, &meta, &ds.CreatedAt, &ds.UpdatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan dataset: %w", err)
		}
		ds.Status = domain.DatasetStatus(status)
		if meta != nil {
			_ = json.Unmarshal(meta, &ds.Metadata)
		}
		datasets = append(datasets, ds)
	}

	var nextCursor string
	if len(datasets) > limit {
		nextCursor = datasets[limit].ID.String()
		datasets = datasets[:limit]
	}
	return datasets, nextCursor, nil
}

// Delete removes a dataset by ID (cascade deletes data_items via FK).
func (r *DatasetRepo) Delete(ctx context.Context, tenantID string, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM datasets WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete dataset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrDatasetNotFound
	}
	return nil
}

// UpdateStatus transitions dataset status.
func (r *DatasetRepo) UpdateStatus(ctx context.Context, tenantID string, id uuid.UUID, status domain.DatasetStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE datasets SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3`,
		string(status), id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// IncrementItems atomically increments file count and total size.
func (r *DatasetRepo) IncrementItems(ctx context.Context, tenantID string, id uuid.UUID, sizeBytes int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE datasets
		SET file_count = file_count + 1, total_size_bytes = total_size_bytes + $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3`,
		sizeBytes, id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("increment items: %w", err)
	}
	return nil
}

// ExistsByName checks if a dataset with the given name exists for the tenant.
func (r *DatasetRepo) ExistsByName(ctx context.Context, tenantID, name string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM datasets WHERE tenant_id = $1 AND name = $2)`,
		tenantID, name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check name exists: %w", err)
	}
	return exists, nil
}

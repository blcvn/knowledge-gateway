// Package pgvector — DatasetRepo for Cognee metadata tracking.
package pgvector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	cogdomain "vnp-memory/services/kg-service/internal/domain/cognee"
)

// DatasetRepo implements port.DatasetRepository.
type DatasetRepo struct {
	pool *pgxpool.Pool
}

// NewDatasetRepo creates a DatasetRepo.
func NewDatasetRepo(pool *pgxpool.Pool) *DatasetRepo {
	return &DatasetRepo{pool: pool}
}

func (r *DatasetRepo) Save(ctx context.Context, ds *cogdomain.Dataset) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO cognee_datasets (id, tenant_id, name, status, data_count, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET name=$3, status=$4, updated_at=$7`,
		ds.ID, ds.TenantID, ds.Name, ds.Status, ds.DataCount, ds.CreatedAt, ds.UpdatedAt,
	)
	return err
}

func (r *DatasetRepo) FindByID(ctx context.Context, id string) (*cogdomain.Dataset, error) {
	ds := &cogdomain.Dataset{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, status, data_count, created_at, updated_at
		 FROM cognee_datasets WHERE id = $1`, id,
	).Scan(&ds.ID, &ds.TenantID, &ds.Name, &ds.Status, &ds.DataCount, &ds.CreatedAt, &ds.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("dataset not found: %s", id)
	}
	return ds, err
}

func (r *DatasetRepo) ListByTenant(ctx context.Context, tenantID string) ([]*cogdomain.Dataset, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, status, data_count, created_at, updated_at
		 FROM cognee_datasets WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var datasets []*cogdomain.Dataset
	for rows.Next() {
		ds := &cogdomain.Dataset{}
		if err := rows.Scan(&ds.ID, &ds.TenantID, &ds.Name, &ds.Status,
			&ds.DataCount, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		datasets = append(datasets, ds)
	}
	return datasets, nil
}

func (r *DatasetRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE cognee_datasets SET status=$1, updated_at=$2 WHERE id=$3`,
		status, time.Now(), id,
	)
	return err
}

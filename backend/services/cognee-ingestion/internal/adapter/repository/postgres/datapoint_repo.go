// Package postgres implements DataPointRepo backed by cognee_datapoints table.
// TASK-CE-007: DataPoint Schema (SOL-003)
// Migration: db/migrations/0046_cognee_datapoints.up.sql
package postgres

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"vnp-memory/services/cognee-ingestion/internal/domain"
)

// DataPointRepo implements DataPointRepository using PostgreSQL (cognee_datapoints table).
type DataPointRepo struct {
	db *pgxpool.Pool
}

// NewDataPointRepo constructs a DataPointRepo.
func NewDataPointRepo(db *pgxpool.Pool) *DataPointRepo {
	return &DataPointRepo{db: db}
}

// GetByID retrieves a DataPoint by its UUID.
func (r *DataPointRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DataPoint, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, dataset_id, tenant_id, type, fields, index_fields, node_sets, created_at, updated_at
		FROM cognee_datapoints WHERE id = $1
	`, id)

	var dp domain.DataPoint
	var fieldsJSON []byte
	err := row.Scan(
		&dp.ID, &dp.Version, &dp.DatasetID, &dp.TenantID, &dp.Type,
		&fieldsJSON, &dp.IndexFields, &dp.NodeSets, &dp.CreatedAt, &dp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(fieldsJSON, &dp.Fields); err != nil {
		dp.Fields = make(map[string]any)
	}
	return &dp, nil
}

// Upsert inserts or updates a DataPoint in the database.
// Uses ON CONFLICT to increment version on re-ingestion (idempotent).
func (r *DataPointRepo) Upsert(ctx context.Context, dp domain.DataPoint) error {
	fieldsJSON, err := json.Marshal(dp.Fields)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO cognee_datapoints
		    (id, version, dataset_id, tenant_id, type, fields, index_fields, node_sets, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE
		SET version      = EXCLUDED.version,
		    fields       = EXCLUDED.fields,
		    index_fields = EXCLUDED.index_fields,
		    node_sets    = EXCLUDED.node_sets,
		    updated_at   = NOW()
	`,
		dp.ID, dp.Version, dp.DatasetID, dp.TenantID, dp.Type,
		fieldsJSON, dp.IndexFields, dp.NodeSets,
	)
	return err
}

// ListByDataset returns DataPoints for a dataset, paginated.
func (r *DataPointRepo) ListByDataset(ctx context.Context, datasetID uuid.UUID, tenantID string, limit, offset int) ([]domain.DataPoint, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, version, type, fields, index_fields, node_sets, created_at, updated_at
		FROM cognee_datapoints
		WHERE dataset_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, datasetID, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DataPoint
	for rows.Next() {
		var dp domain.DataPoint
		var fieldsJSON []byte
		if err := rows.Scan(&dp.ID, &dp.Version, &dp.Type, &fieldsJSON, &dp.IndexFields, &dp.NodeSets, &dp.CreatedAt, &dp.UpdatedAt); err != nil {
			continue
		}
		json.Unmarshal(fieldsJSON, &dp.Fields) // nolint:errcheck
		result = append(result, dp)
	}
	return result, nil
}

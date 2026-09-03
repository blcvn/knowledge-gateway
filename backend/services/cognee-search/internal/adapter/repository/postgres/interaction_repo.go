// Package postgres implements InteractionRepo backed by cognee_interactions table.
// TASK-CE-009: Feedback Loop — Interaction Logging (SOL-005 §2.6)
// Migration: db/migrations/0047_cognee_interactions.up.sql
package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"vnp-memory/services/cognee-search/internal/domain"
)

// InteractionRepo implements InteractionRepository using PostgreSQL.
type InteractionRepo struct {
	db *pgxpool.Pool
}

// NewInteractionRepo constructs an InteractionRepo.
func NewInteractionRepo(db *pgxpool.Pool) *InteractionRepo {
	return &InteractionRepo{db: db}
}

// Save inserts a new Interaction record (logged when save_interaction=true).
func (r *InteractionRepo) Save(ctx context.Context, i domain.Interaction) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cognee_interactions
		    (id, tenant_id, session_id, dataset_id, query, strategy,
		     result_ids, result_scores, node_sets, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`,
		i.ID, i.TenantID, i.SessionID, i.DatasetID, i.Query, i.Strategy,
		pq.Array(i.ResultIDs), pq.Array(i.ResultScores), pq.Array(i.NodeSets), i.Timestamp,
	)
	return err
}

// GetByID retrieves an Interaction by its UUID (used by feedback handler).
func (r *InteractionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Interaction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, session_id, dataset_id, query, strategy,
		       result_ids, node_sets, timestamp
		FROM cognee_interactions WHERE id = $1
	`, id)

	var i domain.Interaction
	err := row.Scan(
		&i.ID, &i.TenantID, &i.SessionID, &i.DatasetID, &i.Query, &i.Strategy,
		pq.Array(&i.ResultIDs), pq.Array(&i.NodeSets), &i.Timestamp,
	)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

// List returns Interactions for a tenant, optionally filtered by session_id.
func (r *InteractionRepo) List(ctx context.Context, tenantID, sessionID string, limit, offset int) ([]domain.Interaction, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, query, strategy, result_ids, node_sets, timestamp
		FROM cognee_interactions
		WHERE tenant_id = $1
		  AND ($2 = '' OR session_id = $2)
		ORDER BY timestamp DESC
		LIMIT $3 OFFSET $4
	`, tenantID, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Interaction
	for rows.Next() {
		var i domain.Interaction
		if err := rows.Scan(&i.ID, &i.Query, &i.Strategy,
			pq.Array(&i.ResultIDs), pq.Array(&i.NodeSets), &i.Timestamp); err != nil {
			continue
		}
		result = append(result, i)
	}
	return result, nil
}

// Package postgres implements FeedbackRepo backed by cognee_feedback_records table.
// TASK-CE-009: Feedback Loop — Feedback Recording (SOL-005 §2.6)
// Migration: db/migrations/0047_cognee_interactions.up.sql
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"vnp-memory/services/cognee-search/internal/domain"
)

// FeedbackRepo implements FeedbackRepository using PostgreSQL.
type FeedbackRepo struct {
	db *pgxpool.Pool
}

// NewFeedbackRepo constructs a FeedbackRepo.
func NewFeedbackRepo(db *pgxpool.Pool) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

// Save inserts a FeedbackRecord.
// Score constraint [-1.0, 1.0] is enforced at DB level via CHECK constraint.
func (r *FeedbackRepo) Save(ctx context.Context, f domain.FeedbackRecord) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cognee_feedback_records
		    (id, interaction_id, tenant_id, score, text, affected_nodes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		f.ID, f.InteractionID, f.TenantID, f.Score, f.Text,
		pq.Array(f.AffectedNodes), f.CreatedAt,
	)
	return err
}

// ListByInteraction returns all feedback records for a specific interaction.
func (r *FeedbackRepo) ListByInteraction(ctx context.Context, interactionID string) ([]domain.FeedbackRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, interaction_id, tenant_id, score, text, affected_nodes, created_at
		FROM cognee_feedback_records
		WHERE interaction_id = $1
		ORDER BY created_at DESC
	`, interactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.FeedbackRecord
	for rows.Next() {
		var f domain.FeedbackRecord
		if err := rows.Scan(&f.ID, &f.InteractionID, &f.TenantID, &f.Score, &f.Text,
			pq.Array(&f.AffectedNodes), &f.CreatedAt); err != nil {
			continue
		}
		result = append(result, f)
	}
	return result, nil
}

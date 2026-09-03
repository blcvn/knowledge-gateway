package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
)

type GistRepo struct {
	db *pgxpool.Pool
}

func NewGistRepo(db *pgxpool.Pool) *GistRepo {
	return &GistRepo{db: db}
}

func (r *GistRepo) Create(ctx context.Context, gist *engine.EventGist) error {
	q := `INSERT INTO event_gists (id, tenant_id, user_id, summary, key_facts, created_at)
	      VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err := r.db.Exec(ctx, q, gist.ID, gist.TenantID, gist.UserID, gist.Summary, gist.KeyFacts, gist.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert gist: %w", err)
	}
	return nil
}

func (r *GistRepo) FindByUser(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]engine.EventGist, error) {
	q := `SELECT id, summary, key_facts, created_at FROM event_gists 
	      WHERE tenant_id = $1 AND user_id = $2 
	      ORDER BY created_at DESC LIMIT $3`
	
	rows, err := r.db.Query(ctx, q, tenantID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query gists: %w", err)
	}
	defer rows.Close()

	var gists []engine.EventGist
	for rows.Next() {
		var g engine.EventGist
		g.TenantID = tenantID
		g.UserID = userID
		if err := rows.Scan(&g.ID, &g.Summary, &g.KeyFacts, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan gist: %w", err)
		}
		gists = append(gists, g)
	}
	return gists, nil
}

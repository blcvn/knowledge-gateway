// Package pgvector implements the EpisodeRepository using PostgreSQL + pgvector.
//
// Episodes are stored with vector embeddings for semantic search.
// (MERGE-P2-T1)
package pgvector

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	pgv "github.com/pgvector/pgvector-go"

	"vnp-memory/services/kg-service/internal/domain/graphiti"
)

// EpisodeRepo implements port.EpisodeRepository using pgvector.
type EpisodeRepo struct {
	pool *pgxpool.Pool
}

// NewEpisodeRepo creates an EpisodeRepo.
func NewEpisodeRepo(pool *pgxpool.Pool) *EpisodeRepo {
	return &EpisodeRepo{pool: pool}
}

// Create persists a new episode.
func (r *EpisodeRepo) Create(ctx context.Context, ep *graphiti.Episode) error {
	var emb *pgv.Vector
	if len(ep.Embedding) > 0 {
		v := pgv.NewVector(ep.Embedding)
		emb = &v
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO kg_episodes (uuid, tenant_id, name, content, source, source_id, embedding, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (uuid) DO NOTHING`,
		ep.UUID, ep.TenantID, ep.Name, ep.Content, ep.Source, ep.SourceID, emb, ep.CreatedAt,
	)
	return err
}

// FindByUUID retrieves an episode by UUID.
func (r *EpisodeRepo) FindByUUID(ctx context.Context, tenantID, uuid string) (*graphiti.Episode, error) {
	ep := &graphiti.Episode{}
	err := r.pool.QueryRow(ctx,
		`SELECT uuid, tenant_id, name, content, source, source_id, created_at
		 FROM kg_episodes WHERE uuid = $1 AND tenant_id = $2`, uuid, tenantID,
	).Scan(&ep.UUID, &ep.TenantID, &ep.Name, &ep.Content, &ep.Source, &ep.SourceID, &ep.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("episode not found: %s", uuid)
	}
	return ep, nil
}

// SemanticSearch finds episodes using vector cosine similarity.
func (r *EpisodeRepo) SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*graphiti.Episode, error) {
	v := pgv.NewVector(embedding)
	rows, err := r.pool.Query(ctx,
		`SELECT uuid, tenant_id, name, content, source, source_id, created_at,
		        1 - (embedding <=> $1) AS score
		 FROM kg_episodes
		 WHERE tenant_id = $2 AND embedding IS NOT NULL
		 ORDER BY embedding <=> $1
		 LIMIT $3`,
		v, tenantID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}
	defer rows.Close()
	return scanEpisodes(rows)
}

// TextSearch finds episodes by full-text content match.
func (r *EpisodeRepo) TextSearch(ctx context.Context, tenantID, query string, limit int) ([]*graphiti.Episode, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT uuid, tenant_id, name, content, source, source_id, created_at
		 FROM kg_episodes
		 WHERE tenant_id = $1 AND (content ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC
		 LIMIT $3`,
		tenantID, query, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("text search: %w", err)
	}
	defer rows.Close()
	return scanEpisodes(rows)
}

func scanEpisodes(rows interface{ Next() bool; Scan(...any) error }) ([]*graphiti.Episode, error) {
	var episodes []*graphiti.Episode
	for rows.Next() {
		ep := &graphiti.Episode{}
		if err := rows.Scan(&ep.UUID, &ep.TenantID, &ep.Name, &ep.Content,
			&ep.Source, &ep.SourceID, &ep.CreatedAt); err != nil {
			return nil, err
		}
		episodes = append(episodes, ep)
	}
	return episodes, nil
}

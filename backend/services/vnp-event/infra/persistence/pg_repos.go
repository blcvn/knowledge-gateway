// Package persistence implements PostgreSQL repositories for vnp-event.
// Uses pgvector for semantic similarity search.
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/vnp-event/domain/model"
)

// EventRepo implements repository.EventRepository.
type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, e *model.UserEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_events (id, user_id, tenant_id, source, content, tags, embedding, created_at, valid_at, invalid_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.ID, e.UserID, e.TenantID, e.Source, e.Content, e.Tags, pgvecFloat32(e.Embedding),
		e.CreatedAt, e.ValidAt, e.InvalidAt,
	)
	return err
}

func (r *EventRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.UserEvent, error) {
	e := &model.UserEvent{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events WHERE id = $1`, id,
	).Scan(&e.ID, &e.UserID, &e.TenantID, &e.Source, &e.Content, &e.Tags, &e.CreatedAt, &e.ValidAt, &e.InvalidAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("event not found: %s", id)
	}
	return e, err
}

// SearchSemantic performs vector similarity search using pgvector cosine distance.
func (r *EventRepo) SearchSemantic(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]model.TimelineEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at,
		        1 - (embedding <=> $1::vector) AS score
		 FROM user_events
		 WHERE tenant_id = $2
		 ORDER BY embedding <=> $1::vector
		 LIMIT $3`,
		pgvecFloat32(embedding), tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.TimelineEntry
	for rows.Next() {
		e := &model.UserEvent{}
		var score float64
		if err := rows.Scan(&e.ID, &e.UserID, &e.TenantID, &e.Source, &e.Content, &e.Tags,
			&e.CreatedAt, &e.ValidAt, &e.InvalidAt, &score); err != nil {
			return nil, err
		}
		entries = append(entries, model.TimelineEntry{Event: e, Score: score})
	}
	return entries, nil
}

// SearchTemporal returns events within a time range using bi-temporal logic.
func (r *EventRepo) SearchTemporal(ctx context.Context, tenantID uuid.UUID, start, end time.Time, limit int) ([]*model.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events
		 WHERE tenant_id = $1
		   AND valid_at >= $2
		   AND valid_at <= $3
		   AND (invalid_at IS NULL OR invalid_at > $3)
		 ORDER BY valid_at DESC
		 LIMIT $4`,
		tenantID, start, end, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// GetTimeline returns a user's events ordered by valid_at.
func (r *EventRepo) GetTimeline(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]*model.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events
		 WHERE tenant_id = $1 AND user_id = $2
		 ORDER BY valid_at DESC
		 LIMIT $3`,
		tenantID, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

// FilterByTags returns events whose tags array overlaps with the given tags.
func (r *EventRepo) FilterByTags(ctx context.Context, tenantID uuid.UUID, tags []string, limit int) ([]*model.UserEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, source, content, tags, created_at, valid_at, invalid_at
		 FROM user_events
		 WHERE tenant_id = $1 AND tags && $2
		 ORDER BY valid_at DESC
		 LIMIT $3`,
		tenantID, tags, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func scanEvents(rows pgx.Rows) ([]*model.UserEvent, error) {
	var events []*model.UserEvent
	for rows.Next() {
		e := &model.UserEvent{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.TenantID, &e.Source, &e.Content, &e.Tags,
			&e.CreatedAt, &e.ValidAt, &e.InvalidAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GistRepo implements repository.GistRepository.
type GistRepo struct {
	pool *pgxpool.Pool
}

func NewGistRepo(pool *pgxpool.Pool) *GistRepo {
	return &GistRepo{pool: pool}
}

func (r *GistRepo) Create(ctx context.Context, g *model.EventGist) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO event_gists (id, event_ids, summary, embedding, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		g.ID, g.EventIDs, g.Summary, pgvecFloat32(g.Embedding), g.CreatedAt,
	)
	return err
}

func (r *GistRepo) SearchSemantic(ctx context.Context, tenantID uuid.UUID, embedding []float32, limit int) ([]*model.EventGist, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT g.id, g.event_ids, g.summary, g.created_at
		 FROM event_gists g
		 JOIN user_events e ON e.id = ANY(g.event_ids)
		 WHERE e.tenant_id = $1
		 GROUP BY g.id, g.event_ids, g.summary, g.created_at, g.embedding
		 ORDER BY g.embedding <=> $2::vector
		 LIMIT $3`,
		tenantID, pgvecFloat32(embedding), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gists []*model.EventGist
	for rows.Next() {
		g := &model.EventGist{}
		if err := rows.Scan(&g.ID, &g.EventIDs, &g.Summary, &g.CreatedAt); err != nil {
			return nil, err
		}
		gists = append(gists, g)
	}
	return gists, nil
}

// pgvecFloat32 converts []float32 to a pgvector-compatible string representation.
func pgvecFloat32(v []float32) string {
	if len(v) == 0 {
		return "[0]"
	}
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%f", f)
	}
	s += "]"
	return s
}

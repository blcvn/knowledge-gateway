// Package pg implements PostgreSQL repositories for storage-service.
//
// Covers: session, message, resource persistence.
// Part of storage-service (MERGE-P1-T4)
package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vnp-memory/services/storage-service/internal/domain/resource"
	"vnp-memory/services/storage-service/internal/domain/session"
)

// SessionRepo implements usecase/session.Repository.
type SessionRepo struct {
	pool *pgxpool.Pool
}

// NewSessionRepo creates a SessionRepo.
func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, s *session.ChatSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ov_sessions (id, tenant_id, base_dir, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.ID, s.TenantID, s.BaseDir, s.Status, s.CreatedAt,
	)
	return err
}

func (r *SessionRepo) FindByID(ctx context.Context, id string) (*session.ChatSession, error) {
	s := &session.ChatSession{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, base_dir, status, created_at FROM ov_sessions WHERE id = $1`, id,
	).Scan(&s.ID, &s.TenantID, &s.BaseDir, &s.Status, &s.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return s, err
}

func (r *SessionRepo) Update(ctx context.Context, s *session.ChatSession) error {
	s.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE ov_sessions SET status=$1 WHERE id=$2`,
		s.Status, s.ID,
	)
	return err
}

func (r *SessionRepo) AddMessage(ctx context.Context, msg *session.Message) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ov_messages (id, session_id, role, content, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.CreatedAt,
	)
	return err
}

func (r *SessionRepo) GetMessages(ctx context.Context, sessionID string) ([]*session.Message, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, session_id, role, content, created_at
		 FROM ov_messages WHERE session_id = $1 ORDER BY created_at ASC`, sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*session.Message
	for rows.Next() {
		m := &session.Message{}
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (r *SessionRepo) SaveCommit(ctx context.Context, record *session.CommitRecord) error {
	// For now store summary in a simple text field (not separate table)
	_, err := r.pool.Exec(ctx,
		`UPDATE ov_sessions SET status='committed' WHERE id=$1`, record.SessionID,
	)
	return err
}

// ResourceRepo implements usecase/resource.Repository.
type ResourceRepo struct {
	pool *pgxpool.Pool
}

// NewResourceRepo creates a ResourceRepo.
func NewResourceRepo(pool *pgxpool.Pool) *ResourceRepo {
	return &ResourceRepo{pool: pool}
}

func (r *ResourceRepo) Create(ctx context.Context, res *resource.Resource) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO ov_resources (id, tenant_id, uri, type, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		res.ID, res.TenantID, res.URI, res.Type, res.Status, res.CreatedAt,
	)
	return err
}

func (r *ResourceRepo) FindByID(ctx context.Context, id string) (*resource.Resource, error) {
	res := &resource.Resource{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, uri, type, status, created_at FROM ov_resources WHERE id = $1`, id,
	).Scan(&res.ID, &res.TenantID, &res.URI, &res.Type, &res.Status, &res.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("resource not found: %s", id)
	}
	return res, err
}

func (r *ResourceRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE ov_resources SET status=$1 WHERE id=$2`, status, id,
	)
	return err
}

func (r *ResourceRepo) List(ctx context.Context, tenantID string) ([]*resource.Resource, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, uri, type, status, created_at
		 FROM ov_resources WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*resource.Resource
	for rows.Next() {
		res := &resource.Resource{}
		if err := rows.Scan(&res.ID, &res.TenantID, &res.URI, &res.Type, &res.Status, &res.CreatedAt); err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	return resources, nil
}

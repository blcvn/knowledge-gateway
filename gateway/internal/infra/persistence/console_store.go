// Package persistence — Console PostgreSQL stores (T08 audit, T09 policy).
package persistence

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
)

// ──── T08: Audit Store ─────────────────────────────────────────

// PGAuditStore implements port.AuditStore using PostgreSQL.
type PGAuditStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPGAuditStore creates a new PostgreSQL-backed audit store.
func NewPGAuditStore(pool *pgxpool.Pool, logger *slog.Logger) *PGAuditStore {
	return &PGAuditStore{pool: pool, logger: logger}
}

// Insert records a new audit entry.
func (s *PGAuditStore) Insert(ctx context.Context, entry *domain.AuditEntry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_logs (tenant_id, user_id, action, resource, details, ip, user_agent, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		entry.TenantID, entry.UserID, entry.Action, entry.Resource,
		entry.Details, entry.IP, entry.UserAgent, entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("audit insert: %w", err)
	}
	return nil
}

// Search queries audit logs with filters. Returns entries and total count.
func (s *PGAuditStore) Search(ctx context.Context, filter *domain.AuditFilter) ([]*domain.AuditEntry, int, error) {
	// Build query with optional filters
	query := `SELECT id, tenant_id, user_id, action, resource, details, ip, user_agent, created_at
	           FROM audit_logs WHERE 1=1`
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	args := make([]any, 0)
	argIdx := 1

	if filter.TenantID != "" {
		clause := fmt.Sprintf(" AND tenant_id = $%d", argIdx)
		query += clause
		countQuery += clause
		args = append(args, filter.TenantID)
		argIdx++
	}
	if filter.UserID != "" {
		clause := fmt.Sprintf(" AND user_id = $%d", argIdx)
		query += clause
		countQuery += clause
		args = append(args, filter.UserID)
		argIdx++
	}
	if filter.Action != "" {
		clause := fmt.Sprintf(" AND action = $%d", argIdx)
		query += clause
		countQuery += clause
		args = append(args, filter.Action)
		argIdx++
	}
	if !filter.From.IsZero() {
		clause := fmt.Sprintf(" AND created_at >= $%d", argIdx)
		query += clause
		countQuery += clause
		args = append(args, filter.From)
		argIdx++
	}
	if !filter.To.IsZero() {
		clause := fmt.Sprintf(" AND created_at <= $%d", argIdx)
		query += clause
		countQuery += clause
		args = append(args, filter.To)
		argIdx++
	}

	// Get total count
	var total int
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit count: %w", err)
	}

	// Apply ordering and pagination
	query += " ORDER BY created_at DESC"
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit search: %w", err)
	}
	defer rows.Close()

	entries := make([]*domain.AuditEntry, 0)
	for rows.Next() {
		var e domain.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.UserID, &e.Action, &e.Resource,
			&e.Details, &e.IP, &e.UserAgent, &e.Timestamp,
		); err != nil {
			return nil, 0, fmt.Errorf("audit scan: %w", err)
		}
		entries = append(entries, &e)
	}

	return entries, total, nil
}

// ──── T09: Policy Store ────────────────────────────────────────

// PGPolicyStore implements port.PolicyStore using PostgreSQL.
type PGPolicyStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPGPolicyStore creates a new PostgreSQL-backed policy store.
func NewPGPolicyStore(pool *pgxpool.Pool, logger *slog.Logger) *PGPolicyStore {
	return &PGPolicyStore{pool: pool, logger: logger}
}

// List returns all policies for a tenant.
func (s *PGPolicyStore) List(ctx context.Context, tenantID string) ([]*domain.Policy, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, type, rego_body, enabled, metadata, created_at, updated_at
		 FROM policies WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("policy list: %w", err)
	}
	defer rows.Close()

	policies := make([]*domain.Policy, 0)
	for rows.Next() {
		var p domain.Policy
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Type,
			&p.RegoBody, &p.Enabled, &p.Metadata, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("policy scan: %w", err)
		}
		policies = append(policies, &p)
	}
	return policies, nil
}

// Get returns a single policy by ID.
func (s *PGPolicyStore) Get(ctx context.Context, id string) (*domain.Policy, error) {
	var p domain.Policy
	err := s.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, type, rego_body, enabled, metadata, created_at, updated_at
		 FROM policies WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.Type,
		&p.RegoBody, &p.Enabled, &p.Metadata, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, domain.ErrNotFound.WithMessage("policy not found: " + id)
	}
	return &p, nil
}

// Create inserts a new policy.
func (s *PGPolicyStore) Create(ctx context.Context, p *domain.Policy) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO policies (id, tenant_id, name, description, type, rego_body, enabled, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		p.ID, p.TenantID, p.Name, p.Description, p.Type,
		p.RegoBody, p.Enabled, p.Metadata, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("policy create: %w", err)
	}
	return nil
}

// Update modifies an existing policy.
func (s *PGPolicyStore) Update(ctx context.Context, p *domain.Policy) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE policies SET name=$1, description=$2, type=$3, rego_body=$4, enabled=$5, metadata=$6, updated_at=$7
		 WHERE id=$8`,
		p.Name, p.Description, p.Type, p.RegoBody, p.Enabled, p.Metadata, p.UpdatedAt, p.ID,
	)
	if err != nil {
		return fmt.Errorf("policy update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound.WithMessage("policy not found: " + p.ID)
	}
	return nil
}

// ──── Migration Extension ──────────────────────────────────────

// MigrateConsoleSchema creates the console-specific tables (audit_logs, policies).
func MigrateConsoleSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id          BIGSERIAL    PRIMARY KEY,
		tenant_id   VARCHAR(64)  NOT NULL,
		user_id     VARCHAR(64)  NOT NULL,
		action      VARCHAR(128) NOT NULL,
		resource    VARCHAR(512) NOT NULL,
		details     JSONB        DEFAULT '{}',
		ip          VARCHAR(45),
		user_agent  TEXT,
		created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_audit_tenant_time ON audit_logs(tenant_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
	CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);

	CREATE TABLE IF NOT EXISTS policies (
		id          VARCHAR(64)  PRIMARY KEY,
		tenant_id   VARCHAR(64)  NOT NULL REFERENCES tenants(id),
		name        VARCHAR(255) NOT NULL,
		description TEXT,
		type        VARCHAR(64)  NOT NULL DEFAULT 'access',
		rego_body   TEXT         NOT NULL,
		enabled     BOOLEAN      NOT NULL DEFAULT true,
		metadata    JSONB        DEFAULT '{}',
		created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_policies_type ON policies(type);
	`
	_, err := pool.Exec(ctx, schema)
	return err
}

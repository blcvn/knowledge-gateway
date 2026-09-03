// Package persistence — Console PostgreSQL stores (T08 audit, T09 policy).
package persistence

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vnp-community/vnp-memory/gateway/domain"
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

// MigrateAuthSchema creates console_users và refresh_tokens tables (TASK-BE-001).
// Gọi sau MigrateSchema để đảm bảo bảng tenants đã tồn tại.
func MigrateAuthSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS console_users (
		id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		name          TEXT        NOT NULL,
		email         TEXT        UNIQUE NOT NULL,
		password_hash TEXT,
		role          TEXT        NOT NULL DEFAULT 'user',
		tenant_id     VARCHAR(64),
		avatar_url    TEXT,
		is_active     BOOLEAN     NOT NULL DEFAULT true,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_console_users_email ON console_users(email);
	CREATE INDEX IF NOT EXISTS idx_console_users_tenant_id ON console_users(tenant_id);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id       UUID        NOT NULL REFERENCES console_users(id) ON DELETE CASCADE,
		token_hash    TEXT        UNIQUE NOT NULL,
		expires_at    TIMESTAMPTZ NOT NULL,
		revoked       BOOLEAN     NOT NULL DEFAULT false,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

	-- Sessions table (TASK-BE-004)
	CREATE TABLE IF NOT EXISTS sessions (
		id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id    VARCHAR(64) NOT NULL,
		user_id      TEXT        NOT NULL,
		agent_id     TEXT,
		engine       TEXT        NOT NULL DEFAULT 'zep',
		status       TEXT        NOT NULL DEFAULT 'active',
		started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		ended_at     TIMESTAMPTZ,
		metadata     JSONB       DEFAULT '{}',
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_tenant_status ON sessions(tenant_id, status);
	CREATE INDEX IF NOT EXISTS idx_sessions_user_id       ON sessions(tenant_id, user_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_created_at    ON sessions(tenant_id, created_at DESC);

	-- Messages table (TASK-BE-004)
	CREATE TABLE IF NOT EXISTS messages (
		id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		session_id     UUID        NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		tenant_id      VARCHAR(64) NOT NULL,
		role           TEXT        NOT NULL,
		content        TEXT        NOT NULL,
		memory_sources TEXT[]      DEFAULT '{}',
		tokens         INT         DEFAULT 0,
		created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id, created_at ASC);
	CREATE INDEX IF NOT EXISTS idx_messages_tenant_id  ON messages(tenant_id);
	`
	_, err := pool.Exec(ctx, schema)
	return err
}

// MigrateGovernanceSchema creates audit_log_entries + opa_policies tables (TASK-BE-009).
func MigrateGovernanceSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_log_entries (
		id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id   VARCHAR(64) NOT NULL,
		actor_id    VARCHAR(64) NOT NULL,
		action      TEXT        NOT NULL,
		entity_type TEXT        NOT NULL,
		entity_id   TEXT,
		result      TEXT        NOT NULL DEFAULT 'success',
		metadata    JSONB       DEFAULT '{}',
		ip          TEXT,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_audit_log_tenant ON audit_log_entries(tenant_id, created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_log_actor  ON audit_log_entries(actor_id);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log_entries(action);

	CREATE TABLE IF NOT EXISTS opa_policies (
		id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id   VARCHAR(64) NOT NULL,
		name        TEXT        NOT NULL,
		rego_code   TEXT        NOT NULL,
		scope       TEXT        NOT NULL DEFAULT 'memory:*',
		enabled     BOOLEAN     NOT NULL DEFAULT true,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_opa_policies_tenant ON opa_policies(tenant_id);
	`
	_, err := pool.Exec(ctx, schema)
	return err
}

// MigrateObservabilitySchema creates error_aggregates + llm_cost_events tables (TASK-BE-010).
func MigrateObservabilitySchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS error_aggregates (
		id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id       VARCHAR(64) NOT NULL,
		service         TEXT        NOT NULL,
		message         TEXT        NOT NULL,
		message_hash    TEXT        NOT NULL,
		count           INT         NOT NULL DEFAULT 1,
		last_occurrence TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		stack           TEXT,
		created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(tenant_id, message_hash)
	);

	CREATE INDEX IF NOT EXISTS idx_error_agg_tenant  ON error_aggregates(tenant_id, last_occurrence DESC);
	CREATE INDEX IF NOT EXISTS idx_error_agg_service ON error_aggregates(service);

	CREATE TABLE IF NOT EXISTS llm_cost_events (
		id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id     VARCHAR(64) NOT NULL,
		model         TEXT        NOT NULL,
		input_tokens  INT         NOT NULL DEFAULT 0,
		output_tokens INT         NOT NULL DEFAULT 0,
		cost_usd      FLOAT       NOT NULL DEFAULT 0,
		service       TEXT,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_llm_cost_tenant ON llm_cost_events(tenant_id, created_at DESC);
	`
	_, err := pool.Exec(ctx, schema)
	return err
}

// MigrateOrgSDKSchema creates sdk_api_keys + webhooks tables (TASK-BE-013).
func MigrateOrgSDKSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sdk_api_keys (
		id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id    VARCHAR(64) NOT NULL,
		name         TEXT        NOT NULL,
		key_hash     TEXT        UNIQUE NOT NULL,
		prefix       TEXT        NOT NULL,
		permissions  TEXT[]      DEFAULT '{}',
		expires_at   TIMESTAMPTZ,
		last_used_at TIMESTAMPTZ,
		revoked      BOOLEAN     NOT NULL DEFAULT false,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_sdk_keys_tenant ON sdk_api_keys(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_sdk_keys_hash   ON sdk_api_keys(key_hash);

	CREATE TABLE IF NOT EXISTS webhooks (
		id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
		tenant_id    VARCHAR(64) NOT NULL,
		url          TEXT        NOT NULL,
		events       TEXT[]      NOT NULL DEFAULT '{}',
		secret_hash  TEXT,
		status       TEXT        NOT NULL DEFAULT 'active',
		success_rate FLOAT       NOT NULL DEFAULT 100.0,
		created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_webhooks_tenant ON webhooks(tenant_id);
	`
	_, err := pool.Exec(ctx, schema)
	return err
}

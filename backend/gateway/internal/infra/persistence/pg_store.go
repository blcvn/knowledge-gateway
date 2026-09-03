package persistence

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
)

// PGTenantStore implements port.TenantStore and port.KeyStore using PostgreSQL.
type PGTenantStore struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPGTenantStore creates a new PostgreSQL-backed tenant/key store.
func NewPGTenantStore(pool *pgxpool.Pool, logger *slog.Logger) *PGTenantStore {
	return &PGTenantStore{pool: pool, logger: logger}
}

// GetTenant retrieves tenant context by ID.
func (s *PGTenantStore) GetTenant(ctx context.Context, id string) (*domain.TenantContext, error) {
	var tc domain.TenantContext
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, rate_tier, enabled, created_at FROM tenants WHERE id = $1`,
		id,
	).Scan(&tc.ID, &tc.Name, &tc.RateTier, &tc.Enabled, &tc.CreatedAt)
	if err != nil {
		s.logger.Debug("tenant not found", "id", id, "error", err)
		return nil, domain.ErrNotFound.WithMessage("tenant not found: " + id)
	}

	if !tc.Enabled {
		return nil, domain.ErrForbidden.WithMessage("tenant is disabled")
	}

	return &tc, nil
}

// ResolveAPIKey looks up an API key hash and returns the associated AuthContext.
func (s *PGTenantStore) ResolveAPIKey(ctx context.Context, keyHash string) (*domain.AuthContext, error) {
	var (
		tenantID  string
		userID    string
		rateTier  string
		revokedAt *time.Time
		expiresAt *time.Time
	)

	err := s.pool.QueryRow(ctx,
		`SELECT t.id, ak.user_id, t.rate_tier, ak.revoked_at, ak.expires_at
		 FROM api_keys ak
		 JOIN tenants t ON t.id = ak.tenant_id
		 WHERE ak.key_hash = $1 AND t.enabled = true`,
		keyHash,
	).Scan(&tenantID, &userID, &rateTier, &revokedAt, &expiresAt)

	if err != nil {
		return nil, fmt.Errorf("api key lookup: %w", err)
	}

	// Check revocation
	if revokedAt != nil {
		return nil, domain.ErrUnauthenticated.WithMessage("API key has been revoked")
	}

	// Check expiry
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, domain.ErrUnauthenticated.WithMessage("API key has expired")
	}

	// Update last_used timestamp asynchronously
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := s.pool.Exec(bgCtx,
			`UPDATE api_keys SET last_used_at = NOW() WHERE key_hash = $1`,
			keyHash,
		)
		if err != nil {
			s.logger.Error("failed to update api key last_used_at", "error", err)
		}
	}()

	return &domain.AuthContext{
		TenantID: tenantID,
		UserID:   userID,
		Roles:    []string{"api_key"},
		Scopes:   []string{"*"},
		RateTier: rateTier,
	}, nil
}

// NewPGPool creates a new PostgreSQL connection pool.
func NewPGPool(dsn string, maxConns, minConns int) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}
	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg ping: %w", err)
	}

	return pool, nil
}

// MigrateSchema creates the required tables if they don't exist.
func MigrateSchema(ctx context.Context, pool *pgxpool.Pool) error {
	schema := `
	CREATE TABLE IF NOT EXISTS tenants (
		id         VARCHAR(64)  PRIMARY KEY,
		name       VARCHAR(255) NOT NULL,
		rate_tier  VARCHAR(32)  NOT NULL DEFAULT 'free',
		enabled    BOOLEAN      NOT NULL DEFAULT true,
		metadata   JSONB        DEFAULT '{}',
		created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS api_keys (
		id          SERIAL       PRIMARY KEY,
		tenant_id   VARCHAR(64)  NOT NULL REFERENCES tenants(id),
		user_id     VARCHAR(64)  NOT NULL,
		key_hash    VARCHAR(128) NOT NULL UNIQUE,
		key_prefix  VARCHAR(16)  NOT NULL,
		label       VARCHAR(255),
		scopes      TEXT[]       DEFAULT ARRAY['*'],
		revoked_at  TIMESTAMPTZ,
		expires_at  TIMESTAMPTZ,
		last_used_at TIMESTAMPTZ,
		created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
	CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
	`

	_, err := pool.Exec(ctx, schema)
	return err
}

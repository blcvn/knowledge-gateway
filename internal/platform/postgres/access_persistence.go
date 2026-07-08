package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"kg-service/internal/access"
)

// AccessPersistence implements access.PersistenceStore backed by Postgres.
// It writes dynamically created tenants and apps into the `tenants` / `apps`
// tables so that subsequent writes (which carry FK references) succeed.
type AccessPersistence struct {
	db *sql.DB
}

// NewAccessPersistence creates a new AccessPersistence backed by db.
func NewAccessPersistence(db *sql.DB) *AccessPersistence {
	return &AccessPersistence{db: db}
}

// PersistTenant inserts a tenant row; silently ignores duplicate-key errors.
func (p *AccessPersistence) PersistTenant(t access.Tenant) error {
	const q = `
INSERT INTO tenants (id, slug, name, status, tier, default_sharing_policy, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, '{}'::jsonb, $7, $8)
ON CONFLICT (id) DO NOTHING`

	now := time.Now().UTC()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = now
	}
	dsp := t.DefaultSharingPolicy
	if dsp == "" {
		dsp = "deny_all"
	}

	_, err := p.db.Exec(q,
		t.ID, t.Slug, t.Name, t.Status, t.Tier, dsp,
		t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("access_persistence: persist tenant %s: %w", t.ID, err)
	}
	return nil
}

// PersistApp inserts an app row; silently ignores duplicate-key errors.
func (p *AccessPersistence) PersistApp(a access.App) error {
	const q = `
INSERT INTO apps (id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO NOTHING`

	now := time.Now().UTC()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}

	_, err := p.db.Exec(q,
		a.ID, a.TenantID, a.Slug, a.Name, a.Type,
		a.APIKeyHash, a.APIKeyPrefix, a.Status, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("access_persistence: persist app %s: %w", a.ID, err)
	}
	return nil
}

// Package persistence — APIKey and User repository implementations for admin domain.
//
// Extends pg_repos.go which covers Tenant, Event, Usage, Space repos.
// Added: APIKeyRepo, AdminUserRepo (MERGE-P1-T2)
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
)

// APIKeyRepo implements port.APIKeyRepository.
type APIKeyRepo struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepo creates a new APIKeyRepo.
func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

// Create stores a new API key.
func (r *APIKeyRepo) Create(ctx context.Context, key *admin.APIKey) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, permissions, active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, true, $7)`,
		key.ID, key.TenantID, key.Name, key.KeyHash, key.KeyPrefix, key.Permissions, key.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

// FindByHash retrieves an API key by its SHA256 hash.
func (r *APIKeyRepo) FindByHash(ctx context.Context, hash string) (*admin.APIKey, error) {
	key := &admin.APIKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, permissions, active, created_at, revoked_at
		 FROM api_keys WHERE key_hash = $1`, hash,
	).Scan(&key.ID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix,
		&key.Permissions, &key.Active, &key.CreatedAt, &key.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("find api key: %w", err)
	}
	return key, nil
}

// Revoke marks an API key as revoked.
func (r *APIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET active = false, revoked_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	return nil
}

// ListByTenant retrieves all active API keys for a tenant.
func (r *APIKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*admin.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, permissions, active, created_at, revoked_at
		 FROM api_keys WHERE tenant_id = $1 AND active = true ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []*admin.APIKey
	for rows.Next() {
		key := &admin.APIKey{}
		if err := rows.Scan(&key.ID, &key.TenantID, &key.Name, &key.KeyHash, &key.KeyPrefix,
			&key.Permissions, &key.Active, &key.CreatedAt, &key.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// AdminUserRepo implements port.UserRepository (tenant-scoped users in admin domain).
type AdminUserRepo struct {
	pool *pgxpool.Pool
}

// NewAdminUserRepo creates a new AdminUserRepo.
func NewAdminUserRepo(pool *pgxpool.Pool) *AdminUserRepo {
	return &AdminUserRepo{pool: pool}
}

// Create inserts a new tenant user.
func (r *AdminUserRepo) Create(ctx context.Context, u *admin.User) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, name, role, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, true, $6, $7)`,
		u.ID, u.TenantID, u.Email, u.Name, u.Role, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// FindByID retrieves a tenant user by UUID.
func (r *AdminUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*admin.User, error) {
	u := &admin.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, role, active, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

// Update persists changes to a tenant user.
func (r *AdminUserRepo) Update(ctx context.Context, u *admin.User) error {
	u.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name=$1, role=$2, active=$3, updated_at=$4 WHERE id=$5`,
		u.Name, u.Role, u.Active, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// Delete removes a tenant user.
func (r *AdminUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// ListByTenant retrieves all users for a tenant with pagination.
func (r *AdminUserRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*admin.User, int, error) {
	var total int
	_ = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, tenantID).Scan(&total)

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, email, name, role, active, created_at, updated_at
		 FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*admin.User
	for rows.Next() {
		u := &admin.User{}
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &u.Active,
			&u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

// Package persistence implements PostgreSQL repositories for vnp-admin.
package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/vnp-admin/internal/domain/model"
)

// TenantRepo implements repository.TenantRepository.
type TenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) *TenantRepo {
	return &TenantRepo{pool: pool}
}

func (r *TenantRepo) Create(ctx context.Context, t *model.Tenant) error {
	configJSON, err := json.Marshal(t.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, plan, config, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.ID, t.Name, t.Plan, configJSON, t.Active, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (r *TenantRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	t := &model.Tenant{}
	var configJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, plan, config, active, created_at, updated_at
		 FROM tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Plan, &configJSON, &t.Active, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, model.ErrTenantNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configJSON, &t.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) FindByName(ctx context.Context, name string) (*model.Tenant, error) {
	t := &model.Tenant{}
	var configJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, plan, config, active, created_at, updated_at
		 FROM tenants WHERE name = $1`, name,
	).Scan(&t.ID, &t.Name, &t.Plan, &configJSON, &t.Active, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(configJSON, &t.Config)
	return t, nil
}

func (r *TenantRepo) Update(ctx context.Context, t *model.Tenant) error {
	configJSON, _ := json.Marshal(t.Config)
	_, err := r.pool.Exec(ctx,
		`UPDATE tenants SET name=$1, plan=$2, config=$3, active=$4, updated_at=$5 WHERE id=$6`,
		t.Name, t.Plan, configJSON, t.Active, t.UpdatedAt, t.ID,
	)
	return err
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrTenantNotFound
	}
	return nil
}

func (r *TenantRepo) List(ctx context.Context, offset, limit int) ([]*model.Tenant, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, name, plan, config, active, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []*model.Tenant
	for rows.Next() {
		t := &model.Tenant{}
		var configJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Plan, &configJSON, &t.Active, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(configJSON, &t.Config)
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

// APIKeyRepo implements repository.APIKeyRepository.
type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

func (r *APIKeyRepo) Create(ctx context.Context, k *model.APIKey) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, tenant_id, name, key_hash, key_prefix, scope, rate_limit, active, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		k.ID, k.TenantID, k.Name, k.KeyHash, k.KeyPrefix, k.Scope, k.RateLimit, k.Active, k.CreatedAt,
	)
	return err
}

func (r *APIKeyRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.APIKey, error) {
	k := &model.APIKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, scope, rate_limit, active, created_at, revoked_at
		 FROM api_keys WHERE id = $1`, id,
	).Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Scope, &k.RateLimit, &k.Active, &k.CreatedAt, &k.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, model.ErrAPIKeyNotFound
	}
	return k, err
}

func (r *APIKeyRepo) FindByHash(ctx context.Context, hash string) (*model.APIKey, error) {
	k := &model.APIKey{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, scope, rate_limit, active, created_at, revoked_at
		 FROM api_keys WHERE key_hash = $1`, hash,
	).Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Scope, &k.RateLimit, &k.Active, &k.CreatedAt, &k.RevokedAt)
	if err == pgx.ErrNoRows {
		return nil, model.ErrAPIKeyNotFound
	}
	return k, err
}

func (r *APIKeyRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, key_hash, key_prefix, scope, rate_limit, active, created_at, revoked_at
		 FROM api_keys WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		k := &model.APIKey{}
		if err := rows.Scan(&k.ID, &k.TenantID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Scope, &k.RateLimit, &k.Active, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	tag, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET active = false, revoked_at = $1 WHERE id = $2 AND active = true`,
		now, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrAPIKeyNotFound
	}
	return nil
}

// UserRepo implements repository.UserRepository.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	metaJSON, _ := json.Marshal(u.Metadata)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, tenant_id, email, name, role, metadata, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, u.TenantID, u.Email, u.Name, u.Role, metaJSON, u.Active, u.CreatedAt, u.UpdatedAt,
	)
	return err
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	u := &model.User{}
	var metaJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, role, metadata, active, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &metaJSON, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, model.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(metaJSON, &u.Metadata)
	return u, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*model.User, error) {
	u := &model.User{}
	var metaJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, email, name, role, metadata, active, created_at, updated_at
		 FROM users WHERE tenant_id = $1 AND email = $2`, tenantID, email,
	).Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &metaJSON, &u.Active, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(metaJSON, &u.Metadata)
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, u *model.User) error {
	metaJSON, _ := json.Marshal(u.Metadata)
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email=$1, name=$2, role=$3, metadata=$4, active=$5, updated_at=$6 WHERE id=$7`,
		u.Email, u.Name, u.Role, metaJSON, u.Active, u.UpdatedAt, u.ID,
	)
	return err
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) ListByTenant(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*model.User, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, email, name, role, metadata, active, created_at, updated_at
		 FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var metaJSON []byte
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Email, &u.Name, &u.Role, &metaJSON, &u.Active, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(metaJSON, &u.Metadata)
		users = append(users, u)
	}
	return users, total, nil
}

// Package persistence implements PostgreSQL repositories for auth domain.
//
// Absorbed from: sm-auth InMemoryUserRepository (MERGE-P1-T1)
// Replaces in-memory implementation with PostgreSQL backend.
package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/auth"
)

// AuthUserRepo implements usecase/auth.AuthRepository using PostgreSQL.
type AuthUserRepo struct {
	pool *pgxpool.Pool
}

// NewAuthUserRepo creates a new AuthUserRepo.
func NewAuthUserRepo(pool *pgxpool.Pool) *AuthUserRepo {
	return &AuthUserRepo{pool: pool}
}

// FindByEmail looks up a user by email address.
func (r *AuthUserRepo) FindByEmail(ctx context.Context, email string) (*auth.AuthUser, error) {
	u := &auth.AuthUser{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, auth_provider, auth_provider_id, role, created_at, updated_at
		 FROM auth_users WHERE email = $1 AND active = true`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider, &u.AuthProviderID,
		&u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil // not found is not an error
	}
	if err != nil {
		return nil, fmt.Errorf("find by email: %w", err)
	}
	return u, nil
}

// FindByProviderID looks up a user by OAuth provider + provider user ID.
func (r *AuthUserRepo) FindByProviderID(ctx context.Context, provider, providerID string) (*auth.AuthUser, error) {
	u := &auth.AuthUser{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, auth_provider, auth_provider_id, role, created_at, updated_at
		 FROM auth_users WHERE auth_provider = $1 AND auth_provider_id = $2 AND active = true`,
		provider, providerID,
	).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.AuthProvider, &u.AuthProviderID,
		&u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by provider id: %w", err)
	}
	return u, nil
}

// Create inserts a new AuthUser into the database.
func (r *AuthUserRepo) Create(ctx context.Context, u *auth.AuthUser) error {
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	_, err := r.pool.Exec(ctx,
		`INSERT INTO auth_users (id, email, name, password_hash, auth_provider, auth_provider_id, role, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, $9)`,
		u.ID, u.Email, u.Name, u.PasswordHash, u.AuthProvider, u.AuthProviderID,
		u.Role, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create auth user: %w", err)
	}
	return nil
}

// Update persists changes to an existing AuthUser.
func (r *AuthUserRepo) Update(ctx context.Context, u *auth.AuthUser) error {
	u.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE auth_users SET name=$1, password_hash=$2, role=$3, updated_at=$4 WHERE id=$5`,
		u.Name, u.PasswordHash, u.Role, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update auth user: %w", err)
	}
	return nil
}

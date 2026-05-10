// Package auth defines domain entities for the auth sub-domain.
//
// Absorbed from: sm-auth
package auth

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents an org-level grouping (from sm-auth).
type Organization struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// RBACPolicy defines a permission policy.
type RBACPolicy struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Role       string    `json:"role"`
	Resource   string    `json:"resource"`
	Action     string    `json:"action"` // read, write, delete, admin
	Effect     string    `json:"effect"` // allow, deny
	CreatedAt  time.Time `json:"created_at"`
}

// JWTClaims represents validated JWT token claims.
type JWTClaims struct {
	TenantID uuid.UUID `json:"tenant_id"`
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Issuer   string    `json:"iss"`
	IssuedAt time.Time `json:"iat"`
	ExpireAt time.Time `json:"exp"`
}

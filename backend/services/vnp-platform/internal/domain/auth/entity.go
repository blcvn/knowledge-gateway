// Package auth defines domain entities for the auth sub-domain of vnp-platform.
//
// Absorbed from: sm-auth (MERGE-P1-T1)
package auth

import "time"

// AuthUser represents an authenticated user in the system.
// Note: admin.User covers tenant-scoped users; AuthUser covers SSO/JWT auth users.
type AuthUser struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	PasswordHash   string    `json:"-"`
	AuthProvider   string    `json:"auth_provider"`   // "email" | "google"
	AuthProviderID string    `json:"auth_provider_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AuthToken represents a JWT auth token response.
type AuthToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
}

// Credentials represents email+password login credentials.
type Credentials struct {
	Email    string
	Password string
}

// Organization represents an org-level grouping.
type Organization struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// JWTClaims represents validated JWT token claims (for gateway middleware).
type JWTClaims struct {
	TenantID string    `json:"tid"`
	UserID   string    `json:"uid"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	Role     string    `json:"role"`
	Issuer   string    `json:"iss"`
	IssuedAt time.Time `json:"iat"`
	ExpireAt time.Time `json:"exp"`
}

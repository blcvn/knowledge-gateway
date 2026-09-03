package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// KeyScope defines what operations an API key permits.
type KeyScope string

const (
	KeyScopeReadOnly  KeyScope = "read"
	KeyScopeReadWrite KeyScope = "read_write"
	KeyScopeAdmin     KeyScope = "admin"
)

// APIKey represents a hashed API key bound to a tenant.
// The plaintext key is returned ONLY once at creation time.
type APIKey struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	KeyHash   string    `json:"-"`            // SHA-256 hash, never exposed
	KeyPrefix string    `json:"key_prefix"`   // First 8 chars for identification
	Scope     KeyScope  `json:"scope"`
	RateLimit int       `json:"rate_limit"`   // RPM override (0 = use tenant default)
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// GenerateAPIKey creates a new API key with a random 32-byte token.
// Returns (key entity with hash, plaintext key).
func GenerateAPIKey(tenantID uuid.UUID, name string, scope KeyScope) (*APIKey, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generate key: %w", err)
	}

	plaintext := "vnp_" + hex.EncodeToString(raw) // vnp_ prefix + 64 hex chars
	hash := sha256.Sum256([]byte(plaintext))

	key := &APIKey{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		KeyHash:   hex.EncodeToString(hash[:]),
		KeyPrefix: plaintext[:12], // "vnp_" + first 8 hex chars
		Scope:     scope,
		Active:    true,
		CreatedAt: time.Now(),
	}

	return key, plaintext, nil
}

// ValidateKey checks if a plaintext key matches this API key's hash.
func (k *APIKey) ValidateKey(plaintext string) bool {
	hash := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(hash[:]) == k.KeyHash
}

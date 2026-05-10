// Package crypto defines domain entities for the OpenViking envelope encryption.
package crypto

import (
	"time"

	"github.com/google/uuid"
)

// EncryptionKey represents a data encryption key (DEK).
type EncryptionKey struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Algorithm    string    `json:"algorithm"` // AES-256-GCM
	WrappedKey   []byte    `json:"-"`         // Encrypted by KEK, never exposed
	KEKVersion   int       `json:"kek_version"`
	CreatedAt    time.Time `json:"created_at"`
	RotatedAt    *time.Time `json:"rotated_at,omitempty"`
}

// KMSBackend enumerates supported KMS backends.
type KMSBackend string

const (
	KMSLocal    KMSBackend = "local"     // File-based KEK (dev)
	KMSVault    KMSBackend = "vault"     // HashiCorp Vault
	KMSAWS      KMSBackend = "aws_kms"   // AWS KMS
)

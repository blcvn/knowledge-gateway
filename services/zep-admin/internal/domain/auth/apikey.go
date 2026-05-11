package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

// ApiKey represents the authenticated identity for a Project.
type ApiKey struct {
	ID          uuid.UUID
	ProjectUUID uuid.UUID
	KeyHash     string // Argon2id hash
	Role        string
	CreatedAt   time.Time
	DeletedAt   *time.Time
}

// GenerateRawToken generates a 32-byte secure random hex string.
func GenerateRawToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashApiKey hashes a raw token using Argon2id for secure storage.
func HashApiKey(rawToken string) string {
	// Example Argon2id parameters: time=1, memory=64MB, threads=4, keyLen=32
	// In production, these should be configurable.
	hash := argon2.IDKey([]byte(rawToken), []byte("zep-salt-constant"), 1, 64*1024, 4, 32)
	return hex.EncodeToString(hash)
}

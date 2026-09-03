package model

import "time"

// KeyStatus represents the status of a cryptographic key
type KeyStatus string

const (
	KeyStatusActive  KeyStatus = "active"
	KeyStatusExpired KeyStatus = "expired"
	KeyStatusRevoked KeyStatus = "revoked"
)

// KeyVersion is the version identifier for a key
type KeyVersion int

// KeyMaterial represents raw key data
type KeyMaterial struct {
	Data []byte
}

// AccountKey represents an account's key metadata
type AccountKey struct {
	ID                 string
	AccountID          string
	KeyVersion         KeyVersion
	ProviderType       string
	EncryptedRootKey   []byte // Only populated for Local provider
	KeyReference       string // ARN or Vault Path
	Status             KeyStatus
	CreatedAt          time.Time
	RotatedAt          *time.Time
	ExpiresAt          *time.Time
}

// KeyRotationLog represents the audit log entry for a key rotation event
type KeyRotationLog struct {
	ID             string
	AccountID      string
	OldVersion     KeyVersion
	NewVersion     KeyVersion
	Reason         string
	InitiatedBy    string
	Status         string // completed, failed, in_progress
	FilesReWrapped int
	DurationMs     int
	CreatedAt      time.Time
}

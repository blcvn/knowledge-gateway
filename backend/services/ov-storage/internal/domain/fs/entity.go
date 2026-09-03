// Package fs defines domain entities for the OpenViking VikingFS.
package fs

import (
	"time"

	"github.com/google/uuid"
)

// File represents a file in the VikingFS virtual filesystem.
type File struct {
	ID         uuid.UUID `json:"id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	Encrypted  bool      `json:"encrypted"`
	TierLevel  TieredLevel `json:"tier_level"`
	Checksum   string    `json:"checksum"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Directory represents a directory node in VikingFS.
type Directory struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TieredLevel defines the storage tier for context retrieval.
type TieredLevel string

const (
	TierL0Hot  TieredLevel = "L0" // <10KB, instant access
	TierL1Warm TieredLevel = "L1" // <100KB, fast access
	TierL2Cold TieredLevel = "L2" // archival
)

// PathLock provides concurrent access control for file operations.
type PathLock struct {
	ID       uuid.UUID `json:"id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Path     string    `json:"path"`
	LockType LockType  `json:"lock_type"`
	HeldBy   string    `json:"held_by"` // session/request ID
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// LockType defines locking granularity.
type LockType string

const (
	LockPoint   LockType = "point"   // Single file
	LockSubtree LockType = "subtree" // Directory + children
	LockMove    LockType = "move"    // Source + destination
)

// VikingURI is a typed path reference: viking://<tenant>/<path>
type VikingURI string

// Package ingestion defines domain entities for the Memobase buffer zone.
package ingestion

import (
	"time"

	"github.com/google/uuid"
)

// Blob represents a raw conversational data blob.
type Blob struct {
	ID        uuid.UUID      `json:"id"`
	TenantID  uuid.UUID      `json:"tenant_id"`
	UserID    uuid.UUID      `json:"user_id"`
	Content   string         `json:"content"`
	Type      string         `json:"type"` // chat, message, note
	Tokens    int            `json:"tokens"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// BufferZone represents a token-aware batch accumulator (FSM).
type BufferZone struct {
	ID           uuid.UUID   `json:"id"`
	TenantID     uuid.UUID   `json:"tenant_id"`
	UserID       uuid.UUID   `json:"user_id"`
	State        BufferState `json:"state"`
	TokenCount   int         `json:"token_count"`
	Threshold    int         `json:"threshold"` // default 1024
	BlobIDs      []uuid.UUID `json:"blob_ids"`
	LastFlushed  *time.Time  `json:"last_flushed,omitempty"`
}

// BufferState represents the FSM states.
type BufferState string

const (
	BufferIdle       BufferState = "idle"
	BufferAccumulating BufferState = "accumulating"
	BufferReady      BufferState = "ready"
	BufferProcessing BufferState = "processing"
	BufferDone       BufferState = "done"
)

// DefaultTokenThreshold is the buffer flush trigger point.
const DefaultTokenThreshold = 1024

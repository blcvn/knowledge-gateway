package model

import (
	"time"
)

// BufferStatus represents the FSM state of a buffer zone entry
type BufferStatus string

const (
	BufferStatusIdle       BufferStatus = "idle"
	BufferStatusProcessing BufferStatus = "processing"
	BufferStatusDone       BufferStatus = "done"
	BufferStatusFailed     BufferStatus = "failed"
)

// BufferZone represents the token-aware buffer for blobs
type BufferZone struct {
	ID        string       `json:"id"`
	UserID    string       `json:"user_id"`
	ProjectID string       `json:"project_id"`
	BlobID    string       `json:"blob_id"`
	BlobType  BlobType     `json:"blob_type"`
	TokenSize int          `json:"token_size"`
	Status    BufferStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ─── BlobType ───────────────────────────────────────────────────────────────────

// BlobType identifies the kind of memory content in a blob.
type BlobType string

const (
	BlobTypeChat    BlobType = "chat"
	BlobTypeDoc     BlobType = "doc"
	BlobTypeSummary BlobType = "summary"
)

// ─── Errors ─────────────────────────────────────────────────────────────────────

var (
	ErrEmptyBlobContent  = errors.New("blob content must not be empty")
	ErrInvalidBlobRole   = errors.New("invalid message role: must be 'user', 'assistant', or 'system'")
	ErrUnknownBlobType   = errors.New("unknown blob type")
)

// ─── BlobData variants ──────────────────────────────────────────────────────────

// BlobData is the union interface for typed blob content.
type BlobData interface {
	Validate() error
	blobDataMarker()
}

// ChatMessage is a single turn in a conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ValidRoles contains allowed message roles.
var validRoles = map[string]bool{
	"user":      true,
	"assistant": true,
	"system":    true,
}

// ChatBlobData holds a multi-turn conversation.
type ChatBlobData struct {
	Messages []ChatMessage `json:"messages"`
}

func (c *ChatBlobData) blobDataMarker() {}

func (c *ChatBlobData) Validate() error {
	if len(c.Messages) == 0 {
		return ErrEmptyBlobContent
	}
	for _, m := range c.Messages {
		if !validRoles[m.Role] {
			return fmt.Errorf("%w: got %q", ErrInvalidBlobRole, m.Role)
		}
	}
	return nil
}

// DocBlobData holds a document or text passage.
type DocBlobData struct {
	Text string `json:"text"`
}

func (d *DocBlobData) blobDataMarker() {}

func (d *DocBlobData) Validate() error {
	if d.Text == "" {
		return ErrEmptyBlobContent
	}
	return nil
}

// SummaryBlobData holds a pre-computed summary.
type SummaryBlobData struct {
	Text string `json:"text"`
}

func (s *SummaryBlobData) blobDataMarker() {}

func (s *SummaryBlobData) Validate() error {
	if s.Text == "" {
		return ErrEmptyBlobContent
	}
	return nil
}

// DeserializeBlobData parses raw JSON bytes into the concrete BlobData type.
func DeserializeBlobData(data []byte, blobType BlobType) (BlobData, error) {
	switch blobType {
	case BlobTypeChat:
		var d ChatBlobData
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("deserialize chat blob: %w", err)
		}
		return &d, nil
	case BlobTypeDoc:
		var d DocBlobData
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("deserialize doc blob: %w", err)
		}
		return &d, nil
	case BlobTypeSummary:
		var d SummaryBlobData
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("deserialize summary blob: %w", err)
		}
		return &d, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownBlobType, blobType)
	}
}

// ─── Blob ───────────────────────────────────────────────────────────────────────

// Blob is a raw memory unit persisted in the general_blobs table.
type Blob struct {
	ID               uuid.UUID
	ProjectID        string
	UserID           uuid.UUID
	BlobType         BlobType
	BlobData         BlobData
	AdditionalFields map[string]any
	CreatedAt        time.Time
}

// ─── BufferZone ─────────────────────────────────────────────────────────────────

// BufferStatus represents the lifecycle state of a buffer entry.
type BufferStatus string

const (
	BufferStatusIdle       BufferStatus = "idle"
	BufferStatusProcessing BufferStatus = "processing"
	BufferStatusDone       BufferStatus = "done"
	BufferStatusFailed     BufferStatus = "failed"
)

// BufferZone is the pending-flush queue entry linking a Blob to processing.
type BufferZone struct {
	ID        uuid.UUID
	ProjectID string
	UserID    uuid.UUID
	BlobID    uuid.UUID
	BlobType  BlobType
	TokenSize int
	Status    BufferStatus
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CanFlush returns true when this buffer entry is eligible for flushing.
func (bz *BufferZone) CanFlush() bool {
	return bz.Status == BufferStatusIdle
}

package model

import (
	"time"

	"github.com/google/uuid"
)

// BlobType represents the type of blob content.
type BlobType string

const (
	BlobTypeChat    BlobType = "chat"
	BlobTypeDoc     BlobType = "doc"
	BlobTypeSummary BlobType = "summary"
)

// Blob represents unstructured data input that the engine processes.
type Blob struct {
	ID        uuid.UUID
	UserID    string
	ProjectID string
	Type      BlobType
	Content   string
	CreatedAt time.Time
}

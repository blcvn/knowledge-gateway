// Package document defines domain entities for the Supermemory document sub-domain.
package document

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a stored content document.
type Document struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	UserID      uuid.UUID `json:"user_id"`
	Title       string    `json:"title"`
	URL         string    `json:"url,omitempty"`
	ContentType string    `json:"content_type"` // article, page, note
	RawContent  string    `json:"raw_content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Chunk represents a semantic chunk of a document.
type Chunk struct {
	ID         uuid.UUID `json:"id"`
	DocumentID uuid.UUID `json:"document_id"`
	TenantID   uuid.UUID `json:"tenant_id"`
	Content    string    `json:"content"`
	Index      int       `json:"index"`  // Position within document
	Tokens     int       `json:"tokens"` // Token count
}

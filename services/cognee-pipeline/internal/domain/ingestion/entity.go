// Package ingestion defines domain entities for the Cognee ingestion sub-domain.
package ingestion

import (
	"time"

	"github.com/google/uuid"
)

// Dataset represents a collection of data items for cognification.
type Dataset struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"` // active, archived
	ItemCount   int       `json:"item_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DataItem represents a single piece of ingested data.
type DataItem struct {
	ID         uuid.UUID `json:"id"`
	DatasetID  uuid.UUID `json:"dataset_id"`
	SourceType string    `json:"source_type"` // file, text, url
	SourceURI  string    `json:"source_uri"`
	MimeType   string    `json:"mime_type"`
	SizeBytes  int64     `json:"size_bytes"`
	Checksum   string    `json:"checksum"` // SHA-256
	CreatedAt  time.Time `json:"created_at"`
}

// DataSource enumerates supported ingestion sources.
type DataSource string

const (
	SourceFile DataSource = "file"
	SourceText DataSource = "text"
	SourceURL  DataSource = "url"
)

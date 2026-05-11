// Package dto defines request/response data transfer objects for the usecase layer.
package dto

import (
	"io"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

// IngestFileRequest contains the inputs for file ingestion.
type IngestFileRequest struct {
	DatasetID uuid.UUID
	TenantID  string
	Filename  string
	MimeType  domain.MimeType
	Reader    io.Reader
	Size      int64
	Metadata  map[string]string
}

// IngestTextRequest contains the inputs for direct text ingestion.
type IngestTextRequest struct {
	DatasetID  uuid.UUID
	TenantID   string
	Text       string
	SourceName string
	Metadata   map[string]string
}

// IngestURLRequest contains the inputs for URL scraping and ingestion.
type IngestURLRequest struct {
	DatasetID uuid.UUID
	TenantID  string
	URL       string
	Metadata  map[string]string
}

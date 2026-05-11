package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

// IngestResult contains the output of a successful ingestion operation.
type IngestResult struct {
	ItemID      uuid.UUID `json:"item_id"`
	DatasetID   uuid.UUID `json:"dataset_id"`
	Source      string    `json:"source"`
	Filename    string    `json:"filename,omitempty"`
	SizeBytes   int64     `json:"size_bytes"`
	TextPreview string    `json:"text_preview,omitempty"` // First 200 chars
	IsDuplicate bool      `json:"is_duplicate"`
}

// DatasetStatusResponse contains dataset status with item statistics.
type DatasetStatusResponse struct {
	ID             uuid.UUID           `json:"id"`
	Name           string              `json:"name"`
	Status         domain.DatasetStatus `json:"status"`
	FileCount      int                 `json:"file_count"`
	TotalSizeBytes int64               `json:"total_size_bytes"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

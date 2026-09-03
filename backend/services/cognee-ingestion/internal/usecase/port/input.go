// Package port defines the input port interfaces for the cognee-ingestion usecase layer.
//
// Input ports define *what* the service can do — they are called by adapters
// (gRPC handlers, NATS subscribers) and implemented by usecase structs.
package port

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
)

// FileIngester handles file upload, text extraction, and persistence.
type FileIngester interface {
	Execute(ctx context.Context, req dto.IngestFileRequest) (*dto.IngestResult, error)
}

// TextIngester handles direct text ingestion.
type TextIngester interface {
	Execute(ctx context.Context, req dto.IngestTextRequest) (*dto.IngestResult, error)
}

// URLIngester handles URL scraping and content ingestion.
type URLIngester interface {
	Execute(ctx context.Context, req dto.IngestURLRequest) (*dto.IngestResult, error)
}

// DatasetManager handles dataset lifecycle operations.
type DatasetManager interface {
	Create(ctx context.Context, tenantID, name, description string) (*domain.Dataset, error)
	Get(ctx context.Context, tenantID string, id uuid.UUID) (*domain.Dataset, error)
	List(ctx context.Context, tenantID, cursor string, limit int) ([]*domain.Dataset, string, error)
	Delete(ctx context.Context, tenantID string, id uuid.UUID) error
	GetStatus(ctx context.Context, tenantID string, id uuid.UUID) (*dto.DatasetStatusResponse, error)
}

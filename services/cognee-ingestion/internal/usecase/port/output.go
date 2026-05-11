package port

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

// DatasetRepository persists and retrieves datasets.
type DatasetRepository interface {
	Create(ctx context.Context, ds *domain.Dataset) error
	GetByID(ctx context.Context, tenantID string, id uuid.UUID) (*domain.Dataset, error)
	List(ctx context.Context, tenantID string, cursor string, limit int) ([]*domain.Dataset, string, error)
	Delete(ctx context.Context, tenantID string, id uuid.UUID) error
	UpdateStatus(ctx context.Context, tenantID string, id uuid.UUID, status domain.DatasetStatus) error
	IncrementItems(ctx context.Context, tenantID string, id uuid.UUID, sizeBytes int64) error
	ExistsByName(ctx context.Context, tenantID, name string) (bool, error)
}

// DataItemRepository persists and retrieves data items.
type DataItemRepository interface {
	Create(ctx context.Context, item *domain.DataItem) error
	ListByDataset(ctx context.Context, datasetID uuid.UUID) ([]*domain.DataItem, error)
	DeleteByDataset(ctx context.Context, datasetID uuid.UUID) error
	ExistsByHash(ctx context.Context, datasetID uuid.UUID, fileHash string) (bool, error)
}

// FileStorage abstracts object storage (MinIO/S3) for raw file persistence.
type FileStorage interface {
	// Upload stores the contents of reader at the given key and returns the storage path.
	Upload(ctx context.Context, key string, reader io.Reader, size int64) (storagePath string, err error)
	// Delete removes the object at the given key.
	Delete(ctx context.Context, key string) error
	// DeletePrefix removes all objects matching the prefix.
	DeletePrefix(ctx context.Context, prefix string) error
}

// TextExtractor extracts plaintext from various file formats.
type TextExtractor interface {
	// Extract reads from the given reader and returns extracted text.
	Extract(ctx context.Context, reader io.Reader, mimeType domain.MimeType) (string, error)
	// Supported returns the list of MIME types this extractor can handle.
	Supported() []domain.MimeType
}

// URLScraper extracts text content from a web URL.
type URLScraper interface {
	// Scrape fetches the URL and returns the extracted text content.
	Scrape(ctx context.Context, url string) (string, error)
}

// EventPublisher publishes domain events to the message bus (NATS).
type EventPublisher interface {
	// PublishDataIngested publishes a DataIngestedEvent for downstream consumers (cognee-cognify).
	PublishDataIngested(ctx context.Context, event domain.DataIngestedEvent) error
}

// HashComputer computes cryptographic hashes for deduplication.
type HashComputer interface {
	// ComputeSHA256 returns the hex-encoded SHA-256 hash of the reader contents.
	// It also returns a new reader that replays the same bytes (since the original is consumed).
	ComputeSHA256(reader io.Reader) (hash string, replayed io.Reader, err error)
}

package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for the ingestion domain.
var (
	// Validation errors.
	ErrMissingTenantID    = errors.New("tenant_id is required")
	ErrMissingDatasetName = errors.New("dataset name is required")
	ErrMissingDatasetID   = errors.New("dataset_id is required")
	ErrEmptyText          = errors.New("text content is empty")
	ErrEmptyURL           = errors.New("url is empty")

	// Business rule errors.
	ErrDuplicateDataset = errors.New("dataset with this name already exists for tenant")
	ErrDatasetNotFound  = errors.New("dataset not found")
	ErrDataItemNotFound = errors.New("data item not found")
	ErrDuplicateFile    = errors.New("file with identical hash already exists in dataset")
)

// ErrUnsupportedFormat indicates the MIME type is not supported for extraction.
type ErrUnsupportedFormat struct {
	MimeType string
}

func (e *ErrUnsupportedFormat) Error() string {
	return fmt.Sprintf("unsupported format: %s", e.MimeType)
}

// ErrExtractionFailed indicates text extraction from a file failed.
type ErrExtractionFailed struct {
	Filename string
	Cause    error
}

func (e *ErrExtractionFailed) Error() string {
	return fmt.Sprintf("text extraction failed for %s: %v", e.Filename, e.Cause)
}

func (e *ErrExtractionFailed) Unwrap() error { return e.Cause }

// ErrScrapeFailed indicates URL scraping failed.
type ErrScrapeFailed struct {
	URL   string
	Cause error
}

func (e *ErrScrapeFailed) Error() string {
	return fmt.Sprintf("scrape failed for %s: %v", e.URL, e.Cause)
}

func (e *ErrScrapeFailed) Unwrap() error { return e.Cause }

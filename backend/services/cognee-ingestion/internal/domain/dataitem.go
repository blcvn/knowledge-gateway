// Package domain defines DataItem — the core ingestion entity for file/text/URL data.
package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DataItem represents a piece of content being ingested into the memory system.
type DataItem struct {
	ID        uuid.UUID
	DatasetID uuid.UUID
	TenantID  string
	Source    DataSource
	// File fields (populated by WithFile)
	Filename  string
	MimeType  MimeType
	SizeBytes int64
	Checksum  string
	LocalPath string
	// Text fields (populated by WithText)
	TextContent string
	// URL fields
	URL      string
	NodeSets    []string
	Metadata    map[string]any
	// Legacy aliases used by postgres adapter
	RawText     string    // alias for TextContent
	StoragePath string    // alias for LocalPath  
	FileHash    string    // alias for Checksum
	// Timestamps
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewDataItem creates a DataItem with validated fields.
func NewDataItem(datasetID uuid.UUID, tenantID string, source DataSource) (*DataItem, error) {
	if datasetID == uuid.Nil {
		return nil, ErrMissingDatasetID
	}
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	return &DataItem{
		ID:        uuid.New(),
		DatasetID: datasetID,
		TenantID:  tenantID,
		Source:    source,
		Metadata:  make(map[string]any),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// StorageKey returns the MinIO/S3 object key for this item.
func (d *DataItem) StorageKey() string {
	return fmt.Sprintf("%s/%s/%s", d.TenantID, d.DatasetID, d.ID)
}

// WithFile populates the file-specific fields of this DataItem.
func (d *DataItem) WithFile(filename string, mimeType MimeType, sizeBytes int64, checksum, localPath string) *DataItem {
	d.Filename = filename
	d.MimeType = mimeType
	d.SizeBytes = sizeBytes
	d.Checksum = checksum
	d.LocalPath = localPath
	return d
}

// WithText populates the text content of this DataItem.
func (d *DataItem) WithText(text string, source ...string) *DataItem {
	d.TextContent = text
	d.RawText = text
	if len(source) > 0 {
		d.URL = source[0] // repurpose URL for source name
	}
	return d
}

// WithURL populates the URL field and optionally stores extracted text.
func (d *DataItem) WithURL(url string, extractedText ...string) *DataItem {
	d.URL = url
	if len(extractedText) > 0 {
		d.TextContent = extractedText[0]
		d.RawText = extractedText[0]
	}
	return d
}

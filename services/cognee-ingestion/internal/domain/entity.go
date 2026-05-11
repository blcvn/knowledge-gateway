package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Dataset represents a named collection of data items belonging to a tenant.
// It is the primary organisational unit for ingested data.
type Dataset struct {
	ID             uuid.UUID         `json:"id"`
	TenantID       string            `json:"tenant_id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Status         DatasetStatus     `json:"status"`
	FileCount      int               `json:"file_count"`
	TotalSizeBytes int64             `json:"total_size_bytes"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// NewDataset constructs a Dataset with sensible defaults.
func NewDataset(tenantID, name, description string) (*Dataset, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if name == "" {
		return nil, ErrMissingDatasetName
	}

	now := time.Now().UTC()
	return &Dataset{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           name,
		Description:    description,
		Status:         DatasetPending,
		FileCount:      0,
		TotalSizeBytes: 0,
		Metadata:       make(map[string]string),
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// MarkReady transitions the dataset to READY status.
func (d *Dataset) MarkReady() {
	d.Status = DatasetReady
	d.UpdatedAt = time.Now().UTC()
}

// MarkCognifying transitions the dataset to COGNIFYING status.
func (d *Dataset) MarkCognifying() {
	d.Status = DatasetCognifying
	d.UpdatedAt = time.Now().UTC()
}

// MarkError transitions the dataset to ERROR status.
func (d *Dataset) MarkError() {
	d.Status = DatasetError
	d.UpdatedAt = time.Now().UTC()
}

// IncrementItems records the addition of a data item.
func (d *Dataset) IncrementItems(sizeBytes int64) {
	d.FileCount++
	d.TotalSizeBytes += sizeBytes
	d.UpdatedAt = time.Now().UTC()
}

// DataItem represents a single piece of ingested content within a dataset.
type DataItem struct {
	ID          uuid.UUID         `json:"id"`
	DatasetID   uuid.UUID         `json:"dataset_id"`
	TenantID    string            `json:"tenant_id"`
	Source      DataSource        `json:"source"`
	Filename    string            `json:"filename,omitempty"`
	MimeType    MimeType          `json:"mime_type,omitempty"`
	RawText     string            `json:"raw_text,omitempty"`
	StoragePath string            `json:"storage_path,omitempty"`
	SizeBytes   int64             `json:"size_bytes"`
	FileHash    string            `json:"file_hash,omitempty"` // SHA-256
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// NewDataItem constructs a DataItem with the given source type.
func NewDataItem(datasetID uuid.UUID, tenantID string, source DataSource) (*DataItem, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if datasetID == uuid.Nil {
		return nil, ErrMissingDatasetID
	}

	return &DataItem{
		ID:        uuid.New(),
		DatasetID: datasetID,
		TenantID:  tenantID,
		Source:    source,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// WithFile sets file-specific fields on the data item.
func (di *DataItem) WithFile(filename string, mimeType MimeType, sizeBytes int64, fileHash, storagePath string) {
	di.Filename = filename
	di.MimeType = mimeType
	di.SizeBytes = sizeBytes
	di.FileHash = fileHash
	di.StoragePath = storagePath
}

// WithText sets text-specific fields on the data item.
func (di *DataItem) WithText(text, sourceName string) {
	di.RawText = text
	di.Filename = sourceName
	di.SizeBytes = int64(len(text))
}

// WithURL sets URL-specific fields on the data item.
func (di *DataItem) WithURL(url, extractedText string) {
	di.Filename = url
	di.RawText = extractedText
	di.SizeBytes = int64(len(extractedText))
	di.Source = SourceURL
}

// StorageKey returns the MinIO/S3 object key for this item.
func (di *DataItem) StorageKey() string {
	return fmt.Sprintf("%s/%s/%s", di.TenantID, di.DatasetID.String(), di.ID.String())
}

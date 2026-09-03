package domain

import (
	"time"

	"github.com/google/uuid"
)

// ContentType enumerates supported content types.
type ContentType string

const (
	ContentTypeText      ContentType = "TEXT"
	ContentTypePDF       ContentType = "PDF"
	ContentTypePDFLayout ContentType = "PDF_LAYOUT"
	ContentTypeHTML      ContentType = "HTML"
	ContentTypeURL       ContentType = "URL"
	ContentTypeDOCX      ContentType = "DOCX"
	ContentTypeCSV       ContentType = "CSV"
	ContentTypeTabularFK ContentType = "TABULAR_FK"
)

// Dataset represents a named container for data entries.
type Dataset struct {
	ID             uuid.UUID
	TenantID       string
	Name           string
	Description    string
	Status         DatasetStatus
	FileCount      int
	TotalSizeBytes int64
	Metadata       map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DataEntry represents a single piece of ingested content.
type DataEntry struct {
	ID        uuid.UUID
	DatasetID uuid.UUID
	TenantID  string
	Content   string
	Type      ContentType    // TEXT | PDF | PDF_LAYOUT | HTML | URL | DOCX | CSV | TABULAR_FK
	URL       string
	Metadata  map[string]any
	NodeSets  []string       // [NEW] CR-002 — e.g. ["customer_123", "preferences", "contracts"]
	CreatedAt time.Time
}

// GraphNode represents a node in the knowledge graph (used by add_data_points usecase).
type GraphNode struct {
	ID         string
	Name       string
	Type       string
	Labels     []string
	Properties map[string]any
}

// GraphEdge represents a directed relationship between two nodes.
type GraphEdge struct {
	ID         string
	SourceID   string
	TargetID   string
	Label      string
	Weight     float64
	Properties map[string]any
	// Memify aliases
	Subject   string
	Object    string
	Predicate string
}

// NewDataset creates a new Dataset with validated fields.
func NewDataset(tenantID, name, description string) (*Dataset, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if name == "" {
		return nil, ErrMissingDatasetID // reuse error
	}
	return &Dataset{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		Status:      DatasetPending,
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}, nil
}

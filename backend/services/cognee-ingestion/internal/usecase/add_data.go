package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"vnp-memory/services/cognee-ingestion/internal/domain"
)

// ─── Request / Response types ─────────────────────────────────────────────────

// AddDataRequest is the input to AddDataUseCase.Execute().
type AddDataRequest struct {
	DatasetID   uuid.UUID
	DatasetName string
	TenantID    string
	Items       []DataItem
	NodeSets    []string // [NEW] CR-002
}

// DataItem represents a single piece of content to ingest.
type DataItem struct {
	Content     string
	URL         string
	ContentType domain.ContentType
	Metadata    map[string]any
	Config      *DataItemConfig
}

// DataItemConfig holds per-item ingestion settings.
type DataItemConfig struct {
	PDFMode string // "LAYOUT_AWARE" | "PLAIN_TEXT"
}

// AddDataResult is the output of AddDataUseCase.Execute().
type AddDataResult struct {
	DatasetID string
	EntryIDs  []string
	Count     int
}

// ─── Ports ────────────────────────────────────────────────────────────────────

// DatasetRepository provides dataset storage operations.
type DatasetRepository interface {
	GetOrCreate(ctx context.Context, id uuid.UUID, name, tenantID string) (*domain.Dataset, error)
}

// DataEntryRepository provides data entry storage operations.
type DataEntryRepository interface {
	SaveBulk(ctx context.Context, entries []domain.DataEntry) error
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	PublishDataIngested(ctx context.Context, evt DataIngestedEvent)
}

// ─── Use Case ─────────────────────────────────────────────────────────────────

// AddDataUseCase handles ingestion of raw data into datasets.
type AddDataUseCase struct {
	datasetRepo    DatasetRepository
	dataEntryRepo  DataEntryRepository
	publisher      EventPublisher
}

// NewAddDataUseCase creates a new AddDataUseCase.
func NewAddDataUseCase(
	datasetRepo DatasetRepository,
	dataEntryRepo DataEntryRepository,
	publisher EventPublisher,
) *AddDataUseCase {
	return &AddDataUseCase{
		datasetRepo:   datasetRepo,
		dataEntryRepo: dataEntryRepo,
		publisher:     publisher,
	}
}

// Execute runs the ingestion pipeline: validate → get/create dataset → build entries → persist → emit.
func (uc *AddDataUseCase) Execute(ctx context.Context, req AddDataRequest) (*AddDataResult, error) {
	dataset, err := uc.datasetRepo.GetOrCreate(ctx, req.DatasetID, req.DatasetName, req.TenantID)
	if err != nil { return nil, fmt.Errorf("get or create dataset: %w", err) }

	var entries []domain.DataEntry
	for _, item := range req.Items {
		entry := domain.DataEntry{
			ID:        uuid.New(),
			DatasetID: dataset.ID,
			TenantID:  req.TenantID,
			Content:   item.Content,
			Type:      item.ContentType,
			URL:       item.URL,
			Metadata:  item.Metadata,
			NodeSets:  req.NodeSets, // [ADDED] attach node_sets to every entry
			CreatedAt: time.Now(),
		}
		entries = append(entries, entry)
	}

	// Persist entries (NodeSets stored as JSON column)
	if err := uc.dataEntryRepo.SaveBulk(ctx, entries); err != nil {
		return nil, fmt.Errorf("save entries: %w", err)
	}

	entryIDs := extractEntryIDs(entries)

	// Emit NATS event — include node_sets for downstream cognify
	uc.publisher.PublishDataIngested(ctx, DataIngestedEvent{
		DatasetID: dataset.ID.String(),
		TenantID:  req.TenantID,
		EntryIDs:  entryIDs,
		NodeSets:  req.NodeSets, // [ADDED] propagate to cognify
	})

	return &AddDataResult{
		DatasetID: dataset.ID.String(),
		EntryIDs:  entryIDs,
		Count:     len(entries),
	}, nil
}

func extractEntryIDs(entries []domain.DataEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID.String()
	}
	return ids
}

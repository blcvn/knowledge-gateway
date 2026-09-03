package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"vnp-memory/services/cognee-ingestion/internal/usecase/port"
)

// IngestTextUseCase handles direct text ingestion.
type IngestTextUseCase struct {
	datasetRepo port.DatasetRepository
	itemRepo    port.DataItemRepository
	eventPub    port.EventPublisher
	logger      *slog.Logger
}

// NewIngestTextUseCase constructs the text ingestion use case.
func NewIngestTextUseCase(
	datasetRepo port.DatasetRepository,
	itemRepo port.DataItemRepository,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *IngestTextUseCase {
	return &IngestTextUseCase{
		datasetRepo: datasetRepo,
		itemRepo:    itemRepo,
		eventPub:    eventPub,
		logger:      logger.With("usecase", "ingest_text"),
	}
}

// Execute stores text content as a DataItem and publishes an ingestion event.
func (uc *IngestTextUseCase) Execute(ctx context.Context, req dto.IngestTextRequest) (*dto.IngestResult, error) {
	log := uc.logger.With("dataset_id", req.DatasetID)

	// Validate
	if req.Text == "" {
		return nil, domain.ErrEmptyText
	}

	ds, err := uc.datasetRepo.GetByID(ctx, req.TenantID, req.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return nil, domain.ErrDatasetNotFound
	}

	// Create DataItem
	item, err := domain.NewDataItem(req.DatasetID, req.TenantID, domain.SourceText)
	if err != nil {
		return nil, fmt.Errorf("create data item: %w", err)
	}

	sourceName := req.SourceName
	if sourceName == "" {
		sourceName = "text-input"
	}
	item.WithText(req.Text, sourceName)
	item.MimeType = domain.MimePlainText
	func() {
		for k, v := range req.Metadata {
			item.Metadata[k] = v
		}
	}()

	if err := uc.itemRepo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("persist data item: %w", err)
	}

	// Update dataset counters
	if err := uc.datasetRepo.IncrementItems(ctx, req.TenantID, req.DatasetID, item.SizeBytes); err != nil {
		log.Error("failed to update dataset counters", "error", err)
	}

	// Publish event
	event := domain.NewDataIngestedEvent(req.DatasetID, req.TenantID, []string{item.ID.String()})
	if err := uc.eventPub.PublishDataIngested(ctx, event); err != nil {
		log.Error("failed to publish ingested event", "error", err)
	}

	log.Info("text ingested successfully", "item_id", item.ID, "size", item.SizeBytes)

	return &dto.IngestResult{
		ItemID:      item.ID,
		DatasetID:   req.DatasetID,
		Source:      string(domain.SourceText),
		Filename:    sourceName,
		SizeBytes:   item.SizeBytes,
		TextPreview: truncate(req.Text, textPreviewMaxLen),
		IsDuplicate: false,
	}, nil
}

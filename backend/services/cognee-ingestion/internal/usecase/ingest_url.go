package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"vnp-memory/services/cognee-ingestion/internal/usecase/port"
)

// IngestURLUseCase handles URL scraping and text ingestion.
type IngestURLUseCase struct {
	datasetRepo port.DatasetRepository
	itemRepo    port.DataItemRepository
	scraper     port.URLScraper
	eventPub    port.EventPublisher
	logger      *slog.Logger
}

// NewIngestURLUseCase constructs the URL ingestion use case.
func NewIngestURLUseCase(
	datasetRepo port.DatasetRepository,
	itemRepo port.DataItemRepository,
	scraper port.URLScraper,
	eventPub port.EventPublisher,
	logger *slog.Logger,
) *IngestURLUseCase {
	return &IngestURLUseCase{
		datasetRepo: datasetRepo,
		itemRepo:    itemRepo,
		scraper:     scraper,
		eventPub:    eventPub,
		logger:      logger.With("usecase", "ingest_url"),
	}
}

// Execute scrapes the URL, extracts text, and stores it as a DataItem.
func (uc *IngestURLUseCase) Execute(ctx context.Context, req dto.IngestURLRequest) (*dto.IngestResult, error) {
	log := uc.logger.With("dataset_id", req.DatasetID, "url", req.URL)

	// Validate
	if req.URL == "" {
		return nil, domain.ErrEmptyURL
	}

	ds, err := uc.datasetRepo.GetByID(ctx, req.TenantID, req.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return nil, domain.ErrDatasetNotFound
	}

	// Scrape URL
	extractedText, err := uc.scraper.Scrape(ctx, req.URL)
	if err != nil {
		return nil, &domain.ErrScrapeFailed{URL: req.URL, Cause: err}
	}

	if extractedText == "" {
		return nil, &domain.ErrScrapeFailed{URL: req.URL, Cause: fmt.Errorf("no text content extracted")}
	}

	// Create DataItem
	item, err := domain.NewDataItem(req.DatasetID, req.TenantID, domain.SourceURL)
	if err != nil {
		return nil, fmt.Errorf("create data item: %w", err)
	}

	item.WithURL(req.URL, extractedText)
	item.MimeType = domain.MimeHTML
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

	log.Info("url ingested successfully", "item_id", item.ID, "size", item.SizeBytes)

	return &dto.IngestResult{
		ItemID:      item.ID,
		DatasetID:   req.DatasetID,
		Source:      string(domain.SourceURL),
		Filename:    req.URL,
		SizeBytes:   item.SizeBytes,
		TextPreview: truncate(extractedText, textPreviewMaxLen),
		IsDuplicate: false,
	}, nil
}

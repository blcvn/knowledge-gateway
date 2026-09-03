package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"vnp-memory/services/cognee-ingestion/internal/domain"
	"vnp-memory/services/cognee-ingestion/internal/usecase/dto"
	"vnp-memory/services/cognee-ingestion/internal/usecase/port"
)

const textPreviewMaxLen = 200

// IngestFileUseCase handles file upload, text extraction, dedup, and event publishing.
type IngestFileUseCase struct {
	datasetRepo  port.DatasetRepository
	itemRepo     port.DataItemRepository
	fileStorage  port.FileStorage
	extractor    port.TextExtractor
	eventPub     port.EventPublisher
	hashComputer port.HashComputer
	logger       *slog.Logger
}

// NewIngestFileUseCase constructs the file ingestion use case.
func NewIngestFileUseCase(
	datasetRepo port.DatasetRepository,
	itemRepo port.DataItemRepository,
	fileStorage port.FileStorage,
	extractor port.TextExtractor,
	eventPub port.EventPublisher,
	hashComputer port.HashComputer,
	logger *slog.Logger,
) *IngestFileUseCase {
	return &IngestFileUseCase{
		datasetRepo:  datasetRepo,
		itemRepo:     itemRepo,
		fileStorage:  fileStorage,
		extractor:    extractor,
		eventPub:     eventPub,
		hashComputer: hashComputer,
		logger:       logger.With("usecase", "ingest_file"),
	}
}

// Execute performs the file ingestion pipeline:
//
//  1. Validate: dataset exists, MIME type supported
//  2. Compute file hash (SHA-256) for dedup check
//  3. Check for duplicate (same hash in same dataset)
//  4. Upload raw file to S3/MinIO
//  5. Extract text content
//  6. Create DataItem record in DB
//  7. Update dataset counters
//  8. Publish DataIngestedEvent
func (uc *IngestFileUseCase) Execute(ctx context.Context, req dto.IngestFileRequest) (*dto.IngestResult, error) {
	log := uc.logger.With("dataset_id", req.DatasetID, "filename", req.Filename)

	// 1. Validate dataset exists
	ds, err := uc.datasetRepo.GetByID(ctx, req.TenantID, req.DatasetID)
	if err != nil {
		return nil, fmt.Errorf("get dataset: %w", err)
	}
	if ds == nil {
		return nil, domain.ErrDatasetNotFound
	}

	// 1b. Validate MIME type
	if !req.MimeType.IsSupported() {
		return nil, &domain.ErrUnsupportedFormat{MimeType: string(req.MimeType)}
	}

	// 2. Compute file hash for dedup
	hash, replayReader, err := uc.hashComputer.ComputeSHA256(req.Reader)
	if err != nil {
		return nil, fmt.Errorf("compute hash: %w", err)
	}

	// 3. Check for duplicate
	exists, err := uc.itemRepo.ExistsByHash(ctx, req.DatasetID, hash)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}
	if exists {
		log.Info("duplicate file detected, skipping", "hash", hash)
		return &dto.IngestResult{
			DatasetID:   req.DatasetID,
			Source:      string(domain.SourceFile),
			Filename:    req.Filename,
			SizeBytes:   req.Size,
			IsDuplicate: true,
		}, nil
	}

	// 4. Upload raw file to storage
	item, err := domain.NewDataItem(req.DatasetID, req.TenantID, domain.SourceFile)
	if err != nil {
		return nil, fmt.Errorf("create data item: %w", err)
	}

	storagePath, err := uc.fileStorage.Upload(ctx, item.StorageKey(), replayReader, req.Size)
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	// 5. Extract text content
	// Re-read from storage or use a buffered copy. For simplicity, we extract
	// during the same request. In production, consider streaming.
	extractedText, err := uc.extractor.Extract(ctx, replayReader, req.MimeType)
	if err != nil {
		log.Warn("text extraction failed, storing without text", "error", err)
		extractedText = ""
	}

	// 6. Set item fields and persist
	item.WithFile(req.Filename, req.MimeType, req.Size, hash, storagePath)
	item.RawText = extractedText
	func() {
		for k, v := range req.Metadata {
			item.Metadata[k] = v
		}
	}()

	if err := uc.itemRepo.Create(ctx, item); err != nil {
		return nil, fmt.Errorf("persist data item: %w", err)
	}

	// 7. Update dataset counters
	if err := uc.datasetRepo.IncrementItems(ctx, req.TenantID, req.DatasetID, req.Size); err != nil {
		log.Error("failed to update dataset counters", "error", err)
	}

	// 8. Publish event
	event := domain.NewDataIngestedEvent(req.DatasetID, req.TenantID, []string{item.ID.String()})
	if err := uc.eventPub.PublishDataIngested(ctx, event); err != nil {
		log.Error("failed to publish ingested event", "error", err)
	}

	log.Info("file ingested successfully", "item_id", item.ID, "size", req.Size)

	return &dto.IngestResult{
		ItemID:      item.ID,
		DatasetID:   req.DatasetID,
		Source:      string(domain.SourceFile),
		Filename:    req.Filename,
		SizeBytes:   req.Size,
		TextPreview: truncate(extractedText, textPreviewMaxLen),
		IsDuplicate: false,
	}, nil
}

// truncate returns the first n runes of s.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

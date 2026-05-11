package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"openviking.com/ov-resource/internal/domain"
	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/domain/repository"
	"openviking.com/ov-resource/internal/usecase/dto"
	"openviking.com/ov-resource/internal/usecase/port"
)

type ingestUseCase struct {
	parserRegistry port.ParserPort
	fileWriter     port.FileWriterPort
	eventPublisher port.EventPublisherPort
	resourceRepo   repository.ResourceRepository
	maxSizeMb      int
}

func NewIngestUseCase(
	parserRegistry port.ParserPort,
	fileWriter port.FileWriterPort,
	eventPublisher port.EventPublisherPort,
	resourceRepo repository.ResourceRepository,
	maxSizeMb int,
) *ingestUseCase {
	return &ingestUseCase{
		parserRegistry: parserRegistry,
		fileWriter:     fileWriter,
		eventPublisher: eventPublisher,
		resourceRepo:   resourceRepo,
		maxSizeMb:      maxSizeMb,
	}
}

func (u *ingestUseCase) Execute(ctx context.Context, req dto.IngestRequest) (dto.IngestResponse, error) {
	start := time.Now()

	if len(req.Content) > u.maxSizeMb*1024*1024 {
		return dto.IngestResponse{}, domain.ErrResourceExhausted
	}

	hash := sha256.Sum256(req.Content)
	contentHash := hex.EncodeToString(hash[:])

	existing, _ := u.resourceRepo.GetByHash(ctx, req.AccountID, contentHash)
	if existing != nil && existing.Status == model.StatusCompleted {
		return dto.IngestResponse{
			ChunksCount:     existing.ChunkCount,
			TotalTokens:     existing.TotalTokens,
			Path:            existing.TargetPath,
			ParseDurationMs: existing.ParseDurationMs,
		}, nil
	}

	resource := &model.Resource{
		ID:          uuid.New().String(),
		AccountID:   req.AccountID,
		Filename:    req.Filename,
		TargetPath:  req.Path,
		ContentHash: contentHash,
		Status:      model.StatusProcessing,
	}
	_ = u.resourceRepo.Create(ctx, resource)

	chunks, err := u.parserRegistry.Parse(ctx, req.Content, req.Filename, model.ParserConfig{})
	if err != nil {
		_ = u.resourceRepo.UpdateStatus(ctx, resource.ID, req.AccountID, model.StatusFailed, err.Error())
		return dto.IngestResponse{}, domain.ErrParseFailed
	}

	totalTokens := 0
	for _, chunk := range chunks {
		totalTokens += chunk.Metadata.TotalTokens
	}

	err = u.fileWriter.WriteChunks(ctx, req.Path, req.AccountID, chunks)
	if err != nil {
		_ = u.resourceRepo.UpdateStatus(ctx, resource.ID, req.AccountID, model.StatusFailed, err.Error())
		return dto.IngestResponse{}, domain.ErrIngestFailed
	}

	resource.ChunkCount = len(chunks)
	resource.TotalTokens = totalTokens
	resource.Status = model.StatusCompleted
	resource.ParseDurationMs = int(time.Since(start).Milliseconds())
	resource.IngestedAt = time.Now()
	_ = u.resourceRepo.Update(ctx, resource)

	event := domain.ResourceIngested{
		Path:       req.Path,
		AccountID:  req.AccountID,
		Chunks:     len(chunks),
		ParserType: "auto",
	}
	_ = u.eventPublisher.PublishResourceIngested(ctx, event)

	return dto.IngestResponse{
		ChunksCount:     len(chunks),
		TotalTokens:     totalTokens,
		Path:            req.Path,
		ParseDurationMs: resource.ParseDurationMs,
	}, nil
}

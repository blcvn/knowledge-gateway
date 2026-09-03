package usecase

import (
	"context"
	"time"

	"github.com/vnp-memory/services/ov-session/adapter/client"
	"github.com/vnp-memory/services/ov-session/adapter/event"
	"github.com/vnp-memory/services/ov-session/domain"
	"github.com/vnp-memory/services/ov-session/domain/model"
	"github.com/vnp-memory/services/ov-session/domain/repository"
)

type CommitUseCase interface {
	CommitSession(ctx context.Context, sessionID string, extractMemories bool, version model.CompressionVersion) (*model.SessionCompression, error)
}

type commitUseCaseImpl struct {
	sessionRepo     repository.SessionRepository
	messageRepo     repository.MessageRepository
	compressor      CompressorUseCase
	extractor       MemoryExtractorUseCase
	deduplicator    MemoryDeduplicatorUseCase
	fsClient        client.FSClient
	publisher       event.Publisher
}

func NewCommitUseCase(
	sRepo repository.SessionRepository,
	mRepo repository.MessageRepository,
	comp CompressorUseCase,
	ext MemoryExtractorUseCase,
	dedup MemoryDeduplicatorUseCase,
	fs client.FSClient,
	pub event.Publisher,
) CommitUseCase {
	return &commitUseCaseImpl{
		sessionRepo:  sRepo,
		messageRepo:  mRepo,
		compressor:   comp,
		extractor:    ext,
		deduplicator: dedup,
		fsClient:     fs,
		publisher:    pub,
	}
}

func (uc *commitUseCaseImpl) CommitSession(ctx context.Context, sessionID string, extractMemories bool, version model.CompressionVersion) (*model.SessionCompression, error) {
	// PHASE 1: Archive
	session, err := uc.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != model.SessionStatusActive {
		return nil, domain.ErrAlreadyCommitted
	}

	messages, err := uc.messageRepo.GetMessagesBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	archiveContent, err := uc.compressor.Compress(ctx, messages, version)
	if err != nil {
		return nil, err
	}

	archivePath, err := uc.fsClient.WriteArchive(ctx, session.AccountID, session.UserID, archiveContent)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session.Status = model.SessionStatusCommitted
	session.ArchivePath = archivePath
	session.CommittedAt = &now
	session.CompressionVersion = string(version)

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	uc.publisher.PublishSessionCommitted(ctx, &domain.SessionCommitted{
		SessionID:   session.ID,
		AccountID:   session.AccountID,
		ArchivePath: archivePath,
	})

	stats := model.ExtractionStats{ByCategory: make(map[string]int)}

	// PHASE 2: Extract & Deduplicate
	if extractMemories {
		candidates, err := uc.extractor.Extract(ctx, archiveContent)
		if err == nil {
			finalMemories, _ := uc.deduplicator.Deduplicate(ctx, candidates)
			
			var fsPaths []string
			for i := range finalMemories {
				m := &finalMemories[i]
				m.SessionID = session.ID
				m.AccountID = session.AccountID
				
				path, _ := uc.fsClient.WriteMemory(ctx, m.AccountID, string(m.Category), m.Content)
				m.FSPath = path
				fsPaths = append(fsPaths, path)
				
				uc.sessionRepo.SaveMemory(ctx, m)
				stats.ByCategory[string(m.Category)]++
			}
			stats.TotalExtracted = len(finalMemories)
			session.MemoriesCount = len(finalMemories)
			uc.sessionRepo.Update(ctx, session)

			if len(finalMemories) > 0 {
				uc.publisher.PublishMemoryExtracted(ctx, &domain.MemoryExtracted{
					SessionID: session.ID,
					Memories:  finalMemories,
					FSPaths:   fsPaths,
				})
			}
		}
	}

	return &model.SessionCompression{
		ArchivePath:        archivePath,
		CompressionVersion: version,
		MemoriesCount:      session.MemoriesCount,
		ExtractionStats:    stats,
	}, nil
}

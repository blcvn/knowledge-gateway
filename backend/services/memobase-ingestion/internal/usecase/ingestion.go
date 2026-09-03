package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/pkg/tokenizer"
	"github.com/vnp-memory/services/memobase-ingestion/internal/domain"
	"github.com/vnp-memory/services/memobase-ingestion/internal/usecase/port"
)

// BufferConfig holds buffer flush configuration.
type BufferConfig struct {
	MaxChatBlobTokenSize  int
	FlushInterval         time.Duration
	CheckInterval         time.Duration
	MaxConcurrentFlush    int
	PersistentChatBlobs   bool
}

// DefaultBufferConfig returns production defaults.
func DefaultBufferConfig() BufferConfig {
	return BufferConfig{
		MaxChatBlobTokenSize: 1024,
		FlushInterval:        time.Hour,
		CheckInterval:        5 * time.Minute,
		MaxConcurrentFlush:   5,
		PersistentChatBlobs:  false,
	}
}

// ─── InsertBlob ────────────────────────────────────────────────────────────────

// InsertBlobUseCase handles the full lifecycle of inserting a new blob.
type InsertBlobUseCase struct {
	blobRepo   port.BlobRepository
	bufferRepo port.BufferRepository
	tokenizer  tokenizer.Tokenizer
	publisher  port.EventPublisher
	flushUC    *FlushBufferUseCase
	config     BufferConfig
}

func NewInsertBlobUseCase(
	blobRepo port.BlobRepository,
	bufferRepo port.BufferRepository,
	tok tokenizer.Tokenizer,
	publisher port.EventPublisher,
	flushUC *FlushBufferUseCase,
	config BufferConfig,
) *InsertBlobUseCase {
	return &InsertBlobUseCase{
		blobRepo:   blobRepo,
		bufferRepo: bufferRepo,
		tokenizer:  tok,
		publisher:  publisher,
		flushUC:    flushUC,
		config:     config,
	}
}

// InsertBlobRequest is the input to InsertBlobUseCase.
type InsertBlobRequest struct {
	UserID           uuid.UUID
	ProjectID        string
	BlobData         domain.BlobData
	AdditionalFields map[string]any
}

// InsertBlobResult is the output from InsertBlobUseCase.
type InsertBlobResult struct {
	BlobID         string
	FlushTriggered bool
}

// Execute inserts a blob and adds it to the buffer, triggering flush if threshold exceeded.
func (uc *InsertBlobUseCase) Execute(ctx context.Context, req InsertBlobRequest) (*InsertBlobResult, error) {
	if err := req.BlobData.Validate(); err != nil {
		return nil, err
	}

	// Determine blob type and count tokens
	var blobType domain.BlobType
	var tokenCount int
	switch d := req.BlobData.(type) {
	case *domain.ChatBlobData:
		blobType = domain.BlobTypeChat
		msgs := make([]tokenizer.ChatMessage, len(d.Messages))
		for i, m := range d.Messages {
			msgs[i] = tokenizer.ChatMessage{Role: m.Role, Content: m.Content}
		}
		tokenCount = uc.tokenizer.CountMessages(msgs)
	case *domain.DocBlobData:
		blobType = domain.BlobTypeDoc
		tokenCount = uc.tokenizer.Count(d.Text)
	case *domain.SummaryBlobData:
		blobType = domain.BlobTypeSummary
		tokenCount = uc.tokenizer.Count(d.Text)
	default:
		return nil, fmt.Errorf("unknown blob data type")
	}

	// Persist blob
	blob := &domain.Blob{
		UserID:           req.UserID,
		ProjectID:        req.ProjectID,
		BlobType:         blobType,
		BlobData:         req.BlobData,
		AdditionalFields: req.AdditionalFields,
	}
	savedBlob, err := uc.blobRepo.Save(ctx, blob)
	if err != nil {
		return nil, err
	}

	// Add to buffer
	bufEntry := &domain.BufferZone{
		ProjectID: req.ProjectID,
		UserID:    req.UserID,
		BlobID:    savedBlob.ID,
		BlobType:  blobType,
		TokenSize: tokenCount,
		Status:    domain.BufferStatusIdle,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if _, err := uc.bufferRepo.Save(ctx, bufEntry); err != nil {
		return nil, err
	}

	// Check flush threshold
	totalTokens, _ := uc.bufferRepo.GetTotalIdleTokens(ctx, req.UserID, req.ProjectID)
	flushTriggered := false
	if totalTokens >= uc.config.MaxChatBlobTokenSize {
		flushTriggered = true
		go func() {
			flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, _ = uc.flushUC.Execute(flushCtx, FlushBufferRequest{
				UserID:    req.UserID,
				ProjectID: req.ProjectID,
				BlobType:  blobType,
				Sync:      false,
			})
		}()
	}

	return &InsertBlobResult{BlobID: savedBlob.ID.String(), FlushTriggered: flushTriggered}, nil
}

// ─── FlushBuffer ───────────────────────────────────────────────────────────────

// FlushBufferUseCase handles buffer flush operations.
type FlushBufferUseCase struct {
	bufferRepo port.BufferRepository
	blobRepo   port.BlobRepository
	publisher  port.EventPublisher
	config     BufferConfig
}

func NewFlushBufferUseCase(
	bufferRepo port.BufferRepository,
	blobRepo port.BlobRepository,
	publisher port.EventPublisher,
	config BufferConfig,
) *FlushBufferUseCase {
	return &FlushBufferUseCase{
		bufferRepo: bufferRepo,
		blobRepo:   blobRepo,
		publisher:  publisher,
		config:     config,
	}
}

// FlushBufferRequest is the input to FlushBufferUseCase.
type FlushBufferRequest struct {
	UserID    uuid.UUID
	ProjectID string
	BlobType  domain.BlobType
	Sync      bool // if true: sync gRPC call; if false: NATS publish
}

// FlushBufferResult is the output from FlushBufferUseCase.
type FlushBufferResult struct {
	BlobsFlushed int
	Skipped      bool
}

// Execute acquires the flush lock and publishes to NATS (or calls engine sync).
func (uc *FlushBufferUseCase) Execute(ctx context.Context, req FlushBufferRequest) (*FlushBufferResult, error) {
	// Atomic lock: get all idle buffer entries → set to processing
	entries, err := uc.bufferRepo.AcquireProcessingLock(ctx, req.UserID, req.ProjectID, req.BlobType)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return &FlushBufferResult{Skipped: true}, nil
	}

	bufferIDs := make([]string, len(entries))
	bufferUUIDs := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		bufferIDs[i] = e.ID.String()
		bufferUUIDs[i] = e.ID
	}

	// Publish buffer ready event
	payload := port.BufferReadyPayload{
		UserID:    req.UserID.String(),
		ProjectID: req.ProjectID,
		BufferIDs: bufferIDs,
		BlobType:  string(req.BlobType),
	}
	if err := uc.publisher.PublishBufferReady(ctx, payload); err != nil {
		// Rollback: mark entries as failed
		_ = uc.bufferRepo.MarkFailed(ctx, bufferUUIDs, req.ProjectID, fmt.Sprintf("publish failed: %v", err))
		return nil, err
	}

	return &FlushBufferResult{BlobsFlushed: len(entries)}, nil
}

// ─── AutoFlushScheduler ────────────────────────────────────────────────────────

// AutoFlushScheduler periodically flushes buffers for stale users.
type AutoFlushScheduler struct {
	bufferRepo port.BufferRepository
	flushUC    *FlushBufferUseCase
	config     BufferConfig
}

func NewAutoFlushScheduler(bufferRepo port.BufferRepository, flushUC *FlushBufferUseCase, config BufferConfig) *AutoFlushScheduler {
	return &AutoFlushScheduler{bufferRepo: bufferRepo, flushUC: flushUC, config: config}
}

// Run starts the auto-flush tick loop. Call as go scheduler.Run(ctx).
func (s *AutoFlushScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.flushStaleUsers(ctx)
		}
	}
}

func (s *AutoFlushScheduler) flushStaleUsers(ctx context.Context) {
	users, err := s.bufferRepo.GetUsersWithStaleIdleBuffers(ctx, s.config.FlushInterval)
	if err != nil {
		return
	}
	for _, up := range users {
		flushCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		_, _ = s.flushUC.Execute(flushCtx, FlushBufferRequest{
			UserID:    up.UserID,
			ProjectID: up.ProjectID,
		})
		cancel()
	}
}

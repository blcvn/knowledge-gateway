// Package memobase implements memobase usecases.
//
// IngestUseCase: blob insertion + embedding + buffer management
// ContextUseCase: user context aggregation
// (MERGE-P2-T3)
package memobase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vnp-memory/services/memory-service/internal/domain/memobase"
	"vnp-memory/services/memory-service/internal/usecase/port"
)

const defaultFlushThreshold = 20

// IngestUseCase handles blob insertion and buffer management.
type IngestUseCase struct {
	blobs    port.BlobRepository
	embedder port.EmbeddingService
	engine   port.MemoryEngine
	pub      port.EventPublisher
	flushAt  int
}

// NewIngestUseCase creates an IngestUseCase.
func NewIngestUseCase(b port.BlobRepository, e port.EmbeddingService, eng port.MemoryEngine, pub port.EventPublisher) *IngestUseCase {
	return &IngestUseCase{blobs: b, embedder: e, engine: eng, pub: pub, flushAt: defaultFlushThreshold}
}

// InsertBlob persists a new blob and optionally triggers flush.
func (uc *IngestUseCase) InsertBlob(ctx context.Context, userID, tenantID string, blob *memobase.Blob) (*memobase.Blob, error) {
	blob.ID = uuid.New().String()
	blob.UserID = userID
	blob.TenantID = tenantID
	blob.CreatedAt = time.Now()

	// Generate embedding (non-fatal)
	if uc.embedder != nil {
		if emb, err := uc.embedder.Embed(ctx, blob.Content); err == nil {
			blob.Embedding = emb
		}
	}

	if err := uc.blobs.Create(ctx, blob); err != nil {
		return nil, fmt.Errorf("insert blob: %w", err)
	}

	// Auto-flush if buffer exceeds threshold
	if uc.engine != nil {
		if bufSize, err := uc.blobs.GetBufferSize(ctx, userID); err == nil && bufSize >= uc.flushAt {
			go func() {
				_ = uc.engine.ProcessBuffer(context.Background(), userID)
			}()
		}
	}

	if uc.pub != nil {
		_ = uc.pub.Publish(ctx, "memory.blob.inserted", blob)
	}
	return blob, nil
}

// Flush explicitly triggers buffer processing.
func (uc *IngestUseCase) Flush(ctx context.Context, userID string) error {
	if uc.engine == nil {
		return fmt.Errorf("memory engine: not configured")
	}
	return uc.engine.ProcessBuffer(ctx, userID)
}

// ContextUseCase aggregates user memory into a unified context.
type ContextUseCase struct {
	blobs    port.BlobRepository
	profiles port.ProfileRepository
	events   port.EventRepository
	engine   port.MemoryEngine
	maxTokens int
}

// NewContextUseCase creates a ContextUseCase.
func NewContextUseCase(b port.BlobRepository, p port.ProfileRepository, e port.EventRepository, eng port.MemoryEngine) *ContextUseCase {
	return &ContextUseCase{blobs: b, profiles: p, events: e, engine: eng, maxTokens: 4096}
}

// GetContext returns the aggregated user context.
func (uc *ContextUseCase) GetContext(ctx context.Context, userID, tenantID string) (*memobase.UserContext, error) {
	blobs, _ := uc.blobs.List(ctx, userID, tenantID, 50)
	profiles, _ := uc.profiles.GetByUser(ctx, userID, tenantID)
	events, _ := uc.events.GetByUser(ctx, userID, 10)

	summary := "No context available."
	if uc.engine != nil && len(blobs) > 0 {
		summary = uc.engine.Summarize(blobs)
	} else if len(blobs) > 0 {
		// Simple heuristic summary
		var parts []string
		for _, b := range blobs {
			if len(b.Content) > 100 {
				parts = append(parts, b.Content[:100]+"…")
			} else {
				parts = append(parts, b.Content)
			}
			if len(parts) >= 5 {
				break
			}
		}
		summary = strings.Join(parts, "\n")
	}

	tokenCount := len(summary) / 4 // rough estimate

	return &memobase.UserContext{
		UserID:   userID,
		TenantID: tenantID,
		Summary:  summary,
		Profiles: profiles,
		Events:   events,
		Tokens:   tokenCount,
	}, nil
}

// GetProfiles returns user profiles.
func (uc *ContextUseCase) GetProfiles(ctx context.Context, userID, tenantID string) ([]*memobase.Profile, error) {
	return uc.profiles.GetByUser(ctx, userID, tenantID)
}

// GetEvents returns recent user events.
func (uc *ContextUseCase) GetEvents(ctx context.Context, userID string, limit int) ([]*memobase.Event, error) {
	if limit <= 0 {
		limit = 20
	}
	return uc.events.GetByUser(ctx, userID, limit)
}

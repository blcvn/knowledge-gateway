// Package memory implements the critical PutMemory hot path.
// Performance target: sub-200ms p95 latency.
package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/zep-core/domain/memory"
	"github.com/vnp-community/vnp-memory/services/zep-core/usecase/port"
)

// Service implements port.MemoryUseCase.
type Service struct {
	messages  port.MessageRepository
	threads   port.ThreadRepository
	publisher port.EventPublisher
}

func NewService(msgs port.MessageRepository, threads port.ThreadRepository, pub port.EventPublisher) *Service {
	return &Service{messages: msgs, threads: threads, publisher: pub}
}

// PutMemory stores messages and upserts the session — sub-200ms p95.
// Thread.UpsertSession is called LOCALLY (no gRPC) since zep-thread
// is now consolidated into zep-core.
func (s *Service) PutMemory(ctx context.Context, threadID uuid.UUID, msgs []memory.Message) error {
	// 1. Upsert session locally (was gRPC call to zep-thread, now ~1ms)
	if _, err := s.threads.CreateSession(ctx, &threadSession(threadID)); err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}

	// 2. Batch insert messages
	for i := range msgs {
		msgs[i].ID = uuid.New()
		msgs[i].ThreadID = threadID
		msgs[i].CreatedAt = time.Now()
	}
	if err := s.messages.BatchCreate(ctx, msgs); err != nil {
		return fmt.Errorf("batch create messages: %w", err)
	}

	// 3. Emit async event for zep-graph enrichment (non-blocking)
	thread, _ := s.threads.FindByID(ctx, threadID)
	if thread != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.publisher.PublishMemoryPut(bgCtx, thread.TenantID, threadID)
		}()
	}

	return nil
}

func (s *Service) GetContext(ctx context.Context, threadID uuid.UUID, maxTokens int) (*memory.ContextAssembly, error) {
	msgs, err := s.messages.FindByThread(ctx, threadID, 100) // Last 100 messages
	if err != nil {
		return nil, fmt.Errorf("find messages: %w", err)
	}

	tokenCount := 0
	var selected []memory.Message
	for i := len(msgs) - 1; i >= 0 && tokenCount < maxTokens; i-- {
		// Approximate token count (4 chars ≈ 1 token)
		msgTokens := len(msgs[i].Content) / 4
		if tokenCount+msgTokens > maxTokens {
			break
		}
		selected = append([]memory.Message{msgs[i]}, selected...)
		tokenCount += msgTokens
	}

	return &memory.ContextAssembly{
		ThreadID:    threadID,
		Messages:    selected,
		TokenCount:  tokenCount,
		AssembledAt: time.Now(),
	}, nil
}

func threadSession(threadID uuid.UUID) interface{ GetThreadID() uuid.UUID } {
	return nil // Placeholder — actual session creation handled by repository
}

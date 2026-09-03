package memory

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/vnp-memory/services/zep-core/domain/memory"
)

type PutMemoryUsecase struct {
	msgRepo  memory.Repository
	eventPub EventPublisher
	logger   *zap.Logger
}

type EventPublisher interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
}

func NewPutMemoryUsecase(msgRepo memory.Repository, eventPub EventPublisher, logger *zap.Logger) *PutMemoryUsecase {
	return &PutMemoryUsecase{
		msgRepo:  msgRepo,
		eventPub: eventPub,
		logger:   logger,
	}
}

// Execute performs the high-performance memory ingestion path (target <200ms).
func (uc *PutMemoryUsecase) Execute(ctx context.Context, msg *memory.Message) error {
	start := time.Now()

	// 1. Persist the message to PostgreSQL
	if err := uc.msgRepo.Save(ctx, msg); err != nil {
		uc.logger.Error("failed to save message", zap.Error(err), zap.String("uuid", msg.UUID.String()))
		return fmt.Errorf("failed to save message: %w", err)
	}

	// 2. Publish async event to zep-graph for KG extraction
	payload := map[string]interface{}{
		"message_uuid": msg.UUID,
		"thread_uuid":  msg.ThreadUUID,
		"content":      msg.Content,
		"timestamp":    msg.CreatedAt,
	}
	
	if err := uc.eventPub.Publish(ctx, "zep.memory.messages.ingested", payload); err != nil {
		// Log but do not fail the synchronous path
		uc.logger.Warn("failed to publish ingestion event", zap.Error(err))
	}

	uc.logger.Info("PutMemory completed successfully", zap.Duration("latency", time.Since(start)))
	return nil
}

// GenerateAdvisoryLockID computes a stable PostgreSQL advisory lock ID using SHA-256
func GenerateAdvisoryLockID(sessionID string) int64 {
	h := sha256.Sum256([]byte(sessionID))
	return int64(binary.BigEndian.Uint64(h[:8]))
}

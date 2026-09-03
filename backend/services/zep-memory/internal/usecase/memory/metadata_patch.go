package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/services/zep-core/domain/user"
	"go.uber.org/zap"
)

type MetadataPatcher struct {
	userRepo user.Repository
	dbLocker DBLocker
	logger   *zap.Logger
}

type DBLocker interface {
	AcquireAdvisoryLock(ctx context.Context, lockID int64) (bool, error)
	ReleaseAdvisoryLock(ctx context.Context, lockID int64) error
}

func NewMetadataPatcher(userRepo user.Repository, locker DBLocker, logger *zap.Logger) *MetadataPatcher {
	return &MetadataPatcher{
		userRepo: userRepo,
		dbLocker: locker,
		logger:   logger,
	}
}

// PatchUserMetadata performs a concurrency-safe JSONB merge patch.
func (mp *MetadataPatcher) PatchUserMetadata(ctx context.Context, projectUUID uuid.UUID, userID string, patch map[string]interface{}) error {
	// Generate stable lock ID using SHA-256 of UserID
	lockID := GenerateAdvisoryLockID(userID)

	// Implement Exponential Backoff Retry (200ms -> 30s)
	backoff := 200 * time.Millisecond
	for i := 0; i < 15; i++ {
		acquired, err := mp.dbLocker.AcquireAdvisoryLock(ctx, lockID)
		if err != nil {
			return fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			defer mp.dbLocker.ReleaseAdvisoryLock(context.Background(), lockID)
			
			// 1. Fetch current user
			usr, err := mp.userRepo.GetByID(ctx, projectUUID, userID)
			if err != nil {
				return err
			}

			// 2. Perform merge patch
			if usr.Metadata == nil {
				usr.Metadata = make(map[string]interface{})
			}
			for k, v := range patch {
				usr.Metadata[k] = v
			}

			// 3. Save
			return mp.userRepo.Upsert(ctx, usr)
		}

		mp.logger.Warn("advisory lock collision, retrying", zap.Int("attempt", i+1), zap.Duration("backoff", backoff))
		time.Sleep(backoff)
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}

	return fmt.Errorf("could not acquire lock after 15 attempts for user %s", userID)
}

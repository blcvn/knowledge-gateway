package surrealdb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// surrealLockManager implements lock.LockManager using SurrealDB table-based locks.
type surrealLockManager struct {
	client *Client
	log    *log.Helper
}

func NewSurrealLockManager(client *Client, logger log.Logger) *surrealLockManager {
	return &surrealLockManager{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (m *surrealLockManager) AcquireNodeLock(ctx context.Context, namespace, nodeID string, ttl time.Duration) (string, error) {
	lockKey := fmt.Sprintf("%s:node:%s", namespace, nodeID)
	return m.acquireLock(ctx, lockKey, ttl)
}

func (m *surrealLockManager) AcquireNamespaceLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	lockKey := fmt.Sprintf("%s:ns", namespace)
	return m.acquireLock(ctx, lockKey, ttl)
}

func (m *surrealLockManager) Release(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	sql := `DELETE FROM kg_locks WHERE token = $token`
	_, err := m.client.Query(ctx, sql, map[string]any{"token": token})
	if err != nil {
		m.log.Errorf("[KGS][SurrealDB] Release lock failed token=%s err=%v", token, err)
	}
	return err
}

func (m *surrealLockManager) acquireLock(ctx context.Context, lockKey string, ttl time.Duration) (string, error) {
	token := uuid.NewString()
	expiresAt := time.Now().Add(ttl)

	// First, try to clean up expired locks
	cleanSQL := `DELETE FROM kg_locks WHERE lock_key = $lock_key AND expires_at < time::now()`
	_, _ = m.client.Query(ctx, cleanSQL, map[string]any{"lock_key": lockKey})

	// Try to create the lock (UNIQUE index on lock_key will prevent duplicates)
	createSQL := `CREATE type::thing('kg_locks', $lock_key) SET
		lock_key = $lock_key,
		token = $token,
		expires_at = $expires_at,
		created_at = time::now()`
	_, err := m.client.Query(ctx, createSQL, map[string]any{
		"lock_key":   lockKey,
		"token":      token,
		"expires_at": expiresAt,
	})
	if err != nil {
		// Lock already held by someone else
		m.log.Warnf("[KGS][SurrealDB] Lock contention lock_key=%s err=%v", lockKey, err)
		return "", fmt.Errorf("lock already held: %s", lockKey)
	}

	return token, nil
}

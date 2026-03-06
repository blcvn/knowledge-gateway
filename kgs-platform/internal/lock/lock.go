package lock

import (
	"context"
	"time"
)

type LockManager interface {
	AcquireNodeLock(ctx context.Context, namespace, nodeID string, ttl time.Duration) (string, error)
	AcquireSubgraphLock(ctx context.Context, namespace, rootID string, depth int, ttl time.Duration) (string, error)
	AcquireVersionLock(ctx context.Context, namespace string, ttl time.Duration) (string, error)
	AcquireNamespaceLock(ctx context.Context, namespace string, ttl time.Duration) (string, error)
	Release(ctx context.Context, lockToken string) error
}

type contextKey string

const OwnerContextKey contextKey = "kgs_lock_owner_id"

func WithOwnerID(ctx context.Context, ownerID string) context.Context {
	return context.WithValue(ctx, OwnerContextKey, ownerID)
}

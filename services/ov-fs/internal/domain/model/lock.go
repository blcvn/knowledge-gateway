package lock

import (
\t"context"
\t"errors"
\t"sync"
\t"time"
)

// PathLock represents a lease on a specific file system path to prevent concurrent modifications.
type PathLock struct {
\tTenantID  string
\tPath      string
\tOwnerID   string    // ID of the process or request holding the lock
\tExpiresAt time.Time
}

// PathLockManager defines the port for distributed lock management (e.g. via Redis).
type PathLockManager interface {
\tAcquire(ctx context.Context, tenantID, path, ownerID string, ttl time.Duration) (*PathLock, error)
\tRelease(ctx context.Context, lock *PathLock) error
}

// InMemPathLockManager is a naive in-memory implementation for single-node testing.
// For production, this should be replaced by a Redis-based distributed lock manager.
type InMemPathLockManager struct {
\tmu    sync.Mutex
\tlocks map[string]*PathLock
}

// NewInMemPathLockManager creates a new InMemPathLockManager.
func NewInMemPathLockManager() *InMemPathLockManager {
\treturn &InMemPathLockManager{
\t\tlocks: make(map[string]*PathLock),
\t}
}

// Acquire attempts to acquire a lock on a path for a given tenant.
func (m *InMemPathLockManager) Acquire(ctx context.Context, tenantID, path, ownerID string, ttl time.Duration) (*PathLock, error) {
\tm.mu.Lock()
\tdefer m.mu.Unlock()

\tkey := tenantID + ":" + path
\texisting, found := m.locks[key]
\tif found {
\t\tif time.Now().Before(existing.ExpiresAt) {
\t\t\treturn nil, errors.New("path is currently locked by another process")
\t\t}
\t\t// Existing lock is expired, we can take it over
\t}

\tlock := &PathLock{
\t\tTenantID:  tenantID,
\t\tPath:      path,
\t\tOwnerID:   ownerID,
\t\tExpiresAt: time.Now().Add(ttl),
\t}
\t
\tm.locks[key] = lock
\treturn lock, nil
}

// Release frees the lock if the owner matches.
func (m *InMemPathLockManager) Release(ctx context.Context, lock *PathLock) error {
\tm.mu.Lock()
\tdefer m.mu.Unlock()

\tkey := lock.TenantID + ":" + lock.Path
\texisting, found := m.locks[key]
\tif !found {
\t\treturn nil // already released or expired
\t}
\tif existing.OwnerID != lock.OwnerID {
\t\treturn errors.New("cannot release lock owned by another process")
\t}

\tdelete(m.locks, key)
\treturn nil
}

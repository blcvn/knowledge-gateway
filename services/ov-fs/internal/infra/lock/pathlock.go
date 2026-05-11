package lock

import (
	"context"
	"sync"
	"time"

	"vnp-memory/services/ov-fs/internal/domain/model"
)

type PathLockManager interface {
	Acquire(ctx context.Context, req model.LockRequest, timeout time.Duration) error
	Release(ctx context.Context, req model.LockRequest) error
}

type pathLock struct {
	mu    sync.Mutex
	locks map[string]*model.LockRequest
}

func NewPathLockManager() PathLockManager {
	return &pathLock{
		locks: make(map[string]*model.LockRequest),
	}
}

func (p *pathLock) Acquire(ctx context.Context, req model.LockRequest, timeout time.Duration) error {
	// Simple in-memory mock implementation
	// Real implementation should use PostgreSQL advisory locks for distributed concurrency
	p.mu.Lock()
	defer p.mu.Unlock()
	p.locks[req.AccountID+":"+req.Path] = &req
	return nil
}

func (p *pathLock) Release(ctx context.Context, req model.LockRequest) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.locks, req.AccountID+":"+req.Path)
	return nil
}

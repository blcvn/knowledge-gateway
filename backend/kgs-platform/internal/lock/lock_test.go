package lock

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type memoryLockStore struct {
	mu   sync.Mutex
	data map[string]string
}

func (s *memoryLockStore) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]string)
	}
	if _, exists := s.data[key]; exists {
		return redis.NewBoolResult(false, nil)
	}
	s.data[key] = value.(string)
	return redis.NewBoolResult(true, nil)
}

func (s *memoryLockStore) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]string)
	}
	key := keys[0]
	token := args[0].(string)
	if s.data[key] == token {
		delete(s.data, key)
		return redis.NewCmdResult(int64(1), nil)
	}
	return redis.NewCmdResult(int64(0), nil)
}

func TestRedisLockManagerConcurrentAcquire(t *testing.T) {
	manager := NewRedisLockManagerWithStore(&memoryLockStore{})
	ctxA := WithOwnerID(context.Background(), "owner-a")
	ctxB := WithOwnerID(context.Background(), "owner-b")

	token1, err := manager.AcquireNodeLock(ctxA, "graph/app/default", "node-1", time.Second)
	if err != nil || token1 == "" {
		t.Fatalf("first acquire failed: %v", err)
	}

	ch := make(chan error, 1)
	go func() {
		_, err := manager.AcquireNodeLock(ctxB, "graph/app/default", "node-1", time.Second)
		ch <- err
	}()

	select {
	case err := <-ch:
		if err == nil {
			t.Fatalf("expected timeout/lock error")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("second acquire did not return in time")
	}

	if err := manager.Release(ctxA, token1); err != nil {
		t.Fatalf("release failed: %v", err)
	}
}

func TestRedisLockManagerHierarchy(t *testing.T) {
	manager := NewRedisLockManagerWithStore(&memoryLockStore{})
	ctx := WithOwnerID(context.Background(), "owner-h")

	nsToken, err := manager.AcquireNamespaceLock(ctx, "graph/app/default", time.Second)
	if err != nil {
		t.Fatalf("AcquireNamespaceLock error: %v", err)
	}
	defer func() { _ = manager.Release(ctx, nsToken) }()

	_, err = manager.AcquireNodeLock(ctx, "graph/app/default", "node-1", time.Second)
	if err == nil {
		t.Fatalf("expected hierarchy violation")
	}
}

func TestRedisLockManagerReentrant(t *testing.T) {
	manager := NewRedisLockManagerWithStore(&memoryLockStore{})
	ctx := WithOwnerID(context.Background(), "owner-r")

	token1, err := manager.AcquireNodeLock(ctx, "graph/app/default", "node-1", time.Second)
	if err != nil {
		t.Fatalf("AcquireNodeLock first error: %v", err)
	}
	token2, err := manager.AcquireNodeLock(ctx, "graph/app/default", "node-1", time.Second)
	if err != nil {
		t.Fatalf("AcquireNodeLock reentrant error: %v", err)
	}
	if token1 != token2 {
		t.Fatalf("expected same token for reentrant lock")
	}

	if err := manager.Release(ctx, token1); err != nil {
		t.Fatalf("release #1 failed: %v", err)
	}
	if err := manager.Release(ctx, token2); err != nil {
		t.Fatalf("release #2 failed: %v", err)
	}
}

func TestRedisLockManagerAcquireTimeoutConfigurable(t *testing.T) {
	manager := NewRedisLockManagerWithStoreAndTimeout(&memoryLockStore{}, 30*time.Millisecond)
	ctxA := WithOwnerID(context.Background(), "owner-a")
	ctxB := WithOwnerID(context.Background(), "owner-b")

	token, err := manager.AcquireNodeLock(ctxA, "graph/app/default", "node-1", time.Second)
	if err != nil || token == "" {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer func() { _ = manager.Release(ctxA, token) }()

	start := time.Now()
	_, err = manager.AcquireNodeLock(ctxB, "graph/app/default", "node-1", time.Second)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected lock acquisition timeout")
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("expected lock wait to honor configured timeout, elapsed=%v", elapsed)
	}
}

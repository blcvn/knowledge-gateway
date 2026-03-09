package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

const (
	levelNode = iota + 1
	levelSubgraph
	levelVersion
	levelNamespace

	defaultLockAcquireTimeout = 2 * time.Second
	lockAcquireTimeoutEnvKey  = "KGS_LOCK_ACQUIRE_TIMEOUT"
)

var (
	ErrLockTimeout       = errors.New("lock acquisition timeout")
	ErrLockHierarchy     = errors.New("lock hierarchy violation")
	ErrUnknownLockToken  = errors.New("unknown lock token")
	ErrLockReleaseFailed = errors.New("lock release failed")
)

type redisLockStore interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

type RedisLockManager struct {
	store          redisLockStore
	acquireTimeout time.Duration

	mu       sync.Mutex
	byToken  map[string]lockRecord
	byOwner  map[string]map[string]lockRecord // owner -> key -> lock
	ownerMax map[string]int
}

type lockRecord struct {
	token string
	key   string
	owner string
	level int
	count int
}

func NewRedisLockManager(redisCli *redis.Client) LockManager {
	return NewRedisLockManagerWithStoreAndTimeout(redisCli, lockAcquireTimeoutFromEnv())
}

func NewRedisLockManagerWithStore(store redisLockStore) LockManager {
	return NewRedisLockManagerWithStoreAndTimeout(store, defaultLockAcquireTimeout)
}

func NewRedisLockManagerWithStoreAndTimeout(store redisLockStore, acquireTimeout time.Duration) LockManager {
	if store == nil {
		return nil
	}
	if acquireTimeout <= 0 {
		acquireTimeout = defaultLockAcquireTimeout
	}
	return &RedisLockManager{
		store:          store,
		acquireTimeout: acquireTimeout,
		byToken:        make(map[string]lockRecord),
		byOwner:        make(map[string]map[string]lockRecord),
		ownerMax:       make(map[string]int),
	}
}

func (m *RedisLockManager) AcquireNodeLock(ctx context.Context, namespace, nodeID string, ttl time.Duration) (string, error) {
	key := fmt.Sprintf("kgs:lock:node:%s:%s", sanitizeNamespace(namespace), nodeID)
	return m.acquire(ctx, key, levelNode, ttl)
}

func (m *RedisLockManager) AcquireSubgraphLock(ctx context.Context, namespace, rootID string, depth int, ttl time.Duration) (string, error) {
	key := fmt.Sprintf("kgs:lock:subgraph:%s:%s:%d", sanitizeNamespace(namespace), rootID, depth)
	return m.acquire(ctx, key, levelSubgraph, ttl)
}

func (m *RedisLockManager) AcquireVersionLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	key := fmt.Sprintf("kgs:lock:version:%s", sanitizeNamespace(namespace))
	return m.acquire(ctx, key, levelVersion, ttl)
}

func (m *RedisLockManager) AcquireNamespaceLock(ctx context.Context, namespace string, ttl time.Duration) (string, error) {
	key := fmt.Sprintf("kgs:lock:ns:%s", sanitizeNamespace(namespace))
	return m.acquire(ctx, key, levelNamespace, ttl)
}

func (m *RedisLockManager) Release(ctx context.Context, lockToken string) error {
	traceCtx, span := observability.StartDependencySpan(ctx, "redis", "redis.lock.release")
	defer span.End()
	m.mu.Lock()
	rec, ok := m.byToken[lockToken]
	if !ok {
		m.mu.Unlock()
		observability.RecordSpanError(span, ErrUnknownLockToken)
		return ErrUnknownLockToken
	}
	if rec.count > 1 {
		rec.count--
		m.byToken[lockToken] = rec
		ownerLocks := m.byOwner[rec.owner]
		ownerLocks[rec.key] = rec
		m.mu.Unlock()
		return nil
	}
	delete(m.byToken, lockToken)
	ownerLocks := m.byOwner[rec.owner]
	delete(ownerLocks, rec.key)
	if len(ownerLocks) == 0 {
		delete(m.byOwner, rec.owner)
		delete(m.ownerMax, rec.owner)
	} else {
		m.ownerMax[rec.owner] = computeOwnerMax(ownerLocks)
	}
	m.mu.Unlock()

	span.SetAttributes(attribute.String("redis.key", rec.key))
	res, err := m.store.Eval(traceCtx, releaseScript, []string{rec.key}, rec.token).Result()
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	switch v := res.(type) {
	case int64:
		if v == 1 {
			return nil
		}
	case string:
		if v == "1" {
			return nil
		}
	}
	return ErrLockReleaseFailed
}

func (m *RedisLockManager) acquire(ctx context.Context, key string, level int, ttl time.Duration) (string, error) {
	started := time.Now()
	traceCtx, span := observability.StartDependencySpan(ctx, "redis", "redis.lock.acquire", attribute.String("redis.key", key))
	defer span.End()

	owner := ownerIDFromContext(ctx)

	m.mu.Lock()
	if m.ownerMax[owner] > level {
		m.mu.Unlock()
		return "", ErrLockHierarchy
	}
	if ownerLocks, ok := m.byOwner[owner]; ok {
		if rec, ok := ownerLocks[key]; ok {
			rec.count++
			m.byToken[rec.token] = rec
			ownerLocks[key] = rec
			m.mu.Unlock()
			return rec.token, nil
		}
	}
	m.mu.Unlock()

	token := uuid.NewString()
	deadline := time.Now().Add(m.acquireTimeout)
	for {
		ok, err := m.store.SetNX(traceCtx, key, token, ttl).Result()
		if err == nil && ok {
			m.mu.Lock()
			rec := lockRecord{
				token: token,
				key:   key,
				owner: owner,
				level: level,
				count: 1,
			}
			m.byToken[token] = rec
			if _, exists := m.byOwner[owner]; !exists {
				m.byOwner[owner] = make(map[string]lockRecord)
			}
			m.byOwner[owner][key] = rec
			if m.ownerMax[owner] < level {
				m.ownerMax[owner] = level
			}
			m.mu.Unlock()
			observability.ObserveLockAcquire(levelToMetricLabel(level), time.Since(started), nil)
			return token, nil
		}
		if err != nil {
			observability.RecordSpanError(span, err)
			observability.ObserveLockAcquire(levelToMetricLabel(level), time.Since(started), err)
			return "", err
		}
		if ctx.Err() != nil || time.Now().After(deadline) {
			observability.RecordSpanError(span, ErrLockTimeout)
			observability.ObserveLockAcquire(levelToMetricLabel(level), time.Since(started), ErrLockTimeout)
			return "", ErrLockTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func lockAcquireTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(lockAcquireTimeoutEnvKey))
	if raw == "" {
		return defaultLockAcquireTimeout
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultLockAcquireTimeout
	}
	return parsed
}

func ownerIDFromContext(ctx context.Context) string {
	if owner, ok := ctx.Value(OwnerContextKey).(string); ok && owner != "" {
		return owner
	}
	return "owner-" + uuid.NewString()
}

func sanitizeNamespace(namespace string) string {
	replacer := strings.NewReplacer("/", ":", " ", "_")
	return replacer.Replace(namespace)
}

func computeOwnerMax(lockMap map[string]lockRecord) int {
	maxLevel := 0
	for _, rec := range lockMap {
		if rec.level > maxLevel {
			maxLevel = rec.level
		}
	}
	return maxLevel
}

func levelToMetricLabel(level int) string {
	switch level {
	case levelNode:
		return "node"
	case levelSubgraph:
		return "subgraph"
	case levelVersion:
		return "version"
	case levelNamespace:
		return "namespace"
	default:
		return "unknown"
	}
}

const releaseScript = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end
`

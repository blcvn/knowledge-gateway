# TASK-GR-014 — Search Result Cache + NATS Invalidation

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-014 |
| **Wave** | 3 |
| **Component** | `services/graphiti-search/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-004 §5, §6 |
| **Priority** | High |
| **Depends On** | TASK-GR-013 |
| **Estimated** | 2h |

**Trạng thái:** ✅ Implemented  
**Ghi chú:** graphiti-search NATS cache invalidation  
---

## Context

Implement Redis search result caching và NATS-based cache invalidation. Khi NATS nhận `graphiti.episode.ingested`, tất cả cache entries của `group_id` đó bị xóa.

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/graphiti-search/internal/adapter/cache/redis_cache.go` |
| CREATE | `services/graphiti-search/internal/adapter/nats/cache_invalidator.go` |

---

## Implementation

### File 1: `services/graphiti-search/internal/adapter/cache/redis_cache.go`

```go
package cache

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "github.com/vnp-memory/services/graphiti-search/internal/domain"
    "github.com/vnp-memory/services/graphiti-search/internal/usecase"
)

const defaultCacheTTL = 5 * time.Minute
const cacheKeyPrefix  = "graphiti:search:"

type RedisSearchCache struct {
    client redis.UniversalClient
    ttl    time.Duration
}

func NewRedisSearchCache(client redis.UniversalClient) *RedisSearchCache {
    return &RedisSearchCache{client: client, ttl: defaultCacheTTL}
}

func (c *RedisSearchCache) Get(ctx context.Context, key string) (*domain.SearchResults, bool) {
    val, err := c.client.Get(ctx, cacheKeyPrefix+key).Bytes()
    if err != nil { return nil, false }
    var results domain.SearchResults
    if err := json.Unmarshal(val, &results); err != nil { return nil, false }
    return &results, true
}

func (c *RedisSearchCache) Set(ctx context.Context, key string, results *domain.SearchResults) {
    data, err := json.Marshal(results)
    if err != nil { return }
    c.client.Set(ctx, cacheKeyPrefix+key, data, c.ttl)
}

// InvalidateGroup removes all cached search results for a group_id.
// Uses Redis SCAN to find matching keys (pattern: graphiti:search:*{groupID}*)
func (c *RedisSearchCache) InvalidateGroup(ctx context.Context, groupID string) {
    pattern := fmt.Sprintf("%s*%s*", cacheKeyPrefix, groupID)
    iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
    var keys []string
    for iter.Next(ctx) { keys = append(keys, iter.Val()) }
    if len(keys) > 0 { c.client.Del(ctx, keys...) }
}

// ComputeCacheKey generates a stable 16-char hex key from request params
func ComputeCacheKey(query string, groupIDs []string, config domain.SearchConfig, filters domain.SearchFilters) string {
    h := sha256.New()
    fmt.Fprintf(h, "%s|%v|", query, groupIDs)
    json.NewEncoder(h).Encode(config)
    json.NewEncoder(h).Encode(filters)
    return hex.EncodeToString(h.Sum(nil))[:16]
}
```

### File 2: `services/graphiti-search/internal/adapter/nats/cache_invalidator.go`

```go
package nats

import (
    "context"
    "encoding/json"
    "log"

    "github.com/nats-io/nats.go"
    "github.com/vnp-memory/services/graphiti-search/internal/adapter/cache"
)

type CacheInvalidator struct {
    cache    *cache.RedisSearchCache
    natsConn *nats.Conn
    subs     []*nats.Subscription
}

func NewCacheInvalidator(cache *cache.RedisSearchCache, conn *nats.Conn) *CacheInvalidator {
    return &CacheInvalidator{cache: cache, natsConn: conn}
}

// Start subscribes to all relevant NATS events and invalidates cache on receipt.
func (ci *CacheInvalidator) Start(ctx context.Context) error {
    subjects := []string{
        "graphiti.episode.ingested",
        "graphiti.entity.resolved",
        "graphiti.community.rebuilt",
    }

    for _, subj := range subjects {
        sub, err := ci.natsConn.Subscribe(subj, func(msg *nats.Msg) {
            var payload struct {
                GroupID string `json:"group_id"`
            }
            if err := json.Unmarshal(msg.Data, &payload); err != nil { return }
            if payload.GroupID != "" {
                ci.cache.InvalidateGroup(context.Background(), payload.GroupID)
                log.Printf("search cache invalidated for group: %s (event: %s)", payload.GroupID, msg.Subject)
            }
        })
        if err != nil { return err }
        ci.subs = append(ci.subs, sub)
    }
    return nil
}

func (ci *CacheInvalidator) Stop() {
    for _, sub := range ci.subs { sub.Unsubscribe() }
}
```

---

## Verification

```bash
cd services/graphiti-search
go build ./internal/adapter/cache/...
go build ./internal/adapter/nats/...
```

**Manual test:**
1. Search query → result cached (Redis key exists)
2. POST new episode → NATS event → cache invalidated (Redis key deleted)
3. Same query again → cache miss → fresh search

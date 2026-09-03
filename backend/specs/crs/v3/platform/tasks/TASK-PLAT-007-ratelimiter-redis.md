# TASK-PLAT-007 — Redis Sliding Window Rate Limiter

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-007 |
| **Wave** | 2 (Auth Flows) |
| **Solution** | [SOL-PLAT-002](../solutions/SOL-PLAT-002-Rate-Limiting-Subscription-Tiers.md) §2.1 |
| **Component** | `shared/pkg/resilience/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 4h |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** RedisRateLimiter (112 lines): sliding window with sorted sets — ratelimit_redis.go
---

## Mục tiêu

Implement Redis-based sliding window rate limiter trong `shared/pkg/resilience`. Dùng Lua script để atomic check + increment.

---

## Công việc cụ thể

### 1. Tạo `shared/pkg/resilience/ratelimiter.go` [NEW]

```go
package resilience

import (
    "context"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
)

// TierConfig defines rate limits for a subscription tier
// -1 = unlimited (enterprise)
type TierConfig struct {
    PerMin  int64
    PerHour int64
    PerDay  int64
}

// DefaultTierLimits per CR-PLAT-002 §2 spec table
var DefaultTierLimits = map[string]TierConfig{
    "free":       {PerMin: 20, PerHour: 500, PerDay: 5000},
    "pro":        {PerMin: 200, PerHour: 5000, PerDay: 50000},
    "enterprise": {PerMin: -1, PerHour: -1, PerDay: -1},
}

// EndpointOverrides — endpoint-specific limits that override tier defaults
// key: endpoint path, value: map[tier]rpm
var EndpointLimits = map[string]map[string]int64{
    "/v1/memory/store":   {"free": 20, "pro": 200, "enterprise": -1},
    "/v1/memory/recall":  {"free": 50, "pro": 500, "enterprise": -1},
    "/v1/sm/search":      {"free": 30, "pro": 300, "enterprise": -1},
    "__mcp__":            {"free": 100, "pro": 1000, "enterprise": -1},
}

type RateLimitResult struct {
    Allowed    bool
    Limit      int64
    Remaining  int64
    ResetAt    int64 // Unix timestamp
    RetryAfter int64 // seconds until reset
}

// Lua script: atomic sliding window check (prevents race conditions)
const slidingWindowLua = `
local key = KEYS[1]
local window = tonumber(ARGV[1])    -- window size in seconds
local limit = tonumber(ARGV[2])     -- max requests
local now = tonumber(ARGV[3])       -- current unix ts (ms precision)
local window_start = now - (window * 1000)

redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

local count = redis.call('ZCARD', key)

if count >= limit then
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset_ms = oldest[2] and (oldest[2] + window * 1000) or (now + window * 1000)
    return {0, limit, 0, math.ceil(reset_ms / 1000), math.ceil((reset_ms - now) / 1000)}
end

local member = tostring(now) .. '-' .. tostring(math.random(99999))
redis.call('ZADD', key, now, member)
redis.call('PEXPIRE', key, window * 1000)

return {1, limit, limit - count - 1, math.ceil((now + window * 1000) / 1000), 0}
`

type SlidingWindowRateLimiter struct {
    redis  redis.UniversalClient
    limits map[string]TierConfig
    nats   NATSPublisher // optional: publish rate_limit.exceeded events
}

func NewSlidingWindowRateLimiter(redisClient redis.UniversalClient, nats NATSPublisher) *SlidingWindowRateLimiter {
    return &SlidingWindowRateLimiter{
        redis:  redisClient,
        limits: DefaultTierLimits,
        nats:   nats,
    }
}

// Allow checks if request is within rate limit for tenant+endpoint+tier
func (r *SlidingWindowRateLimiter) Allow(ctx context.Context, tenantID, endpoint, tier string) (RateLimitResult, error) {
    limit := r.limitForEndpoint(endpoint, tier)

    // Enterprise or unlimited → skip Redis
    if limit == -1 {
        return RateLimitResult{Allowed: true, Limit: -1, Remaining: -1}, nil
    }

    key := fmt.Sprintf("rl:%s:%s:1m", tenantID, endpoint)
    nowMs := time.Now().UnixMilli()

    result, err := r.redis.Eval(ctx, slidingWindowLua,
        []string{key},
        60,      // 1 minute window
        limit,
        nowMs,
    ).Slice()

    if err != nil {
        // Fail open on Redis error
        return RateLimitResult{Allowed: true}, fmt.Errorf("redis eval: %w", err)
    }

    toInt64 := func(v interface{}) int64 {
        switch x := v.(type) {
        case int64: return x
        default:    return 0
        }
    }

    rl := RateLimitResult{
        Allowed:    toInt64(result[0]) == 1,
        Limit:      toInt64(result[1]),
        Remaining:  toInt64(result[2]),
        ResetAt:    toInt64(result[3]),
        RetryAfter: toInt64(result[4]),
    }

    if !rl.Allowed && r.nats != nil {
        r.nats.Publish("rate_limit.exceeded", map[string]interface{}{
            "tenant_id": tenantID,
            "endpoint":  endpoint,
            "limit":     limit,
            "window":    "1m",
        })
    }
    return rl, nil
}

func (r *SlidingWindowRateLimiter) limitForEndpoint(endpoint, tier string) int64 {
    if limits, ok := EndpointLimits[endpoint]; ok {
        if v, ok := limits[tier]; ok {
            return v
        }
    }
    // fallback to tier default
    cfg := r.limits[tier]
    if cfg.PerMin == 0 {
        cfg = DefaultTierLimits["free"]
    }
    return cfg.PerMin
}

// GetUsage returns current usage stats (for admin API)
func (r *SlidingWindowRateLimiter) GetUsage(ctx context.Context, tenantID string) map[string]interface{} {
    // Scan Redis keys for this tenant
    pattern := fmt.Sprintf("rl:%s:*", tenantID)
    keys, _ := r.redis.Keys(ctx, pattern).Result()
    usage := make(map[string]int64, len(keys))
    for _, k := range keys {
        count, _ := r.redis.ZCard(ctx, k).Result()
        endpoint := strings.TrimPrefix(k, fmt.Sprintf("rl:%s:", tenantID))
        usage[endpoint] = count
    }
    return map[string]interface{}{"tenant_id": tenantID, "usage": usage}
}
```

### 2. Unit tests `shared/pkg/resilience/ratelimiter_test.go` [NEW]

```go
// Integration test requires Redis (use miniredis for unit test)
func TestSlidingWindow_AllowsUnderLimit(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    rl := NewSlidingWindowRateLimiter(rdb, nil)

    for i := 0; i < 20; i++ {
        result, err := rl.Allow(ctx, "tenant-1", "/v1/memory/store", "free")
        assert.NoError(t, err)
        assert.True(t, result.Allowed)
    }
}

func TestSlidingWindow_BlocksOverLimit(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    rl := NewSlidingWindowRateLimiter(rdb, nil)

    for i := 0; i < 20; i++ {
        rl.Allow(ctx, "tenant-1", "/v1/memory/store", "free")
    }
    // 21st request should be blocked
    result, _ := rl.Allow(ctx, "tenant-1", "/v1/memory/store", "free")
    assert.False(t, result.Allowed)
    assert.Equal(t, int64(20), result.Limit)
    assert.Greater(t, result.RetryAfter, int64(0))
}

func TestSlidingWindow_EnterpriseTier_Unlimited(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    rl := NewSlidingWindowRateLimiter(rdb, nil)

    for i := 0; i < 10000; i++ {
        result, _ := rl.Allow(ctx, "enterprise-tenant", "/v1/memory/store", "enterprise")
        assert.True(t, result.Allowed)
    }
}
```

---

## Acceptance Criteria

- [ ] Lua script atomically checks + increments sliding window (no race conditions)
- [ ] Free tier: limit 20 req/min for `/v1/memory/store`
- [ ] Pro tier: limit 200 req/min for `/v1/memory/store`
- [ ] Enterprise tier: always allowed (returns `Limit=-1`)
- [ ] Redis error → fail open (allow request, log warning)
- [ ] NATS `rate_limit.exceeded` published with tenant_id, endpoint, limit, window
- [ ] `go test ./shared/pkg/resilience/...` passes

## Files

```
shared/pkg/resilience/ratelimiter.go       [NEW]
shared/pkg/resilience/ratelimiter_test.go  [NEW]
```

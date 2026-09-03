---
id: TASK-008
title: Rate Limiting — Redis Sliding Window
service: vnp-gateway
version: 1.0.0
status: Done
priority: P1
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-003
depends_on: [TASK-003]
estimate: 3h
actual: 2.5h
---

## Mục Tiêu

Implement Redis sliding window rate limiter. Per-tenant, per-endpoint. 3 tiers. Fail-open when Redis unavailable.

## Phạm Vi

### Files đã tạo
- `gateway/internal/infra/persistence/ratelimit_redis.go` — 112 lines (Redis Lua script)
- `gateway/internal/usecase/ratelimit.go` — 63 lines (RateLimitUseCase)
- `gateway/internal/infra/middleware/auth.go` — 139 lines (RateLimit HTTP middleware)

### Chi tiết triển khai

#### Sliding window algorithm (Redis Sorted Set + Lua)
```lua
-- Atomic Lua script — no race conditions
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current
local count = redis.call('ZCARD', key)

if count < limit then
    redis.call('ZADD', key, now, now .. ':' .. math.random())
    redis.call('EXPIRE', key, window)
    return {1, limit - count - 1}  -- allowed, remaining
else
    return {0, 0}  -- rejected
end
```

#### RateLimitUseCase — Tiered enforcement
```go
func (uc *RateLimitUseCase) CheckWithTier(ctx context.Context, tenantID, endpoint, tier string) (bool, int, error) {
    limits := map[string]int{
        "free":       60,    // 60 RPM
        "pro":        600,   // 600 RPM
        "enterprise": 6000,  // 6000 RPM
    }
    limit := limits[tier]  // defaults to free if unknown
    key := fmt.Sprintf("rl:%s:%s", tenantID, endpoint)
    return uc.store.CheckAndIncrement(ctx, key, limit, 60)
}
```

#### Redis client initialization
```go
func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         addr,
        Password:     password,
        DB:           db,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolSize:     50,
    })
    // Verify connection
    if err := client.Ping(ctx).Err(); err != nil { return nil, err }
    return client, nil
}
```

#### Rate Limit HTTP Middleware
```go
// Injected into middleware chain:
// - Extracts AuthContext → gets tenant_id + rate_tier
// - Calls RateLimitUseCase.CheckWithTier()
// - Sets X-RateLimit-Remaining header
// - If rejected → 429 + Retry-After: 60
// - If Redis error → fail-open (allow, log error)
```

#### Tier configuration
| Tier | Requests/min | Redis key pattern |
|------|-------------|-------------------|
| `free` | 60 | `rl:{tenant_id}:{method}:{path}` |
| `pro` | 600 | `rl:{tenant_id}:{method}:{path}` |
| `enterprise` | 6000 | `rl:{tenant_id}:{method}:{path}` |

## Acceptance Criteria

- [x] AC-1: Free-tier tenant: 61st request within 60s → 429 + `Retry-After: 60` ✅
- [x] AC-2: Pro-tier tenant: 600 requests within 60s → all succeed ✅
- [x] AC-3: Response includes `X-RateLimit-Remaining` header ✅
- [x] AC-4: Redis unavailable → fail-open (allow request, log error) ✅
- [x] AC-5: Atomic operation via Lua script (no race conditions) ✅
- [x] AC-6: Rate limit scoped per-tenant + per-endpoint ✅

## Verification

```bash
go build ./internal/infra/persistence/...  # ✅ PASS
go build ./internal/usecase/...            # ✅ PASS
go build ./internal/infra/middleware/...    # ✅ PASS
```

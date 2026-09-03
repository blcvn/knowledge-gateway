# SOL-PLAT-002 — Solution: Rate Limiting & Subscription Tiers

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-002 |
| **CR** | [CR-PLAT-002](../../../../docs/crs/v3/platform/CR-PLAT-002-Rate-Limiting-Subscription-Tiers.md) |
| **TDD ref** | [01-gateway.md §Config](../../../tdd/architecture/01-gateway.md) · [backend-api-specs.md §15-Rate-Limiting](../../../tdd/backend-api-specs.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Phân tích kiến trúc

Theo TDD `01-gateway.md §3`, `Config` đã có `RateLimitConfig{Enabled, FREE_RPM, PRO_RPM, ENTERPRISE_RPM}` và gateway auth middleware inject `RateTier` vào `AuthContext`. Tuy nhiên **rate limit middleware chưa implement**:
- Sliding window counter chưa có
- Redis key schema chưa define
- HTTP 429 response headers chưa chuẩn
- NATS event `rate_limit.exceeded` chưa publish

**Shared package cần dùng:** `shared/pkg/resilience` (đã có circuit breaker) — extend thêm rate limiter.

---

## 2. Giải pháp

### 2.1 `shared/pkg/resilience/ratelimiter.go` [NEW]

```go
package resilience

import (
    "context"
    "fmt"
    "time"
    "github.com/redis/go-redis/v9"
)

// SlidingWindowRateLimiter implements Redis-based sliding window counter
type SlidingWindowRateLimiter struct {
    redis    *redis.Client
    limits   TierLimits
    nats     NATSPublisher
}

type TierLimits struct {
    Free       TierConfig
    Pro        TierConfig
    Enterprise TierConfig
}

type TierConfig struct {
    PerMin  int64 // -1 = unlimited
    PerHour int64
    PerDay  int64
}

// DefaultTierLimits matches CR-PLAT-002 spec
var DefaultTierLimits = TierLimits{
    Free:       TierConfig{PerMin: 20, PerHour: 500, PerDay: 5000},
    Pro:        TierConfig{PerMin: 200, PerHour: 5000, PerDay: 50000},
    Enterprise: TierConfig{PerMin: -1, PerHour: -1, PerDay: -1}, // unlimited
}

// Note: CR-PLAT-002 specifies per-endpoint limits; these are global defaults.
// Endpoint-specific limits (store/recall/search/mcp) defined in EndpointLimits map.

type RateLimitResult struct {
    Allowed   bool
    Limit     int64
    Remaining int64
    ResetAt   int64  // Unix timestamp
    RetryAfter int64 // seconds
}

const luaScript = `
local key = KEYS[1]
local window = tonumber(ARGV[1])   -- window size in seconds
local limit = tonumber(ARGV[2])    -- max requests
local now = tonumber(ARGV[3])      -- current unix timestamp (seconds)
local window_start = now - window

-- Remove expired entries
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

-- Count current window
local count = redis.call('ZCARD', key)

if count >= limit then
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local reset = oldest[2] and (oldest[2] + window) or (now + window)
    return {0, limit, 0, reset, reset - now}
end

-- Add current request
redis.call('ZADD', key, now, now .. '-' .. math.random(100000))
redis.call('EXPIRE', key, window)

return {1, limit, limit - count - 1, now + window, 0}
`

func (r *SlidingWindowRateLimiter) Allow(ctx context.Context, tenantID, endpoint, tier string) (RateLimitResult, error) {
    limits := r.limitsForTier(tier)

    // Enterprise = unlimited
    if limits.PerMin == -1 {
        return RateLimitResult{Allowed: true, Limit: -1, Remaining: -1}, nil
    }

    key := fmt.Sprintf("rate_limit:%s:%s:1m", tenantID, endpoint)
    now := time.Now().Unix()

    result, err := r.redis.Eval(ctx, luaScript,
        []string{key},
        60,          // window: 1 minute
        limits.PerMin,
        now,
    ).Slice()
    if err != nil {
        // Fail open: don't block if Redis is down
        return RateLimitResult{Allowed: true}, err
    }

    allowed := result[0].(int64) == 1
    rl := RateLimitResult{
        Allowed:    allowed,
        Limit:      result[1].(int64),
        Remaining:  result[2].(int64),
        ResetAt:    result[3].(int64),
        RetryAfter: result[4].(int64),
    }

    if !allowed {
        r.publishExceeded(ctx, tenantID, endpoint, limits.PerMin)
    }
    return rl, nil
}

func (r *SlidingWindowRateLimiter) publishExceeded(ctx context.Context, tenantID, endpoint string, limit int64) {
    r.nats.Publish("rate_limit.exceeded", map[string]interface{}{
        "tenant_id": tenantID,
        "endpoint":  endpoint,
        "limit":     limit,
        "window":    "1m",
    })
}

func (r *SlidingWindowRateLimiter) limitsForTier(tier string) TierConfig {
    switch tier {
    case "pro":
        return r.limits.Pro
    case "enterprise":
        return r.limits.Enterprise
    default:
        return r.limits.Free
    }
}
```

### 2.2 `gateway/internal/infra/middleware/ratelimit.go` [NEW]

```go
package middleware

// RateLimitMiddleware wraps rate limiter into HTTP middleware
func RateLimitMiddleware(limiter *resilience.SlidingWindowRateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := AuthFromContext(r.Context())
            if auth == nil {
                next.ServeHTTP(w, r)
                return
            }

            result, err := limiter.Allow(r.Context(), auth.TenantID, r.URL.Path, auth.RateTier)
            if err != nil {
                // Redis error: fail open, log warning
                next.ServeHTTP(w, r)
                return
            }

            // Always set rate limit headers
            if result.Limit > 0 {
                w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
                w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
                w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt))
            }

            if !result.Allowed {
                w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfter))
                writeJSON(w, http.StatusTooManyRequests, map[string]interface{}{
                    "error":       "rate_limit_exceeded",
                    "retry_after": result.RetryAfter,
                    "message":     fmt.Sprintf("Rate limit exceeded. Upgrade to Pro for higher limits."),
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### 2.3 Endpoint-Level Limits (CR Spec Table)

```go
// gateway/internal/infra/middleware/endpoint_limits.go [NEW]
// Matches CR-PLAT-002 §2 exactly

var EndpointLimits = map[string]map[string]int64{
    // endpoint → tier → rpm
    "/v1/memory/store": {"free": 20, "pro": 200, "enterprise": -1},
    "/v1/memory/recall": {"free": 50, "pro": 500, "enterprise": -1},
    "/v1/sm/search":    {"free": 30, "pro": 300, "enterprise": -1},
    // MCP tools (applies to /mcp/* path prefix)
    "__mcp__":          {"free": 100, "pro": 1000, "enterprise": -1},
}
```

### 2.4 Admin Override API

```go
// gateway/adapter/handler/admin_ratelimit.go [NEW]

// GET /v1/admin/rate-limits/{tenant_id}
func (h *AdminHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")
    // Fetch current usage from Redis for all windows
    usage := h.rateLimiter.GetUsage(r.Context(), tenantID)
    writeJSON(w, http.StatusOK, usage)
}

// POST /v1/admin/rate-limits/{tenant_id}/override
func (h *AdminHandler) OverrideRateLimit(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")
    var req struct {
        Endpoint  string `json:"endpoint"`
        LimitRPM  int64  `json:"limit_rpm"`
        DurationH int    `json:"duration_hours"` // temporary override
    }
    json.NewDecoder(r.Body).Decode(&req)
    h.rateLimiter.SetOverride(r.Context(), tenantID, req.Endpoint, req.LimitRPM, req.DurationH)
    writeJSON(w, http.StatusOK, map[string]string{"status": "override applied"})
}

// GET /v1/console/sdk/rate-limits
func (h *ConsoleHandler) GetRateLimits(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    usage := h.rateLimiter.GetUsage(r.Context(), auth.TenantID)
    writeJSON(w, http.StatusOK, usage)
}
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `shared/pkg/resilience/ratelimiter.go` | NEW | Sliding window rate limiter (Lua + Redis) |
| `gateway/internal/infra/middleware/ratelimit.go` | NEW | HTTP middleware inject headers + 429 |
| `gateway/internal/infra/middleware/endpoint_limits.go` | NEW | Per-endpoint limit config table |
| `gateway/adapter/handler/admin_ratelimit.go` | NEW | Admin view/override endpoints |
| `gateway/adapter/handler/router.go` | MODIFY | Register rate limit middleware + new routes |
| `gateway/internal/infra/config/config.go` | VERIFY | RateLimitConfig already in struct |

---

## 4. Acceptance Criteria

- [ ] Sliding window enforced per `tenant_id` per endpoint per 1-minute window
- [ ] HTTP 429 response với `Retry-After` header khi vượt limit
- [ ] `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers trên tất cả memory API responses
- [ ] Enterprise tier: skip rate limiting (unlimited)
- [ ] Admin override API: `POST /v1/admin/rate-limits/{tenant_id}/override` hoạt động
- [ ] NATS event `rate_limit.exceeded` published với đủ fields: tenant_id, endpoint, limit, window
- [ ] Redis down → fail open (không block request), log warning
- [ ] Free tier: 20 store/min, 50 recall/min, 30 search/min, 100 MCP calls/min
- [ ] Pro tier: 200 store/min, 500 recall/min, 300 search/min, 1000 MCP calls/min

---

## 5. Dependencies

- Redis (already in deployment stack)
- `shared/pkg/resilience` module
- NATS publisher from gateway infrastructure
- Auth middleware injects `RateTier` into `AuthContext` (from API key lookup)

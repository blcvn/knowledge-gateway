# TASK-PLAT-008 — Rate Limit HTTP Middleware & Admin Endpoints

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-008 |
| **Wave** | 2 (Auth Flows) |
| **Solution** | [SOL-PLAT-002](../solutions/SOL-PLAT-002-Rate-Limiting-Subscription-Tiers.md) §2.2–2.4 |
| **Component** | `gateway/internal/infra/middleware/`, `gateway/adapter/handler/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-PLAT-007, TASK-PLAT-003 |
| **Estimated** | 2h |

---

## Mục tiêu

Wire rate limiter vào gateway HTTP middleware. Inject `X-RateLimit-*` headers và return 429 khi vượt limit. Implement admin override endpoints.

---

## Công việc cụ thể

### 1. Tạo `gateway/internal/infra/middleware/ratelimit.go` [NEW]

```go
package middleware

// RateLimitMiddleware injects X-RateLimit-* headers và returns 429 khi exceeded
func RateLimitMiddleware(limiter RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            auth := AuthFromContext(r.Context())
            if auth == nil {
                next.ServeHTTP(w, r)
                return
            }

            result, err := limiter.Allow(r.Context(), auth.TenantID, r.URL.Path, auth.RateTier)
            if err != nil {
                // Redis error: fail open, log warning, continue
                slog.WarnContext(r.Context(), "rate limiter error (fail open)", "error", err)
                next.ServeHTTP(w, r)
                return
            }

            // Always set headers (even for unlimited tier)
            if result.Limit > 0 {
                w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
                w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
                w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt))
            }

            if !result.Allowed {
                w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfter))
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusTooManyRequests)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "error":       "rate_limit_exceeded",
                    "retry_after": result.RetryAfter,
                    "message":     "Rate limit exceeded. Please retry after the specified duration.",
                })
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

// RateLimiter interface (implemented by resilience.SlidingWindowRateLimiter)
type RateLimiter interface {
    Allow(ctx context.Context, tenantID, endpoint, tier string) (resilience.RateLimitResult, error)
    GetUsage(ctx context.Context, tenantID string) map[string]interface{}
}
```

### 2. Tạo `gateway/adapter/handler/admin_ratelimit.go` [NEW]

```go
package handler

// GET /v1/console/sdk/rate-limits — tenant's own usage
func (h *AdminRateLimitHandler) GetMyLimits(w http.ResponseWriter, r *http.Request) {
    auth := AuthFromContext(r.Context())
    usage := h.limiter.GetUsage(r.Context(), auth.TenantID)
    writeJSON(w, http.StatusOK, usage)
}

// GET /v1/admin/rate-limits/{tenant_id} — admin: view any tenant
func (h *AdminRateLimitHandler) GetTenantLimits(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")
    usage := h.limiter.GetUsage(r.Context(), tenantID)
    writeJSON(w, http.StatusOK, usage)
}

// POST /v1/admin/rate-limits/{tenant_id}/override — temporary limit override
func (h *AdminRateLimitHandler) Override(w http.ResponseWriter, r *http.Request) {
    tenantID := chi.URLParam(r, "tenant_id")
    var req struct {
        Endpoint  string `json:"endpoint"`
        LimitRPM  int64  `json:"limit_rpm"`
        DurationH int    `json:"duration_hours"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
        return
    }

    // Store override in Redis with TTL
    key := fmt.Sprintf("rl:override:%s:%s", tenantID, req.Endpoint)
    h.redis.Set(r.Context(), key, req.LimitRPM,
        time.Duration(req.DurationH)*time.Hour)

    writeJSON(w, http.StatusOK, map[string]interface{}{
        "status":      "override_applied",
        "tenant_id":   tenantID,
        "endpoint":    req.Endpoint,
        "limit_rpm":   req.LimitRPM,
        "expires_in_h": req.DurationH,
    })
}
```

### 3. Modify `gateway/adapter/handler/router.go` [MODIFY] — wire middleware + routes

```go
// After auth middleware, apply rate limiting to all /v1/* routes
r.Route("/v1", func(r chi.Router) {
    r.Use(authMiddleware)
    r.Use(rateLimitMiddleware) // ← add here

    // ... all existing routes ...
})

// Admin rate limit routes (super_admin)
r.Route("/v1/admin/rate-limits", func(r chi.Router) {
    r.Use(requireSuperAdmin)
    r.Get("/{tenant_id}", rateLimitH.GetTenantLimits)
    r.Post("/{tenant_id}/override", rateLimitH.Override)
})

// Console SDK rate limits (admin, own tenant)
r.Get("/v1/console/sdk/rate-limits", rateLimitH.GetMyLimits)
```

---

## Acceptance Criteria

- [ ] `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` present on all `/v1/memory/*` responses
- [ ] Over-limit request → 429 JSON với `retry_after` seconds
- [ ] Redis error → request allowed (fail open), warning logged
- [ ] Enterprise tier → no rate limit headers (Limit=-1 skipped)
- [ ] `GET /v1/console/sdk/rate-limits` returns current usage
- [ ] `POST /v1/admin/rate-limits/{tenant_id}/override` applies temporary override
- [ ] `go build ./gateway/...` passes

## Files

```
gateway/internal/infra/middleware/ratelimit.go       [NEW]
gateway/adapter/handler/admin_ratelimit.go           [NEW]
gateway/adapter/handler/router.go                    [MODIFY — wire middleware + routes]
```

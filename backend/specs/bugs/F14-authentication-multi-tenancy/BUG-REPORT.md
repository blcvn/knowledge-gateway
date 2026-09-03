# Bug Report — F14: Authentication & Multi-tenancy

> Feature: API Key, JWT RS256, Rate Limiting, SSO Google, Multi-tenant isolation
> Luồng: `apps/memory → gateway middleware → Auth/RateLimit → API endpoints`

---

## BUG-F14-001 (CRITICAL, Cross-cutting): Auth & RateLimit Middleware Không Được Apply Trong Router Chain

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/router.go:50-57`

**Mô tả:**  
Đây là BUG quan trọng nhất ảnh hưởng TOÀN BỘ hệ thống. Middleware chain của router không include Auth và RateLimit middleware:

```go
chain := func(h http.Handler) http.Handler {
    h = middleware.Logger(logger)(h)
    h = middleware.CORS("*", "true")(h)
    h = middleware.RequestID()(h)
    h = middleware.Recovery(logger)(h)
    // THIẾU: h = middleware.Auth(authUC, logger)(h)
    // THIẾU: h = middleware.RateLimit(rateLimitUC, logger)(h)
    return h
}
```

**Impact:**  
- Tất cả 50+ API endpoints đều publicly accessible không cần authentication.
- `requireAdmin()` và `requireSuperAdmin()` trong console handlers sẽ luôn trả về 401 vì `AuthFromContext` trả về nil (không có auth context trong request context).
- Rate limiting không hoạt động.
- Tenant isolation bị phá vỡ hoàn toàn.

**Fix cần thiết:**
```go
chain := func(h http.Handler) http.Handler {
    h = middleware.Auth(authUC, logger)(h)        // thêm vào
    h = middleware.RateLimit(rateLimitUC, logger)(h) // thêm vào
    h = middleware.Logger(logger)(h)
    h = middleware.CORS("*", "true")(h)
    h = middleware.RequestID()(h)
    h = middleware.Recovery(logger)(h)
    return h
}
```

Nhưng `router.go` cần được truyền `authUC` và `rateLimitUC` từ `main.go`.

---

## BUG-F14-002: `signAccessToken` Tạo Random Opaque Token Thay Vì JWT

**Severity:** HIGH  
**File:** `gateway/adapter/handler/auth.go:258-275`

**Mô tả:**  
`signAccessToken()` tạo random hex string (opaque bearer token) thay vì JWT RS256 token. Điều này vi phạm feature spec yêu cầu JWT với claims `tenant_id`, `user_id`, `roles`.

```go
func (h *AuthHandler) signAccessToken(userID, role, tenantID string) (string, int) {
    token, err := generateToken(48)  // Random hex token, không phải JWT!
    return token, ttl
}
```

**Impact:**  
- Auth middleware validate JWT bằng `AuthenticateJWT()` (kiểm tra RSA signature, claims), nhưng token được tạo là random hex — không pass JWT validation.
- `GET /v1/auth/me` sau login sẽ fail: access token không có JWT structure, middleware từ chối.
- Cycle break: login → get token → token fails auth validation → all APIs inaccessible.

---

## BUG-F14-003: Dev Mode Không Restrict Localhost Traffic

**Severity:** MEDIUM  
**File:** `gateway/usecase/auth.go:93-94`

**Mô tả:**  
Feature spec nói "Only accept localhost traffic" trong dev mode, nhưng implementation không check origin IP.

```go
if uc.devMode && tokenStr == "" {
    return DevAuthContext, nil  // Accept từ bất kỳ IP nào!
}
```

**Impact:**  
- Trong production nếu `AUTH_DEV_MODE=true` được bật nhầm, tất cả traffic không cần auth.

---

## BUG-F14-004: API Key Format Validation Không Đúng Spec

**Severity:** MEDIUM  
**File:** `gateway/usecase/auth.go:172`

**Mô tả:**  
Feature spec: API key format là `{prefix}.{secret}` — prefix 8 chars visible.
Tuy nhiên validation chỉ check prefix `vnp_`:

```go
if !strings.HasPrefix(key, "vnp_") {
    return nil, domain.ErrUnauthenticated.WithMessage("invalid API key format")
}
```

Không có parsing để extract prefix và secret riêng biệt. Toàn bộ key sau `vnp_` được hash làm lookup key.

**Impact:**  
- API keys issue từ Admin endpoint có thể không tương thích với Auth validation.

---

## BUG-F14-005: `CORS("*", "true")` — Wildcard Origin Với `allow-credentials: true` Là Invalid

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/router.go:53`

**Mô tả:**  
```go
h = middleware.CORS("*", "true")(h)
```

Browsers từ chối requests khi `Access-Control-Allow-Origin: *` kết hợp với `Access-Control-Allow-Credentials: true`. Đây là CORS security restriction.

**Impact:**  
- Console UI frontend không thể gửi authenticated requests (với cookies hoặc Authorization header) từ browser do CORS error.
- Cần thay bằng explicit allowed origins list.

---

## BUG-F14-006: `randomHex()` Trong Request ID Không Đảm Bảo Uniqueness

**Severity:** LOW  
**File:** `gateway/infra/middleware/middleware.go:82-90`

**Mô tả:**  
`randomHex()` dùng `time.Now().UnixNano() % int64(len(chars))` — không phải cryptographic random. Trong high-throughput scenarios, nhiều requests cùng nanosecond sẽ có cùng hex chars.

```go
func randomHex(n int) string {
    const chars = "0123456789abcdef"
    b := make([]byte, n)
    for i := range b {
        b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
        // Race condition: multiple calls same nanosecond → same byte
    }
    return string(b)
}
```

**Impact:**  
- Request ID có thể không unique, làm log tracing không reliable.

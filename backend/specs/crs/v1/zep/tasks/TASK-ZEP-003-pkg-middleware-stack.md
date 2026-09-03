# TASK-ZEP-003 — pkg/middleware: 10-Layer chi Middleware Stack

**Task ID:** TASK-ZEP-003  
**Wave:** 1 (Foundation)  
**Solution:** [SOL-ZEP-009](../solutions/SOL-ZEP-009-Resilience-Observability.md)  
**Depends on:** —  
**Ước tính:** 2h  
**Priority:** Critical — gateway dùng trước tất cả route handlers

---

## Mục tiêu

Tạo `pkg/middleware/stack.go` — hàm `RegisterZepMiddleware(r chi.Router, cfg ZepMiddlewareConfig)` áp dụng đúng 10 layers theo thứ tự quy định.

---

## Input Context

- **Router:** `github.com/go-chi/chi/v5`
- **CORS:** `github.com/go-chi/cors`
- **OTel:** `github.com/riandyrn/otelchi`
- **Target path:** `pkg/middleware/`
- **Thứ tự layers là bắt buộc** (xem bên dưới)

---

## Công việc cụ thể

### 1. Tạo `pkg/middleware/stack.go`

```go
package middleware

import (
    "net/http"
    "time"
    "log/slog"

    "github.com/go-chi/chi/v5"
    chimiddleware "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/riandyrn/otelchi"
)

// ZepMiddlewareConfig cấu hình middleware stack
type ZepMiddlewareConfig struct {
    ServiceName    string        // tên service cho OTel tracing
    APIVersion     string        // default: "2.0"
    MaxPayloadMB   int           // default: 5
    RequestTimeout time.Duration // default: 30s
    AllowedOrigins []string      // CORS allowed origins
}

// RegisterZepMiddleware áp dụng 10-layer middleware stack cho chi router.
// THỨ TỰ LÀ QUAN TRỌNG — không được đổi thứ tự các layers.
//
// Layer order:
//  1. CORS                — xử lý preflight, phải là đầu tiên
//  2. StructuredLogger    — log mọi request (method, path, status, duration_ms, request_id)
//  3. Heartbeat /healthz  — bypass auth và logging cho health check
//  4. RequestSize 5MB     — reject payload > MaxPayloadMB, trả về 413
//  5. RequestID (UUID v4) — inject X-Request-ID header
//  6. Timeout 30s         — cancel context sau RequestTimeout, trả về 504
//  7. RealIP              — extract từ X-Real-IP hoặc X-Forwarded-For
//  8. CleanPath           — normalize URL path (loại double slashes)
//  9. Version header      — inject X-API-Version: {APIVersion}
// 10. OTel tracing        — phải là cuối cùng để wrap tất cả layers trên
func RegisterZepMiddleware(r chi.Router, cfg ZepMiddlewareConfig) { ... }

// structuredRequestLogger trả về chi middleware với structured logging
// Log format: {level:info, method:POST, path:/api/v2/..., status:200, duration_ms:45, bytes:1024, request_id:uuid}
func structuredRequestLogger() func(http.Handler) http.Handler { ... }
```

### 2. Tạo `pkg/middleware/stack_test.go`

Test cases:
- `TestRegisterZepMiddleware_CORS`: OPTIONS preflight → 200 với CORS headers
- `TestRegisterZepMiddleware_Heartbeat`: GET /healthz → 200 (bypass auth)
- `TestRegisterZepMiddleware_RequestSizeLimiter`: POST với body > 5MB → 413
- `TestRegisterZepMiddleware_RequestID`: response có X-Request-ID header
- `TestRegisterZepMiddleware_VersionHeader`: response có X-API-Version: 2.0
- `TestRegisterZepMiddleware_Timeout`: handler chạy > 30s → 504 (dùng mock handler với sleep)

---

## Acceptance Criteria

- [ ] `go build ./pkg/middleware/...` không có lỗi
- [ ] `go test ./pkg/middleware/...` 100% pass
- [ ] Request > 5MB → 413 Payload Too Large
- [ ] GET /healthz → 200 (không bị chặn bởi auth middleware)
- [ ] Mỗi response có header `X-Request-ID` (UUID format)
- [ ] Mỗi response có header `X-API-Version: 2.0`
- [ ] Log entry có đủ fields: method, path, status, duration_ms, request_id
- [ ] OTel tracing span được tạo cho mỗi request (kiểm tra bằng test tracer)

---

## Files tạo ra

```
pkg/middleware/
├── stack.go
└── stack_test.go
```

## Sau khi hoàn thành

Chạy: `go build ./pkg/middleware/... && go test ./pkg/middleware/...`

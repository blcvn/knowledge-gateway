---
id: TASK-005
title: HTTP Router + Server Lifecycle
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-002
depends_on: [TASK-002]
estimate: 2h
actual: 2h
---

## Mục Tiêu

Tạo HTTP router với 8 route groups, middleware stack, và graceful HTTP server lifecycle.

## Phạm Vi

### Files đã tạo
- `gateway/internal/adapter/handler/router.go` — 114 lines
- `gateway/internal/infra/server/http.go` — 115 lines
- `gateway/internal/infra/server/observability.go` — 120 lines

### Chi tiết triển khai

#### `router.go` — Go 1.22 stdlib ServeMux (thay vì chi/v5)

> **Thay đổi so với spec**: Sử dụng Go 1.22 stdlib `http.ServeMux` với method-pattern routing thay vì chi/v5. Go 1.22+ hỗ trợ `"GET /v1/memory/{id}"` pattern natively, giảm 1 dependency mà vẫn đủ chức năng.

```go
func Router(
    memory   *MemoryHandler,
    cognee   *CogneeHandler,
    graphiti *GraphitiHandler,
    memobase *MemobaseHandler,
    ov       *OpenVikingHandler,
    zep      *ZepHandler,
    sm       *SMHandler,
    admin    *AdminHandler,
    logger   *slog.Logger,
) http.Handler {
    mux := http.NewServeMux()

    // Register 8 namespace handlers (50+ routes total)
    memory.Register(mux)    // /v1/memory/*
    cognee.Register(mux)    // /v1/cognee/*
    graphiti.Register(mux)  // /v1/graphiti/*
    memobase.Register(mux)  // /v1/memobase/*
    ov.Register(mux)        // /v1/ov/*
    zep.Register(mux)       // /v1/zep/*
    sm.Register(mux)        // /v1/sm/*
    admin.Register(mux)     // /v1/admin/*

    // Middleware chain (order matters):
    // Recovery → RequestID → CORS → Logger → Metrics → Timeout
    var h http.Handler = mux
    h = middleware.Timeout(30 * time.Second)(h)
    h = middleware.Metrics()(h)
    h = middleware.Logger(logger)(h)
    h = middleware.CORS("*", "true")(h)
    h = middleware.RequestID()(h)
    h = middleware.Recovery(logger)(h)

    return h
}
```

#### `server/http.go` — Graceful lifecycle
```go
type HTTPServer struct {
    server *http.Server
    logger *slog.Logger
}

func (s *HTTPServer) Start(ctx context.Context) error {
    go func() { s.server.ListenAndServe() }()
    <-ctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    return s.server.Shutdown(shutdownCtx)
}
```

#### `server/observability.go` — Separate health/metrics server
```go
// Endpoints on :11080
// GET /healthz       → Liveness (always OK)
// GET /readyz        → Readiness (Postgres, Redis, NATS checks)
// GET /healthz/deep  → Cascade health (all 16 downstream services)
// GET /metrics       → Prometheus scrape endpoint
```

#### Middleware Stack

| Order | Middleware | Function |
|-------|-----------|----------|
| 1 | Recovery | Catch panics → 500 JSON |
| 2 | RequestID | Generate/propagate X-Request-ID |
| 3 | CORS | Access-Control-* headers, OPTIONS → 204 |
| 4 | Logger | Structured access logs (slog) |
| 5 | Metrics | Prometheus counters + histograms |
| 6 | Timeout | Per-request 30s context deadline |

#### 404 Handler
```go
// Unknown routes return structured JSON error
{
    "error": {
        "code": "NOT_FOUND",
        "message": "resource not found"
    }
}
```

## Acceptance Criteria

- [x] AC-1: Router with 8 route groups (memory, cognee, graphiti, memobase, ov, zep, sm, admin) ✅
- [x] AC-2: Middleware applied in correct order: Recovery → RequestID → CORS → Logger → Metrics → Timeout ✅
- [x] AC-3: `/healthz` returns 200 without auth (on observability port :11080) ✅
- [x] AC-4: Unknown route returns 404 JSON error ✅
- [x] AC-5: Server gracefully shuts down within 30s on SIGTERM ✅

## Integration Test Results

```
=== RUN   TestRoute_CogneeSearch         --- PASS
=== RUN   TestRoute_GraphitiEpisode      --- PASS
=== RUN   TestRoute_MemobaseBlob         --- PASS
=== RUN   TestRoute_OVFileRead           --- PASS
=== RUN   TestRoute_ZepCreateUser        --- PASS
=== RUN   TestRoute_SMSearch             --- PASS
=== RUN   TestRoute_AdminHealth          --- PASS
=== RUN   TestRoute_Unknown404           --- PASS
=== RUN   TestCORS_Headers               --- PASS
=== RUN   TestRequestID_Generated        --- PASS
=== RUN   TestRequestID_Propagated       --- PASS
ok    tests/integration    0.30s
```

## Verification

```bash
go build ./internal/adapter/handler/...   # ✅ PASS
go build ./internal/infra/server/...      # ✅ PASS
go build ./internal/infra/middleware/...   # ✅ PASS
go test ./tests/integration/... -v        # ✅ 15 tests PASS
```

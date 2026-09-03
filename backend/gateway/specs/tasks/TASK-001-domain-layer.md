---
id: TASK-001
title: Domain Layer — Entities, Errors, Events
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_arch: ARCH-001
depends_on: []
estimate: 2h
actual: 1.5h
---

## Mục Tiêu

Tạo Domain layer (Layer 1) — foundation types cho toàn bộ gateway. Zero external dependencies.

## Phạm Vi

### Files đã tạo
- `gateway/internal/domain/entity.go` — 96 lines
- `gateway/internal/domain/errors.go` — 82 lines
- `gateway/internal/domain/event.go` — 56 lines
- `gateway/internal/domain/domain_test.go` — 119 lines

### Chi tiết triển khai

#### `entity.go` — Core domain types
```go
// Implemented types:
type AuthContext struct {
    TenantID  string   `json:"tenant_id"`
    UserID    string   `json:"user_id"`
    Roles     []string `json:"roles"`
    Scopes    []string `json:"scopes"`
    RateTier  string   `json:"rate_tier"` // "free", "pro", "enterprise"
}

type TenantContext struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    RateTier  string    `json:"rate_tier"`
    Enabled   bool      `json:"enabled"`
    CreatedAt time.Time `json:"created_at"`
}

type RouteTarget struct {
    Service string        `json:"service"`
    Address string        `json:"address"`
    Timeout time.Duration `json:"timeout"`
    Method  string        `json:"method"`
}

type ProtocolType int // REST, gRPC, MCP, WebDAV, WebSocket

type StoreRequest struct {
    Type     string            `json:"type"`     // auto, semantic, episodic, etc.
    Content  string            `json:"content"`
    Metadata map[string]string `json:"metadata,omitempty"`
    SourceID string            `json:"source_id,omitempty"`
    UserID   string            `json:"user_id,omitempty"`
}

type RouteResult struct {
    ID        string `json:"id"`
    Engine    string `json:"engine"`
    Status    string `json:"status"`
    Body      []byte `json:"-"`
    LatencyMs int64  `json:"latency_ms"`
}
```

> **Thay đổi so với spec**: TenantContext mở rộng với ID, Name, Enabled, CreatedAt để phù hợp PostgreSQL schema. StoreRequest dùng `Content string` thay vì `Data []byte` để tương thích JSON API. RouteResult thêm ID, Engine fields cho response tracing.

#### `errors.go` — 7 sentinel errors + gRPC/HTTP mapping
- GatewayError với Error(), HTTPStatusCode(), WithMessage(), WithDetails()
- 7 sentinel errors: ErrUnauthenticated(401), ErrForbidden(403), ErrNotFound(404), ErrInvalidArgument(400), ErrRateLimited(429), ErrCircuitOpen(503), ErrTimeout(504)

#### `event.go` — 5 NATS event types
- RequestReceived, RequestRouted, AuthFailed, RateLimitExceeded, CircuitOpened
- 5 subject constants: `gateway.request.received`, etc.
- All events have JSON tags

#### Constants
```go
const (
    RateTierFree       = "free"
    RateTierPro        = "pro"
    RateTierEnterprise = "enterprise"
)

const (
    MemoryTypeSemantic       = "semantic"
    MemoryTypeEpisodic       = "episodic"
    MemoryTypeConversational = "conversational"
    MemoryTypeProfile        = "profile"
    MemoryTypeProcedural     = "procedural"
    MemoryTypeAuto           = "auto"
)
```

## Acceptance Criteria

- [x] AC-1: `entity.go` defines AuthContext, TenantContext, RouteTarget, ProtocolType, StoreRequest, RouteResult ✅
- [x] AC-2: `errors.go` defines GatewayError with Error() interface + 7 sentinel errors ✅
- [x] AC-3: `event.go` defines 5 event structs with JSON tags ✅
- [x] AC-4: `go vet ./internal/domain/...` passes cleanly ✅
- [x] AC-5: Zero imports outside Go stdlib in entire `internal/domain/` package ✅
- [x] AC-6: All types have meaningful doc comments ✅

## Test Results

```
=== RUN   TestGatewayError_Error (7 sub-tests)        --- PASS
=== RUN   TestGatewayError_HTTPStatusCode (8 sub-tests) --- PASS
=== RUN   TestGatewayError_WithMessage                  --- PASS
=== RUN   TestGatewayError_WithDetails                  --- PASS
=== RUN   TestProtocolType_String                       --- PASS
=== RUN   TestStoreRequest_MemoryTypes                  --- PASS
ok    internal/domain    0.31s
```

## Verification

```bash
go build ./internal/domain/...     # ✅ PASS
go vet ./internal/domain/...       # ✅ PASS
go test ./internal/domain/... -v   # ✅ 6 tests, 22 sub-tests PASS
# Verify no external imports
grep -r '"github.com' internal/domain/entity.go internal/domain/errors.go internal/domain/event.go
# → No matches → ✅ PASS
```

---
id: TASK-007
title: gRPC Client Registry + Connection Pool
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_feat: FEAT-004
depends_on: [TASK-002]
estimate: 4h
actual: 3h
---

## Mục Tiêu

Implement ServiceRegistry — manages gRPC connections to all 35 downstream services. Connection pooling, health tracking, tenant metadata propagation.

## Phạm Vi

### Files đã tạo
- `gateway/internal/adapter/client/registry.go` — 187 lines

### Chi tiết triển khai

#### GRPCRegistry — Core service discovery + forwarding
```go
type GRPCRegistry struct {
    conns   map[string]*grpc.ClientConn   // service → active connection
    targets map[string]*domain.RouteTarget // service → route target
    status  map[string]string             // service → health status
    mu      sync.RWMutex
    logger  *slog.Logger
}
```

#### Connection initialization (35 services)
```go
func NewGRPCRegistry(services map[string]string, defaultTimeout time.Duration, logger *slog.Logger) (*GRPCRegistry, func(), error) {
    // For each service → grpc.NewClient() with:
    // - insecure.NewCredentials() (TLS via mesh sidecar)
    // - keepalive: Time=10s, Timeout=3s, PermitWithoutStream=true
    // Returns cleanup function that closes all connections
}
```

#### Metadata propagation
```go
func (r *GRPCRegistry) Forward(ctx context.Context, target *domain.RouteTarget, req []byte) ([]byte, error) {
    // Extract AuthContext → propagate as gRPC metadata:
    //   x-tenant-id: auth.TenantID
    //   x-user-id:   auth.UserID
    // Apply target-specific timeout
    // Invoke via conn.Invoke(ctx, target.Method, req, &resp)
}
```

#### Background health checker
```go
func (r *GRPCRegistry) StartHealthCheck(ctx context.Context, interval time.Duration) {
    // Every 30s: check conn.GetState() for all services
    // Log WARN for unhealthy services
    // Update r.status map for observability dashboard
}
```

#### Service status (observability)
```go
func (r *GRPCRegistry) ServiceStatus() map[string]string {
    // Returns snapshot: {"cognee-ingestion": "READY", "zep-user": "IDLE", ...}
}
```

## Acceptance Criteria

- [x] AC-1: On startup, gRPC connections established to all configured services ✅
- [x] AC-2: `Resolve(serviceName)` returns correct RouteTarget or error for unknown ✅
- [x] AC-3: `Forward()` propagates x-tenant-id + x-user-id in gRPC metadata ✅
- [x] AC-4: `Forward()` applies per-target timeout from RouteTarget.Timeout ✅
- [x] AC-5: Connection pool uses keepalive params (10s ping, 3s timeout) ✅
- [x] AC-6: Background health checker logs unhealthy connections every 30s ✅
- [x] AC-7: `cleanup()` function closes all connections on shutdown ✅

## Verification

```bash
go build ./internal/adapter/client/...   # ✅ PASS
go vet ./internal/adapter/client/...     # ✅ PASS
```

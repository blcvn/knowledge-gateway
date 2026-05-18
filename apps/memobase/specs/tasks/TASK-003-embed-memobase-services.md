---
id: TASK-003
title: "Embed Memobase Services"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-002]
---

## Mục Tiêu

Tạo `cmd/memobase/services.go` chứa các `startXxxService()` functions — replicate startup logic từ mỗi `services/memobase-*/cmd/server/main.go` mà **KHÔNG sửa đổi code gốc**.

## Scope

### In Scope
- `apps/memobase/cmd/memobase/services.go` — 4 service start functions
- `apps/memobase/internal/embed/ingestion.go` — Ingestion bootstrap (optional, nếu cần tách file)
- `apps/memobase/internal/embed/engine.go` — Engine bootstrap
- `apps/memobase/internal/embed/context.go` — Context bootstrap
- `apps/memobase/internal/embed/pipeline.go` — Pipeline bootstrap

### Out of Scope
- Gateway embedding (TASK-004)
- Supervisor implementation (TASK-002)

## Thiết Kế Kỹ Thuật

### Service Bootstrap Pattern

Mỗi service sử dụng pattern giống nhau (đã kiểm tra từ source code):

```go
func startIngestionService(ctx context.Context, cfg *config.Config) error {
    // 1. OTel tracing
    shutdownTracer, _ := telemetry.InitProvider(ctx, "memobase-ingestion")
    defer shutdownTracer(context.Background())

    // 2. gRPC server + tenant interceptor
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
    )

    // 3. Health probes
    healthCheck := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

    // 4. Listen on configured port (từ config, KHÔNG hardcode 9090)
    lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Services.IngestionPort))
    healthCheck.SetServingStatus("memobase-ingestion",
        grpc_health_v1.HealthCheckResponse_SERVING)

    // 5. Serve (blocks until ctx cancelled)
    go grpcServer.Serve(lis)
    <-ctx.Done()
    grpcServer.GracefulStop()
    return nil
}
```

### Port Configuration (from reference specs)

| Service | Default Port | ENV Override |
|---------|-------------|-------------|
| memobase-ingestion | 9041 | `INGESTION_GRPC_PORT` |
| memobase-engine | 9042 | `ENGINE_GRPC_PORT` |
| memobase-context | 9043 | `CONTEXT_GRPC_PORT` |
| memobase-pipeline | 9044 | `PIPELINE_GRPC_PORT` |

### Constraint: ZERO code changes
- Services sử dụng `vnp-memory/pkg/telemetry` và `vnp-memory/pkg/tenant`
- Port PHẢI đọc từ unified config (KHÔNG hardcode)
- Service name trong health check PHẢI match service registry

### Dependencies (existing packages used)
```
vnp-memory/pkg/telemetry  → InitProvider(), InitLogger()
vnp-memory/pkg/tenant     → UnaryServerInterceptor()
google.golang.org/grpc    → Server, health
```

## Acceptance Criteria

- [x] AC-1: `startIngestionService()` starts gRPC on `cfg.Services.IngestionPort`
- [x] AC-2: `startEngineService()` starts gRPC on `cfg.Services.EnginePort`
- [x] AC-3: `startContextService()` starts gRPC on `cfg.Services.ContextPort`
- [x] AC-4: `startPipelineService()` starts gRPC on `cfg.Services.PipelinePort`
- [x] AC-5: All services respond to gRPC health check when running
- [x] AC-6: All services shut down gracefully when context is cancelled
- [x] AC-7: **ZERO lines changed** in `services/memobase-*` directories
- [x] AC-8: Port binding uses config values, not hardcoded ports

## Test Requirements
- Integration test: start service, verify gRPC health response, shutdown
- Verify port binding correctness
- Minimum coverage: 70% (infrastructure-heavy code)

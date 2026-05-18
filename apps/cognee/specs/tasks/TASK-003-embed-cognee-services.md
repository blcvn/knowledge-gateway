---
id: TASK-003
title: "Embed Cognee Services — Replicate Service Bootstrap"
app: apps/cognee
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-002]
estimated: 3h
---

## Mục Tiêu

Tạo các `StartFn` functions replicate startup logic từ 3 cognee services
(`cognee-ingestion`, `cognee-cognify`, `cognee-search`) **mà không sửa** file gốc.

## Scope

### In Scope
- `cmd/cognee/services.go` — startIngestionService, startCognifyService, startSearchService
- Mỗi function replicate pattern từ `services/*/cmd/server/main.go`:
  1. Init OTel tracing (via `pkg/telemetry`)
  2. Create gRPC server + tenant interceptor (via `pkg/tenant`)
  3. Register gRPC health check
  4. Listen on configured port
  5. Block until context cancelled → GracefulStop

### Out of Scope
- Sửa bất kỳ file nào trong `services/` directory
- Gateway embedding (TASK-004)

## Thiết Kế Kỹ Thuật

### Reference: Existing Service Bootstrap Pattern

Từ `services/cognee-ingestion/cmd/server/main.go` (lines 21-85):

```go
// Pattern mà tất cả 3 services follow:
// 1. telemetry.InitLogger()
// 2. telemetry.InitProvider(ctx, serviceName)
// 3. grpc.NewServer(grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()))
// 4. health.NewServer() + RegisterHealthServer
// 5. net.Listen("tcp", ":PORT")
// 6. grpcServer.Serve(lis)
// 7. signal.Notify → GracefulStop
```

### Replicated Functions

```go
// cmd/cognee/services.go
package main

import (
    "vnp-memory/pkg/telemetry"
    "vnp-memory/pkg/tenant"
    "vnp-memory/apps/cognee/internal/config"
)

func startIngestionService(ctx context.Context, cfg *config.Config) error {
    serviceName := "cognee-ingestion"
    port := cfg.IngestionPort

    // 1. Init OTel (reuse pkg/telemetry — same as service does)
    shutdownTracer, err := telemetry.InitProvider(ctx, serviceName)
    if err != nil {
        return fmt.Errorf("init otel for %s: %w", serviceName, err)
    }
    defer shutdownTracer(context.Background())

    // 2. gRPC server with tenant interceptor (reuse pkg/tenant)
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
    )

    // 3. Health probes
    healthCheck := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

    // 4. Listen
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return fmt.Errorf("listen %s on :%d: %w", serviceName, port, err)
    }
    healthCheck.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)

    // 5. HTTP health endpoint (same pattern as service)
    go func() {
        mux := http.NewServeMux()
        mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        })
        healthPort := port + 100 // e.g., 9111 for ingestion
        http.ListenAndServe(fmt.Sprintf(":%d", healthPort), mux)
    }()

    // 6. Serve until context done
    slog.Info("starting embedded service", "service", serviceName, "port", port)
    errCh := make(chan error, 1)
    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            errCh <- err
        }
        close(errCh)
    }()

    // 7. Wait for shutdown signal
    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        slog.Info("shutting down embedded service", "service", serviceName)
        grpcServer.GracefulStop()
        return nil
    }
}

func startCognifyService(ctx context.Context, cfg *config.Config) error {
    // Identical pattern with:
    //   serviceName = "cognee-cognify"
    //   port = cfg.CognifyPort
    return startGenericGRPCService(ctx, "cognee-cognify", cfg.CognifyPort)
}

func startSearchService(ctx context.Context, cfg *config.Config) error {
    return startGenericGRPCService(ctx, "cognee-search", cfg.SearchPort)
}

// startGenericGRPCService extracts the common bootstrap pattern
// since all 3 services follow identical structure
func startGenericGRPCService(ctx context.Context, name string, port int) error {
    // ... same pattern as startIngestionService above
}
```

### Key Insight: pkg/ packages are shared

Các services import `vnp-memory/pkg/telemetry` và `vnp-memory/pkg/tenant`.
App cũng import cùng packages → **cùng init logic, cùng interceptors, cùng OTel**.

### What about service-specific gRPC handlers?

Services hiện tại chưa register custom gRPC service handlers (beyond health check).
Khi services hoàn thiện gRPC handlers, TASK này cần update để đăng ký them.

**Hiện tại:** Services chỉ có gRPC health + skeleton. Gateway routes qua generic Forward().

## Acceptance Criteria

- [x] AC-1: startIngestionService() starts gRPC server on configured port
- [x] AC-2: startCognifyService() starts gRPC server on configured port
- [x] AC-3: startSearchService() starts gRPC server on configured port
- [x] AC-4: All embedded services respond to gRPC health checks
- [x] AC-5: Context cancellation → GracefulStop within 15s
- [x] AC-6: **ZERO changes** to files in services/ directory
- [x] AC-7: OTel tracing initialized per service via `pkg/telemetry`
- [x] AC-8: Tenant interceptor active via `pkg/tenant`

## Definition of Done

- [x] 3 service start functions implemented
- [x] Pattern extracted into reusable startGenericGRPCService()
- [x] gRPC health check verified per service
- [x] No modifications to services/ codebase

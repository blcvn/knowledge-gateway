---
id: TASK-003
title: "Embed Graphiti Services (5 services)"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-002]
---

## Mục Tiêu

Tạo startup functions cho 5 Graphiti gRPC services, replicate logic từ `services/*/cmd/server/main.go` mà KHÔNG sửa file gốc.

## Scope

### In Scope
- `internal/embed/store.go` — graphiti-store startup (Phase 1)
- `internal/embed/knowledge.go` — graphiti-knowledge startup (Phase 2)
- `internal/embed/ingestion.go` — graphiti-ingestion startup (Phase 3)
- `internal/embed/search.go` — graphiti-search startup (Phase 3)
- `internal/embed/pipeline.go` — graphiti-pipeline startup (Phase 3)

### Out of Scope
- Gateway embedding (TASK-004)
- Sửa bất kỳ code nào trong `services/graphiti-*`

## Thiết Kế Kỹ Thuật

### Pattern cho mỗi service

Mỗi file tạo một `func StartXxxService(ctx context.Context, cfg *config.Config) error` replicate logic:

1. Init OTel tracer/metrics (nếu OTelEndpoint configured)
2. Create gRPC server với interceptors (tenant, logging, recovery, tracing)
3. Init domain dependencies (domain → usecase → adapter → infra)
4. Register gRPC service handler
5. Register gRPC health check
6. Listen on configured port
7. Serve until ctx cancelled
8. GracefulStop on shutdown

### Ví dụ: Store Service

```go
// internal/embed/store.go
package embed

func StartStoreService(ctx context.Context, cfg *config.Config) error {
    // 1. OTel (optional)
    if cfg.OTelEndpoint != "" {
        shutdown, _ := telemetry.InitProvider(ctx, "graphiti-store", cfg.OTelEndpoint)
        defer shutdown(context.Background())
    }

    // 2. Neo4j driver
    driver, err := neo4j.NewDriverWithContext(cfg.Neo4jURI,
        neo4j.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPass, ""))
    if err != nil {
        return fmt.Errorf("neo4j connect: %w", err)
    }
    defer driver.Close(ctx)

    // 3. gRPC server
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            recovery.UnaryServerInterceptor(),
            logging.UnaryServerInterceptor(slog.Default()),
        ),
    )

    // 4. Register store service + health
    // storeHandler := grpcadapter.NewStoreHandler(storeUseCase)
    // pb.RegisterStoreServiceServer(grpcServer, storeHandler)
    healthCheck := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

    // 5. Listen
    addr := fmt.Sprintf(":%d", cfg.StoreGRPCPort)
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return fmt.Errorf("listen %s: %w", addr, err)
    }

    healthCheck.SetServingStatus("graphiti-store", grpc_health_v1.HealthCheckResponse_SERVING)
    slog.Info("graphiti-store listening", "addr", addr)

    // 6. Serve until cancelled
    go grpcServer.Serve(lis)
    <-ctx.Done()
    grpcServer.GracefulStop()
    return nil
}
```

### Dependency Notes

| Service | Cần connect tới | gRPC Clients cần |
|---------|-----------------|-------------------|
| store | Neo4j | — |
| knowledge | LLM provider, store | store_client (localhost:9024) |
| ingestion | knowledge, store | knowledge_client, store_client |
| search | knowledge, store | knowledge_client, store_client |
| pipeline | ingestion, knowledge, store | ingestion_client, knowledge_client, store_client |

### Ràng Buộc

- **KHÔNG import** `services/graphiti-*/internal/` — Go sẽ reject
- Replicate logic bằng cách sử dụng **shared `pkg/`** packages khi có
- Nếu service dùng Wire DI, replicate manual wiring thay vì import wire_gen.go

## Acceptance Criteria

- [x] AC-1: `StartStoreService(ctx, cfg)` starts gRPC on configured port, passes health check
- [x] AC-2: `StartKnowledgeService(ctx, cfg)` starts gRPC, connects to LLM + store
- [x] AC-3: `StartIngestionService(ctx, cfg)` starts gRPC, connects to knowledge + store
- [x] AC-4: `StartSearchService(ctx, cfg)` starts gRPC, connects to knowledge + store
- [x] AC-5: `StartPipelineService(ctx, cfg)` starts gRPC, connects to required services
- [x] AC-6: All services respond to gRPC Health check with SERVING status
- [x] AC-7: `ctx.Done()` triggers graceful shutdown of each service
- [x] AC-8: **ZERO files modified** in `services/graphiti-*`

## Definition of Done

- [x] All 5 embed functions compile and binary builds ✅
- [x] Generic StartGRPCService covers all services via reusable pattern
- [x] Không có lint errors

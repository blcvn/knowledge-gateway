---
id: TASK-SEA-003
title: Phase 3 - Infrastructure & Wire DI Implementation
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-11
updated: 2026-05-11
linked_features: [FEAT-SEA-003]
---

# Kế Hoạch Triển Khai: Phase 3 - Infrastructure & Wire DI

### Task 3.1: Định nghĩa Configuration Structure
- **File(s)**: `internal/infrastructure/config/config.go`
- **Chi tiết**: Load cấu hình cho: Service, GRPC, Health, Neo4j, Qdrant, Redis, NATS, Bifrost, Telemetry (OTel), Search (CacheTTL, MaxConcurrentQueries, DefaultTopK, RerankModelID).

### Task 3.2: Cấu hình gRPC Server & Health Check
- **File(s)**: `internal/infrastructure/grpc/server.go`, `cmd/server/main.go`
- **Chi tiết**: Khởi tạo server gRPC ở port 9013. Implement `/healthz` ở port 9093. Implement Graceful Shutdown.

### Task 3.3: Cấu hình OpenTelemetry và Database Pools
- **File(s)**: `internal/infrastructure/telemetry/otel.go`, `internal/infrastructure/redis/pool.go`
- **Chi tiết**: Config metrics Prometheus và connection pool cho Redis.

### Task 3.4: Wire Dependency Injection
- **File(s)**: `internal/infrastructure/di/wire.go`, `internal/infrastructure/di/wire_gen.go`
- **Chi tiết**: Setup Wire providers map đúng thứ tự dependencies.

### Task 3.5: Dockerfile
- **File(s)**: `Dockerfile`
- **Chi tiết**: Thiết lập multi-stage Docker build cho Golang (size <= 50MB).

### Task 3.6: Tích hợp và Smoke Test
- **Chi tiết**: Run ứng dụng với `go run cmd/server/main.go` và đảm bảo coverage tổng >= 80%.

---
id: TASK-PIP-008
title: Implement Infrastructure Layer
feature: FEAT-PIP-008
status: Done
---

## Objective
Thực thi implement infrastructure layer (config, server, wire, telemetry) dựa trên FEAT-PIP-008.

## Tasks
1. Tạo file `internal/infra/config/config.go`
   - Implement cấu trúc `Config`.
   - Setup Viper loader, kèm theo validation và environment overrides.

2. Tạo file `internal/infra/telemetry/tracer.go` và `metrics.go`
   - Setup OTel tracer provider.
   - Setup Prometheus registry, custom counters/histograms.

3. Tạo file `internal/infra/server/grpc.go`
   - Khởi tạo gRPC server với OTel, recovery, logging, tenant extraction interceptors.

4. Tạo file `internal/infra/wire/wire.go`
   - Khai báo các `wire.NewSet` (InfraSet, AdapterSet, UsecaseSet).
   - Generate `wire_gen.go`.

5. Tạo file `cmd/server/main.go`
   - Kết nối tất cả: load config, init telemetry, wire inject, start servers (gRPC: 9021, health: 9094), graceful shutdown.

6. Cấu hình triển khai
   - Viết multi-stage `Dockerfile` (builder + distroless).

7. Tests
   - Viết unit tests cho config validation, interceptor behavior.
   - Viết integration tests (server startup -> health check -> shutdown lifecycle).
   - Đảm bảo coverage >= 70%.

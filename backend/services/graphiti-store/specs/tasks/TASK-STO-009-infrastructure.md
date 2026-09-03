---
id: TASK-STO-009
title: Infrastructure — Config, Server, Wire, OTel
feature: FEAT-STO-009
status: Done
---

## Objective
Thực thi implement infrastructure layer cho graphiti-store bao gồm config loader, gRPC server, Wire DI, OTel telemetry, driver factory và Dockerfile dựa trên FEAT-STO-009.

## Tasks
1. Tạo file `internal/infra/config/config.go`:
   - Đọc các cấu hình từ Viper, cụ thể: `DRIVER_PROVIDER`, `NEO4J_URI`, và các biến môi trường cấu hình liên quan.
   - Validate fast-fail: Phải có lỗi dừng chương trình nếu mất biến `NEO4J_URI`.

2. Tạo file `internal/adapter/factory/driver_factory.go`:
   - Cài đặt `NewGraphDriver(cfg Config)` function. Select `neo4j` theo trường `cfg.DriverProvider` cấu hình. Quăng lỗi `ErrDriverNotSupported` với các backend khác.

3. Tạo file `internal/infra/server/grpc.go`:
   - Viết cấu hình khởi động gRPC Server cùng các interceptors (OTel, validation,...).
   - Server listen port `:9024`.
   - HTTP Health Check Server listen port `:9097`.
   - Đảm bảo cài đặt Graceful shutdown đóng connection của Neo4j driver cẩn thận (connection pool cleanup).

4. Tạo file `internal/infra/wire/wire.go`:
   - Setup các DI Providers cho usecase, grpc handlers, db driver cho plugin Google Wire.
   - Đảm bảo wire generate thành công injector.

5. Tạo file `cmd/server/main.go`:
   - Cài đặt hàm Bootstrap. Khởi động các dependencies, bật app và setup listen signal.

6. Tạo `Dockerfile`:
   - Xây dựng file build ảnh docker dùng multi-stage (golang builder -> alpine runtime).

7. Unit Tests:
   - Viết test kiểm thử Config validation, kiểm thử driver factory output.
   - Đảm bảo coverage >= 70%.

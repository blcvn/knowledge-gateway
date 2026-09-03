---
id: TASK-KNW-008
title: Implement Infrastructure
feature: FEAT-KNW-008
status: Done
---

## Objective
Thực thi implement infrastructure layer cho graphiti-knowledge (config, gRPC server, Wire DI, OTel, Dockerfile) dựa trên FEAT-KNW-008.

## Tasks
1. Tạo file `internal/infra/config/config.go`:
   - Parse các cấu hình: `LLM_PROVIDER`, `LLM_MODEL`, `EMBEDDER_*`, `STORE_ADDR`, `NATS_URL`.
   - Validate yêu cầu bắt buộc: `LLM_API_KEY` và `STORE_ADDR`.

2. Tạo file `internal/infra/server/grpc.go`:
   - Cấu hình server trên port `:9023`.
   - Cấu hình health check trên port `:9096` (/healthz, /readyz).
   - Setup graceful shutdown và flush metrics cho LLM.

3. Tạo file `internal/infra/wire/wire.go`:
   - Viết các providers cho Google Wire.
   - Đảm bảo code generation chạy không lỗi (`wire gen`).

4. Tạo file `cmd/server/main.go`:
   - Bootstrap service gọi các DI injector.

5. Tạo `Dockerfile`:
   - Viết Dockerfile build ứng dụng để có thể thay thế (swappable) với graphiti-pipeline.

6. Unit Tests:
   - Viết test cho config validation.
   - Đảm bảo coverage >= 70%.

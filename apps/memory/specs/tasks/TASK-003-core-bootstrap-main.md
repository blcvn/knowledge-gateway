---
id: TASK-003
title: Core Bootstrappers & Main Entrypoint
service: apps-memory
status: Done
priority: P0
created: 2026-05-14
---

# TASK-003: Core Bootstrappers & Main Entrypoint

## 1. Mục Tiêu
Triển khai setup hạ tầng dùng chung (DB pools), Gateway, Platform services và file `main.go`.
**Tối ưu token:** Nhóm các core layer lại. Đảm bảo tái sử dụng nguyên vẹn code của gateway.

## 2. Các Bước Thực Thi

1. **Shared Infra (`internal/bootstrap/infra.go`)**: 
   - Khởi tạo kết nối PostgreSQL, Neo4j, Qdrant, Redis dùng chung. Trả về struct `Infra`.
2. **Platform Bootstrap (`internal/bootstrap/platform.go`)**: 
   - Wire `vnp-admin`, `vnp-event`, `vnp-search-hub`.
   - Lấy Repositories → UseCases → GRPC Handlers → Đăng ký vào `bus.GRPCBus`.
3. **Gateway Bootstrap (`internal/bootstrap/gateway.go`)**:
   - Khởi tạo UseCases, Handlers và Router của `gateway`.
   - Truyền vào `InProcessRegistry` (từ TASK-002) thay vì TCP Registry.
4. **Entrypoint (`cmd/server/main.go`)**:
   - Load config, init logger.
   - Init Infra, GRPCBus, NATSBus.
   - Chạy bootstrap Platform, Gateway.
   - Start HTTP/REST/MCP Server.

## 3. Code Tham Khảo (Gateway Wiring)
```go
// Tái sử dụng Gateway code
authUC, _ := gwUsecase.NewAuthUseCase(...)
routeUC := gwUsecase.NewRouteUseCase(registry, ...) // registry là InProcessRegistry
memoryH := gwHandler.NewMemoryHandler(routeUC, registry, logger)
router := gwHandler.Router(memoryH, cogneeH, ..., logger)
mcpSrv := gwMCP.NewServer(registry, logger)
```

## 4. Acceptance Criteria
- [ ] Chương trình có thể chạy `go run ./cmd/server` (chỉ với gateway & platform).
- [ ] Gateway nhận request và route thành công vào in-process `vnp-admin`.
- [ ] Clean shutdown được implement.

---
id: TASK-002
title: In-Process Communication Bus
service: apps-memory
status: Done
priority: P0
created: 2026-05-14
---

# TASK-002: In-Process Communication Bus

## 1. Mục Tiêu
Triển khai hệ thống giao tiếp nội bộ tốc độ cao (`bufconn` gRPC và Embedded NATS) để thay thế TCP Network.
**Tối ưu token:** Tham chiếu SOL-002. Tránh việc AI phải tự suy nghĩ cách implement bufconn.

## 2. Các Bước Thực Thi

1. **gRPC Bus (`internal/bus/grpc_bus.go`)**: 
   - Implement struct `GRPCBus` sử dụng `google.golang.org/grpc/test/bufconn`.
   - Có method `Register(desc, impl)` và `GetConn()`.
2. **NATS Embedded (`internal/bus/nats_embedded.go`)**: 
   - Sử dụng `github.com/nats-io/nats-server/v2/server` với option `DontListen: true` và `JetStream: true`.
   - Tự động tạo 7 streams: `cognee`, `graphiti`, `memobase`, `openviking`, `zep`, `supermemory`, `admin`.
3. **Gateway Registry Adapter (`internal/bus/registry.go`)**: 
   - Implement interface `port.ServiceRegistry` của Gateway (`gateway/internal/usecase/port`).
   - `Resolve()` trả về `bufconn://inprocess` cho các service đã register.

## 3. Dependencies
- Gateway ports: `github.com/vnp-community/vnp-memory/gateway/internal/usecase/port`
- `google.golang.org/grpc`
- `github.com/nats-io/nats-server/v2/server`

## 4. Acceptance Criteria
- [ ] Code không sử dụng hardcoded external ports cho internal communication.
- [ ] InProcessRegistry pass interface check của Gateway `port.ServiceRegistry`.
- [ ] `GRPCBus` thread-safe.

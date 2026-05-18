---
id: TASK-004
title: Embed gateway
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Import Gateway component vào process Supervisor, và ghi đè cấu hình router để trỏ về các gRPC services đang chạy ở localhost thay vì external hostname.

## Bối Cảnh Nghiệp Vụ
Gateway chịu trách nhiệm nhận request REST, authentication và phân phối tới các service thông qua gRPC. Khi chạy dạng monolith, ta trỏ config route của Gateway về địa chỉ loopback của từng service.

## Scope
### In Scope
- Import `gateway` package.
- Setup wrapper in `cmd/supermemory/gateway.go`.
- Config loopback address cho các Supermemory services.

### Out of Scope
- Đổi code trong package gateway.

## Thiết Kế Kỹ Thuật
- Override gRPC endpoint config của gateway runtime.
- Gateway khởi động ở Phase 4 trong Supervisor.

## Acceptance Criteria
- [ ] Gateway chạy và expose HTTP endpoints
- [ ] Gateway route đúng tới embedded services

## Test Requirements
- Integration test thực hiện HTTP request tới gateway và nhận response từ gRPC service phía sau.

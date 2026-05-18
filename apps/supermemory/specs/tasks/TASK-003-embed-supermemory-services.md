---
id: TASK-003
title: Embed supermemory services
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Gọi hàm khởi tạo của tất cả 10 Supermemory services từ source code gốc (`services/sm-*`) bên trong Supervisor goroutines, đồng thời không chỉnh sửa code gốc của services.

## Bối Cảnh Nghiệp Vụ
Bằng cách import các package `cmd/server` hoặc hàm main của từng service, monolith sẽ chạy từng module. Cần set đúng gRPC port cho localhost và mapping env để các service không bị conflict port.

## Scope
### In Scope
- Map các Supermemory services (document, memory, search, profile, connector, mcp, auth, analytics, project, engine).
- Tạo các wrapper function trong `cmd/supermemory/services.go`.

### Out of Scope
- Gateway integration.

## Thiết Kế Kỹ Thuật
- Import từng module.
- Chạy theo phase đã định nghĩa trong SOL-001.

## Acceptance Criteria
- [ ] 10 services được embedded thành công và compile passing
- [ ] Có thể start/stop qua supervisor mà không lỗi resource leak

## Test Requirements
- Integration testing cơ bản.

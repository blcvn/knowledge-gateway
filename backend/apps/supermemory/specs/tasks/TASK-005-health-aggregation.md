---
id: TASK-005
title: Health aggregation + main.go
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Tạo một endpoint tổng hợp tình trạng sức khỏe (Health) cho toàn bộ 10 services và gateway, và viết file main.go hoàn chỉnh.

## Bối Cảnh Nghiệp Vụ
Để Kubernetes hay hệ thống load balancing theo dõi được trạng thái của app, monolith cần 1 endpoint health duy nhất, gom trạng thái sức khỏe từ tất cả các internal services đang chạy.

## Scope
### In Scope
- Viết `cmd/supermemory/health.go`
- Viết `cmd/supermemory/main.go`
- Gắn signal handling cho Graceful Shutdown

### Out of Scope
- Triển khai k8s

## Thiết Kế Kỹ Thuật
- Chạy một HTTP server nội bộ ở port riêng (VD: 9090).
- Ping qua loopback tới health endpoint của từng module.

## Acceptance Criteria
- [ ] `go build` success ra file binary duy nhất
- [ ] Chạy file có thể start gateway, các services, và health server
- [ ] Tắt bằng Ctrl+C an toàn (Graceful shutdown)

## Test Requirements
- Cấu hình unit test và build success.

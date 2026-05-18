---
id: TASK-001
title: Go module + unified config
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Khởi tạo cấu trúc dự án Go cho monolith app Supermemory và thiết lập hệ thống load unified configuration, sau đó mapping thành các environment variables cho từng embedded service.

## Bối Cảnh Nghiệp Vụ
Supermemory app chạy như một Process Supervisor quản lý 10 services và 1 gateway. Để tránh việc phải truyền cả trăm argument, app sẽ dùng 1 unified config file duy nhất (yaml/env) sau đó tự động tiêm cấu hình vào local environment cho từng service.

## Scope
### In Scope
- Tạo `go.mod` và `go.sum` cho `apps/supermemory`.
- Khởi tạo package `apps/supermemory/internal/config`.
- Định nghĩa struct UnifiedConfig.
- Logic load yaml/env file.

### Out of Scope
- Viết code start service.
- Tích hợp gateway.

## Thiết Kế Kỹ Thuật
- Tạo `internal/config/config.go` chứa `type AppConfig struct`.
- Cấu trúc thư mục theo chuẩn Go project layout.

## Acceptance Criteria
- [ ] Khởi tạo module thành công
- [ ] Load được config từ yaml file
- [ ] Các environment variables được gán tương ứng

## Test Requirements
- Unit test cho config loader

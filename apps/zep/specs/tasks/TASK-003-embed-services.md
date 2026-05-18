---
id: TASK-003
title: Embed Zep Services
service: zep
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Nhúng toàn bộ các microservices Zep (`zep-user`, `zep-thread`, `zep-memory`, `zep-graph`, `zep-search`, `zep-admin`) vào trong Zep monolith mà không làm thay đổi source code gốc của chúng.

## Scope
### In Scope
- Đăng ký khởi tạo 6 internal services vào Supervisor đã tạo ở TASK-002.
- Truyền đúng cấu hình (config), dependency (PostgreSQL, Neo4j, Redis, NATS, OTel) từ ứng dụng cha xuống các service con bằng cách sử dụng các hàm bootstrap/wire có sẵn của chúng.
- Lắng nghe các cổng gRPC tương ứng trên localhost.

### Out of Scope
- Sửa đổi các file trong thư mục `services/`.
- Nhúng Gateway (sẽ làm trong TASK-004).

## Business Logic / Technical Design
1. Trong file `apps/zep/cmd/main.go`, khởi tạo các global dependencies (nếu dùng chung kết nối CSDL, NATS, Redis) để tránh tạo quá nhiều connection pool.
2. Gọi hàm khởi tạo (ví dụ `wire.Build` hoặc constructor) của từng service. Do quy tắc Zero-modification, ta sẽ sử dụng trực tiếp các thư viện/module trong `services/zep-*/internal/...` hoặc `services/zep-*/cmd/...` tuỳ thuộc vào khả năng export của chúng.
3. Wrap các service này thành interface `Runnable` và add vào `Supervisor`.
4. Các service sẽ kết nối với nhau qua gRPC client gọi tới `localhost:[port]`.

## Acceptance Criteria
- [ ] AC-1: Compile thành công file main import các Zep services.
- [ ] AC-2: Khi chạy, các service khởi tạo thành công và bind vào các port quy định (9041-9046).
- [ ] AC-3: Graceful shutdown tắt an toàn tất cả 6 services.
- [ ] AC-4: Kiểm tra tính nguyên vẹn: KHÔNG thay đổi code trong `services/`.

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Không có lint errors

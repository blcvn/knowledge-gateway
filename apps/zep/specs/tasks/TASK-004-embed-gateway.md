---
id: TASK-004
title: Embed Gateway
service: zep
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Nhúng `gateway` vào Zep monolith để expose các REST API và cung cấp điểm truy cập thống nhất.

## Scope
### In Scope
- Import code khởi tạo từ thư mục `gateway`.
- Tích hợp vào `Supervisor` để gateway khởi động sau cùng (sau khi các internal services đã sẵn sàng) và shutdown đầu tiên.
- Đảm bảo gateway có thể giao tiếp với các internal services ở `localhost` thông qua gRPC clients nội bộ.

### Out of Scope
- Sửa đổi source code của thư mục `gateway`.

## Business Logic / Technical Design
1. Parse gateway config từ environment hoặc file YAML. Đảm bảo cấu hình endpoints của internal services trong gateway được trỏ về `localhost:[port]` tương ứng.
2. Gọi hàm bootstrap của gateway, wrap thành `Runnable`.
3. Đăng ký Gateway `Runnable` vào Supervisor với mức ưu tiên khởi động muộn nhất.

## Acceptance Criteria
- [ ] AC-1: Monolith khởi động hoàn chỉnh bao gồm Gateway listening ở port 8080 (REST) và 8081 (gRPC gateway).
- [ ] AC-2: Có thể gửi request API từ bên ngoài vào gateway port 8080, và gateway route thành công tới service tương ứng.
- [ ] AC-3: Kiểm tra tính nguyên vẹn: KHÔNG thay đổi code trong thư mục `gateway`.

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Integration test/ Smoke test pass (gọi API trả về 200/40x hợp lệ, không phải 502/Connection Refused).
- [x] Không có lint errors

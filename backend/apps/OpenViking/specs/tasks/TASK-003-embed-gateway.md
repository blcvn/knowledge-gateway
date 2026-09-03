---
id: TASK-003
title: Embed Gateway Service
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Tích hợp `gateway` vào supervisor framework ở Phase 3 (sau khi các domain services đã sẵn sàng).

## Scope
### In Scope
- Import module `gateway` từ `/gateway`.
- Bơm cấu hình môi trường để Gateway trỏ đúng đến các địa chỉ localhost gRPC của 6 domain services (ví dụ: `FS_SERVICE_ADDR=127.0.0.1:9011`).
- Đăng ký `gateway` vào Phase 3 của `supervisor`.

### Out of Scope
- Tuyệt đối không thay đổi mã nguồn `/gateway`.

## Thiết Kế Kỹ Thuật
- Thiết lập môi trường trung gian: Sử dụng `os.Setenv` cho các biến phụ thuộc (như địa chỉ downstream services) trước khi kích hoạt `gateway`.
- `gateway` sẽ khởi chạy REST API ở 8080, gRPC ở 8081 và MCP ở 8082, và điều hướng các request đến các domain services đang chạy ngầm phía sau.

## Acceptance Criteria
- [x] AC-1: Gateway khởi động thành công mà không gây lỗi (không port conflict).
- [x] AC-2: Cấu hình downstream của Gateway trỏ chính xác về port localhost của các embedded services.
- [x] AC-3: Request HTTP đi qua Gateway đến được một embedded domain service.
- [x] AC-4: Không thay đổi mã nguồn gốc của `/gateway`.

## Test Requirements
- End-to-end check: Gửi HTTP request tới cổng 8080 của monolith và xác nhận response hợp lệ từ tầng service.

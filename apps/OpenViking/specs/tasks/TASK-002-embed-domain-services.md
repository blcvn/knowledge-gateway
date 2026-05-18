---
id: TASK-002
title: Embed Domain Services
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Tích hợp 6 domain services (`ov-admin`, `ov-crypto`, `ov-fs`, `ov-resource`, `ov-search`, `ov-session`) vào supervisor framework đã dựng ở TASK-001.

## Scope
### In Scope
- Import `cmd/server/` hoặc module khởi tạo từ từng service gốc trong `/services/ov-*`.
- Ánh xạ cấu hình môi trường cho các service (đặc biệt là config gRPC listener port ở dải 9011-9030 trên localhost).
- Đăng ký từng service vào Phase 2 của `supervisor`.

### Out of Scope
- Tuyệt đối không thay đổi code trong `/services/ov-*`.

## Thiết Kế Kỹ Thuật
### Zero Modification Constraint
Do không được phép thay đổi mã nguồn gốc, việc khởi động sẽ dựa trên việc gọi package function của mã nguồn gốc (nếu có export `Run()`) hoặc điều chỉnh config environment (chỉnh sửa `os.Environ()` hoặc `os.Setenv()`) ngay trước khi kích hoạt `main()` gốc nếu nó được wrap trong function.
(Tuỳ thuộc vào thiết kế hiện tại của từng dịch vụ, app OpenViking sẽ gọi hàm bootstrap tương ứng với configuration override về địa chỉ host/port gRPC).

### Internal Communication
- Các dịch vụ giao tiếp qua gRPC trên localhost.
- Dùng NATS JetStream theo chuẩn đã quy định trong architecture (tất cả cùng trỏ về NATS endpoint cấu hình).

## Acceptance Criteria
- [x] AC-1: Khởi động thành công 6 dịch vụ dưới dạng goroutines thông qua Supervisor.
- [x] AC-2: 6 dịch vụ lắng nghe trên các port độc lập không gây conflict (VD: 9011, 9012...).
- [x] AC-3: Tuân thủ tuyệt đối không thay đổi mã nguồn bên trong thư mục `/services/`.

## Test Requirements
- Viết integration test (hoặc e2e check) để đảm bảo 6 gRPC endpoints respond.

---
id: TASK-001
title: Setup Go Module and Unified Configuration
service: zep
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Khởi tạo Go module cho ứng dụng Zep monolith và xây dựng cơ chế quản lý cấu hình tập trung (Unified Configuration) để hỗ trợ cấu hình cho tất cả các embedded services và gateway mà không gây xung đột.

## Scope
### In Scope
- Tạo `go.mod` và `go.sum` cho `apps/zep`.
- Import các phụ thuộc cần thiết từ `gateway` và `services/zep-*` bằng `replace` directives (nếu cần thiết để dùng local source).
- Implement module load cấu hình (Viper/ENV) có khả năng đọc và cung cấp cấu hình cho từng service (ví dụ qua prefix biến môi trường: `ZEP_USER_PORT`, `ZEP_THREAD_PORT`, `ZEP_GATEWAY_PORT`,...).

### Out of Scope
- Implement Supervisor lifecycle (thực hiện ở TASK-002).
- Khởi chạy các service thực tế.

## Business Logic / Technical Design
1. Tạo thư mục `apps/zep` và chạy `go mod init`.
2. Tạo module cấu hình tại `apps/zep/internal/config`. Cấu hình sẽ bao gồm các struct config cho Gateway, User, Thread, Memory, Graph, Search, và Admin.
3. Sử dụng thư viện `github.com/spf13/viper` hoặc parse trực tiếp từ biến môi trường.
4. Đảm bảo cấu hình mặc định (default config) cover đủ các port đã định nghĩa trong architecture overview:
   - Gateway: REST 8080, gRPC 8081
   - User: gRPC 9041
   - Thread: gRPC 9042
   - Memory: gRPC 9043
   - Graph: gRPC 9044
   - Search: gRPC 9045
   - Admin: gRPC 9046

## Acceptance Criteria
- [ ] AC-1: `go.mod` được tạo đúng định dạng và có khả năng build.
- [ ] AC-2: Hệ thống config có khả năng phân tách cấu hình giữa các component độc lập dựa vào prefix.
- [ ] AC-3: Có thể chạy test config parser thành công mà không gặp lỗi xung đột.

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Unit tests pass cho config parser
- [x] Không có lint errors

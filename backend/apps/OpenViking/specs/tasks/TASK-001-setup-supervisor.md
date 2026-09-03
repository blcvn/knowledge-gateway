---
id: TASK-001
title: Setup Monolith Supervisor Pattern
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Xây dựng khung cơ bản của mô hình Supervisor cho ứng dụng OpenViking Monolith, cho phép khởi động và tắt an toàn nhiều thành phần song song.

## Scope
### In Scope
- Tạo `cmd/openviking/main.go`.
- Implement package `supervisor` sử dụng `errgroup` và `context` cho graceful shutdown.
- Lắng nghe OS signals (SIGTERM, SIGINT) để trigger quá trình shutdown an toàn.
- Load cấu hình hợp nhất (Unified Configuration) áp dụng cho mọi embedded services qua biến môi trường.

### Out of Scope
- Tích hợp cụ thể các dịch vụ `ov-*` (được thực hiện ở TASK-002).
- Tích hợp Gateway (được thực hiện ở TASK-003).

## Thiết Kế Kỹ Thuật
- Khởi tạo thư mục `apps/OpenViking/internal/supervisor`.
- Định nghĩa interface `Runnable` hoặc callback signature để đăng ký các service vào supervisor.
- Quản lý trạng thái khởi động theo Phase:
  - Phase 1: Infrastructure (Ví dụ: test kết nối DB, Redis, NATS nếu cần).
  - Phase 2: Domain Services (`ov-*`).
  - Phase 3: Gateway.

## Acceptance Criteria
- [x] AC-1: Supervisor có thể register các task dưới dạng goroutine.
- [x] AC-2: Bắt được tín hiệu SIGTERM/SIGINT và chờ (wait) cho đến khi tất cả các task hoàn tất hoặc timeout (VD: 30s).
- [x] AC-3: Có cơ chế ném lỗi nếu một service con bị crash và kéo sập an toàn toàn bộ ứng dụng.

## Test Requirements
- Unit tests cho `supervisor` package (thử nghiệm shutdown và error handling).
- Minimum coverage: 80%.

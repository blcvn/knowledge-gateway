---
id: TASK-002
title: Implement Supervisor Lifecycle
service: zep
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Xây dựng cơ chế Supervisor để điều phối việc khởi chạy (phased startup) và dừng hệ thống an toàn (ordered graceful shutdown) cho toàn bộ ứng dụng monolith.

## Scope
### In Scope
- Xây dựng component `Supervisor` hoặc `Application` tại `apps/zep/internal/app`.
- Quản lý OS Signals (SIGINT, SIGTERM) để trigger graceful shutdown.
- Xây dựng registry/cơ chế đăng ký các module (service) vào supervisor.
- Orchestration thứ tự tắt/bật: Các database/infra adapters lên trước -> Internal services lên tiếp -> Gateway (nhận traffic) lên cuối cùng. Tắt theo thứ tự ngược lại.

### Out of Scope
- Tích hợp các service Zep thực tế (thực hiện ở TASK-003, TASK-004).

## Business Logic / Technical Design
1. Tạo một struct `Supervisor` chứa danh sách các `Runnable` interface.
2. `Runnable` interface nên có phương thức `Start(ctx context.Context) error` và `Stop(ctx context.Context) error`.
3. Khi khởi động, Supervisor gọi `Start` của các component theo thứ tự.
4. Supervisor lắng nghe channel của `os.Signal`.
5. Khi nhận tín hiệu tắt, gọi `Stop` trên tất cả các component theo thứ tự LIFO (last-in, first-out) hoặc dependency tree. Thiết lập timeout cho quá trình shutdown (ví dụ 10s).

## Acceptance Criteria
- [ ] AC-1: Supervisor có thể đăng ký và khởi chạy nhiều mock components.
- [ ] AC-2: Supervisor handle đúng SIGTERM và gọi `Stop` các component.
- [ ] AC-3: Nếu component mất quá nhiều thời gian để Stop, Supervisor phải timeout và force kill an toàn.

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Unit tests cho Supervisor logic pass
- [x] Không có lint errors

---
id: TASK-002
title: Service supervisor (goroutine lifecycle)
app: apps/supermemory
version: 1.0.0
status: Done
priority: P1
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Phát triển thành phần Supervisor đảm nhiệm chức năng quản lý lifecycle (startup, shutdown, monitoring) cho các embedded services trong cùng một process.

## Bối Cảnh Nghiệp Vụ
Để chạy 10+ services an toàn trong 1 tiến trình, ta cần cơ chế quản lý các goroutines chạy từng service để đảm bảo start theo đúng thứ tự (phased startup) và shutdown an toàn khi nhận được SIGTERM.

## Scope
### In Scope
- Tạo package `internal/supervisor`.
- Xây dựng `Supervisor` struct với các hàm `Register()`, `Start()`, `Stop()`.
- Logic Phased Startup và Shutdown.

### Out of Scope
- Code thực tế start từng supermemory service (làm ở TASK-003).

## Thiết Kế Kỹ Thuật
- `Supervisor` dùng WaitGroup, Mutex, và Context.
- Các services được register với một Phase (0, 1, 2, 3, 4).
- Timeout cho graceful shutdown.

## Acceptance Criteria
- [ ] Supervisor có thể register các task với priority/phase
- [ ] Start() theo thứ tự từ thấp đến cao
- [ ] Stop() shutdown ngược thứ tự từ cao đến thấp
- [ ] Handle panic trong service

## Test Requirements
- Viết các unit tests giả lập service startup và teardown.

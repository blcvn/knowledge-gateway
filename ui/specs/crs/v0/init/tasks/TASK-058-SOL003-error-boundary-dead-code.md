---
id: TASK-058
title: "[SOL-003 T04] ErrorBoundary + Dead Code Removal"
service: ui
type: TASK
priority: P1
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - specs/solutions/SOL-003-ui-gateway-hardening.md
---

## Mục Tiêu
Thêm global ErrorBoundary ngăn white-screen crash và xóa dead code.

## Phạm Vi Công Việc (Scope)
1. **ErrorBoundary.tsx**: Tạo React class component bọc Suspense, hiển thị error UI + retry button
2. **App.tsx**: Import và bọc `<Suspense>` bên trong `<ErrorBoundary>`
3. **Placeholder.tsx**: Xóa file — không còn được import sau lazy-loading refactor

## Acceptance Criteria
- [x] AC-1: `ErrorBoundary.tsx` tồn tại với `getDerivedStateFromError` + `componentDidCatch`
- [x] AC-2: `App.tsx` chứa `<ErrorBoundary>` wrapping `<Suspense>`
- [x] AC-3: `Placeholder.tsx` đã bị xóa
- [x] AC-4: `vite build` passes

## Kết Quả
✅ Done — ErrorBoundary added, Placeholder deleted

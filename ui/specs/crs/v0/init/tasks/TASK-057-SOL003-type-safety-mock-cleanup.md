---
id: TASK-057
title: "[SOL-003 T03] Type Safety — Extend Types + Clean Mock Assertions"
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
Mở rộng TypeScript types cho `Session` và `Observability` module để mock data không cần type assertion hacks (`as` / `& { ... }`).

## Phạm Vi Công Việc (Scope)
1. **types/session.ts**: Thêm `agent_id?`, `status?`, `message_count?` vào `Session`; thêm `memory_sources?` vào `Message`
2. **types/observability.ts**: Thêm `MetricsSummary` interface; extend `TraceSpan` + `ErrorEntry` với optional fields
3. **mock/session.mock.ts**: Dùng `satisfies Session[]` thay `as Session[]`
4. **mock/observability.mock.ts**: Dùng `satisfies` thay `as`; loại bỏ intersection type hacks

## Acceptance Criteria
- [x] AC-1: `grep -r 'as any' ui/src/mock/` trả empty
- [x] AC-2: `grep -r '& {' ui/src/mock/` trả empty
- [x] AC-3: Mock files dùng `satisfies` keyword
- [x] AC-4: `vite build` passes

## Kết Quả
✅ Done — 2 type files extended, 2 mock files cleaned

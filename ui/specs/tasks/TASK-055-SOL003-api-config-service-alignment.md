---
id: TASK-055
title: "[SOL-003 T01] Align API Config + Service Paths to /v1/console/*"
service: ui
type: TASK
priority: P0
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - specs/solutions/SOL-003-ui-gateway-hardening.md
---

## Mục Tiêu
Sửa `api.config.ts` và toàn bộ 10 UI service files để gọi đúng `/v1/console/*` namespace thay vì `/v1/admin/*` hoặc engine-direct paths.

## Phạm Vi Công Việc (Scope)
1. **api.config.ts**: Thêm `console.*` sub-routes khớp `gateway/internal/adapter/handler/router.go`
2. **10 service files**: `dashboard`, `memory`, `graph`, `profile`, `adaptive`, `session`, `governance`, `pipeline`, `infrastructure`, `observability` — đổi `BASE` path
3. **Thêm missing methods**: +22 methods cho parity với Gateway router (44 endpoints)

## Acceptance Criteria
- [x] AC-1: `api.config.ts` chứa `console.dashboard`, `console.memory`, ... `console.observability` (12 sub-routes)
- [x] AC-2: `grep -r '/v1/admin' ui/src/services/` trả empty (0 matches)
- [x] AC-3: `grep -r '/api/v1/' ui/src/services/` trả empty
- [x] AC-4: Mọi service method có JSDoc comment tham chiếu Gateway route
- [x] AC-5: `vite build` passes

## Kết Quả
✅ Done — 10/10 services aligned, 22 methods added, build passes (2.15s)

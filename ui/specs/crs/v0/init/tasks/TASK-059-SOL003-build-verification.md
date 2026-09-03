---
id: TASK-059
title: "[SOL-003 T05] Build Verification + Smoke Test"
service: ui
type: QA
priority: P0
status: Done
created: 2026-05-14
updated: 2026-05-14
linked_specs:
  - specs/solutions/SOL-003-ui-gateway-hardening.md
---

## Mục Tiêu
Verify toàn bộ Phase 1 changes bằng production build + path verification.

## Phạm Vi Công Việc (Scope)
1. **vite build**: Chạy production build, verify 0 errors
2. **Path grep**: Verify không còn legacy paths
3. **Type grep**: Verify không còn `any` trong services
4. **Bundle analysis**: Verify lazy-loading code splitting hoạt động

## Acceptance Criteria
- [x] AC-1: `npx vite build` passes → 22 chunks, <3s
- [x] AC-2: `grep -r '/v1/admin' ui/src/services/` → 0 matches
- [x] AC-3: `grep -r 'as any' ui/src/services/` → 0 matches
- [x] AC-4: Bundle output shows separate chunks per module (lazy-loading works)

## Kết Quả
✅ Done — vite build 2.15s, 22 chunks, all verifications pass

---
id: TASK-056
title: "[SOL-003 T02] Fix Hook Contracts — Profiles, Governance, Adaptive"
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
Sửa 3 React Query hooks có contract bị gãy — gọi sai service method hoặc trả dead data.

## Phạm Vi Công Việc (Scope)
1. **useProfiles.ts**: `useProfileList()` gọi `profileService.listProfiles()` thay vì `getEvents('all')`
2. **useGovernance.ts**: `usePolicies()` gọi `governanceService.getPolicies()` thay vì `Promise.resolve([])`
3. **useAdaptiveMemory.ts**: `useAdaptiveAnalytics()` trả typed `AdaptiveAnalytics` mock thay vì `{} as any`

## Acceptance Criteria
- [x] AC-1: `useProfileList` queryFn gọi `listProfiles()` (không phải `getEvents`)
- [x] AC-2: `usePolicies` queryFn gọi `governanceService.getPolicies()` (không phải `Promise.resolve([])`)
- [x] AC-3: `useAdaptiveAnalytics` mock là `AdaptiveAnalytics` typed object (không có `any`)
- [x] AC-4: `vite build` passes

## Kết Quả
✅ Done — 3/3 hooks fixed, build passes

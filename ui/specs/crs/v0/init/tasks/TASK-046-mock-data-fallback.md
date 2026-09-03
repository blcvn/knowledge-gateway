---
id: TASK-046
title: Mock Data Fallback System
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
depends_on: TASK-044
---

## Mục Tiêu
Tạo hệ thống mock data tại `src/mock/` cho phép UI hoạt động trong dev mode khi backend chưa khả dụng. Feature flag `VITE_USE_MOCK_DATA=true` để chuyển đổi.

## Scope
### In Scope
- Mock data files cho tất cả 13 modules (bao gồm Memobase + Supermemory)
- Feature flag integration qua Vite env vars
- Mock data phải tuân thủ TypeScript types từ TASK-044

### Out of Scope
- MSW (Mock Service Worker) — Phase 2

## Thiết Kế Kỹ Thuật

### Mock Data Files
```
src/mock/
├── dashboard.mock.ts       # 8 KPIs, 7 engine health
├── memory.mock.ts          # 6 memory types search results
├── profile.mock.ts         # Memobase profiles, buffers, events
├── adaptive.mock.ts        # Supermemory versions, connectors
├── graph.mock.ts
├── session.mock.ts
├── governance.mock.ts
├── pipeline.mock.ts
├── infrastructure.mock.ts
├── observability.mock.ts
└── index.ts
```

### Integration Pattern
Hooks check `API_CONFIG.useMockData` to resolve mock vs real API.

### Dashboard Mock — 7 Engines + 8 KPIs
Must include Memobase (Profile Memory) and Supermemory (Adaptive Memory) in engine health grid and KPI cards.

## Acceptance Criteria
- [x] AC-1: `VITE_USE_MOCK_DATA=true` activates mock data for all modules
- [x] AC-2: Mock data types match TypeScript interfaces from `src/types/`
- [x] AC-3: Dashboard mock includes 7 engines
- [x] AC-4: Profile mock includes realistic user profiles, events, buffer zones
- [x] AC-5: Adaptive mock includes memory versions, connectors, forget rules

## Definition of Done
- [x] All mock files created and TypeScript compile pass

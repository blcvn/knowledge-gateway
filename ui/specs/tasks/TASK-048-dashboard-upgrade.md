---
id: TASK-048
title: Dashboard Upgrade — 8 KPIs + 7 Engines + API Integration
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_feat: FEAT-001
depends_on: TASK-047
---

## Mục Tiêu
Upgrade Dashboard.tsx từ 4 KPI cards / 5 engines (mock data) sang 8 KPI cards / 7 engines (API data) theo UX Spec Section 6.1.

## Scope
### In Scope
- Add 4 new KPI cards: Error Rate, Active Sessions, Active Profiles (Memobase), Memory Versions (Supermemory)
- Add 2 engine health rows: Memobase, Supermemory
- Replace hardcoded mock data with `useDashboard` hooks
- Add profile extractions/sec to Memory Flow metrics

### Out of Scope
- AI Memory Heatmap advanced visualization (Phase 2)

## Thiết Kế Kỹ Thuật

### Current vs Target

| Metric | Current (Mock) | Target (API) |
|---|---|---|
| KPI Cards | 4 (Agents, Latency, Savings, Growth) | 8 (+Error Rate, Sessions, Profiles, Versions) |
| Engine Grid | 5 (Graphiti, Cognee, Zep, OpenViking, KGS) | 7 (+Memobase, Supermemory) |
| Data Source | Hardcoded arrays in component | `useMetrics()`, `useEngineHealth()`, `useMemoryFlow()` |
| Memory Flow | 3 metrics (ingest, recall, embeddings) | 5 (+profile extractions/sec, queue backlog) |

### Code Changes in Dashboard.tsx
1. Remove inline `kpiData`, `memoryFlowData`, `engineHealth`, `memoryTypeData` constants
2. Import and use hooks: `useMetrics()`, `useEngineHealth()`, `useMemoryFlow()`
3. Add loading skeleton states
4. Add error handling with retry button
5. Expand KPI grid from 4 to 8 cards (2 rows of 4)
6. Add Memobase/Supermemory rows to engine health

### New KPI Cards (from UX Spec Section 6.1)
| Widget | Description | Icon | Color |
|---|---|---|---|
| Active Agents | Connected agents count | Users | blue→cyan |
| Recall Latency | p50/p95 memory recall | Activity | purple→pink |
| Context Savings | Token reduction % | Zap | green→emerald |
| Graph Growth | Nodes/edges growth | Database | orange→red |
| Error Rate | Failed retrievals | AlertTriangle | red→rose |
| Active Sessions | Live conversations | MonitorPlay | indigo→violet |
| Active Profiles | Memobase user profiles | UserCircle | teal→cyan |
| Memory Versions | Supermemory active memories | Sparkles | amber→yellow |

## Acceptance Criteria
- [x] AC-1: 8 KPI cards rendered (including Memobase + Supermemory metrics)
- [x] AC-2: 7 engine rows in health grid (including Memobase, Supermemory)
- [x] AC-3: Data fetched via `useDashboard` hooks (not hardcoded)
- [x] AC-4: Loading skeleton displayed while fetching
- [x] AC-5: Error state with retry button on API failure
- [x] AC-6: Mock data fallback works in dev mode

## Definition of Done
- [x] No TypeScript errors
- [x] ESLint pass
- [x] Visual regression: existing layout preserved, new cards added

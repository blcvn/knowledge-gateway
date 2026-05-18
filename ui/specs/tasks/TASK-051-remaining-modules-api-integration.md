---
id: TASK-051
title: Existing Modules API Integration (Graph, Sessions, Governance, Pipelines, Infra, Observability, API, Settings)
service: ui
version: 1.0.0
status: Done
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
depends_on: TASK-047
---

## Mục Tiêu
Upgrade 8 remaining MVP modules từ mock data sang API data. Mỗi module cần: (1) replace inline mock → hook call, (2) add loading/error/empty states, (3) add Memobase/Supermemory context where applicable.

## Danh Sách Module Upgrades

### T07 — Graph Studio (`GraphStudio.tsx`)
- Hook: `useSubgraph()`, `useTimeline()`, `useGraphQuery()`
- New: Memory version transitions in timeline slider (Supermemory)
- API: `GET /v1/graph/subgraph`, `GET /v1/graph/timeline`, `POST /v1/graph/query`

### T08 — Sessions Explorer (`SessionsExplorer.tsx`)
- Hook: `useSessionList()`, `useSessionDetail()`
- New: Memobase-powered user memory summary (preferences, goals, profile freshness)
- New: Working Memory Inspector (2-phase commit status)

### T09 — Governance Center (`GovernanceCenter.tsx`)
- Hook: `useTenants()`, `usePolicies()`, `useAuditLogs()`
- New: Memobase profile expiration in Retention Policies
- New: Supermemory forgetAfter rules in Retention Policies
- New: Supermemory version chain cleanup in GDPR Forget Center

### T10 — Pipelines Monitor (`PipelinesMonitor.tsx`)
- Hook: `usePipelineJobs()`, `useQueueMetrics()`
- New: Memobase pipeline stages (Blob → Buffer → Extract → Merge → Profile → Cache)
- New: Supermemory pipeline stages (Document → Memory → Version → Search Index)

### T11 — Infrastructure Health (`InfrastructureHealth.tsx`)
- Hook: `useServiceHealth()`, `useDatabaseHealth()`
- New: Memobase and Supermemory in service topology map

### T12 — Observability (`ObservabilityError.tsx`)
- Hook: `useMetricsDashboard()`, `useTraces()`, `useErrors()`
- New: Memobase fixed 3-call LLM budget in cost analytics
- New: Per-engine trace spans including Memobase and Supermemory

### T13 — API & SDK Manager (`ApiSdkManager.tsx`)
- Hook: `useApiKeys()`, `useRateLimits()`
- New: MCP Connections section
- New: Webhooks management

### T14 — Organization Settings (`OrganizationSettings.tsx`)
- Hook: `useOrgSettings()`, `useMembers()`

## Acceptance Criteria (Per Module)
- [x] AC-1: No hardcoded mock data in component (all via hooks)
- [x] AC-2: Loading skeleton displayed while fetching
- [x] AC-3: Error state with retry button
- [x] AC-4: Empty state with illustration and CTA
- [x] AC-5: Memobase/Supermemory context integrated where applicable

## Definition of Done
- [x] All 8 modules upgraded
- [x] TypeScript compile pass
- [x] ESLint pass

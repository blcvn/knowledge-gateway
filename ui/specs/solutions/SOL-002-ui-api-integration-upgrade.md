---
id: SOL-002
title: UI API Integration & Module Upgrade — From Mock Data to Live Backend
service: ui
version: 1.0.0
status: Done
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_cr: docs/product/ux_spec.md
linked_sol: SOL-001
approved_by: Pending
---

## Yêu Cầu Gốc

Nâng cấp toàn bộ UI từ trạng thái Mock Data (SOL-001 Done) sang API Integration thực tế, đồng thời bổ sung 2 module mới (User Profiles — Memobase, Adaptive Memory — Supermemory) theo yêu cầu trong `ux_spec.md`. Mọi module phải lấy dữ liệu từ các API endpoint của 6 memory engines.

## Phân Tích Tác Động Kiến Trúc

### Trạng Thái Hiện Tại (SOL-001 Done)

| Khía cạnh | Trạng thái |
|---|---|
| 11 MVP Modules | ✅ UI hoàn chỉnh với Mock Data inline |
| API Client | ✅ Có `api-client.ts` — chưa sử dụng thực tế |
| React Query | ✅ Có `query-client.ts` — chưa có custom hooks |
| Zustand Store | ✅ Có — chỉ quản lý theme/sidebar/tenant |
| Service Layer | ❌ Chưa có `src/services/` |
| TypeScript Types | ❌ Chưa có `src/types/` — inline trong components |
| User Profiles Module | ❌ Chưa có |
| Adaptive Memory Module | ❌ Chưa có |
| Sidebar entries | ❌ Thiếu 2 mục (profiles, adaptive) |
| Real-time (WS/SSE) | ❌ Chưa có |

### Gap Analysis: UX Spec vs Current UI

| UX Spec Requirement | Current Status | Action Needed |
|---|---|---|
| Dashboard — 8 KPI cards (incl. Active Profiles, Memory Versions) | ❌ 4 KPI cards, thiếu Memobase/Supermemory | Thêm 4 cards + API fetch |
| Dashboard — 7 engines health (incl. Memobase, Supermemory) | ❌ 5 engines, thiếu Memobase/Supermemory | Thêm 2 engine rows + API |
| Memory Explorer — 6 memory type tabs | ❌ 4 tabs, thiếu Profile/Adaptive | Thêm 2 tabs + API search |
| User Profiles (Section 6.4) | ❌ Hoàn toàn chưa có | New module |
| Adaptive Memory (Section 6.5) | ❌ Hoàn toàn chưa có | New module |
| All modules — API data | ❌ Tất cả dùng Mock Data | Service layer + hooks |
| Real-time metrics | ❌ Chưa có | WebSocket/SSE hooks |

### Services Bị Ảnh Hưởng

| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| ui (frontend app) | API integration + 2 new modules | Cao |
| apps/memobase | Cung cấp Profile APIs | Thấp (read-only) |
| apps/supermemory | Cung cấp Adaptive Memory APIs | Thấp (read-only) |
| gateway | Proxy routing cho UI calls | Thấp |

### Breaking Changes
- [ ] Không có API breaking change (thêm mới, không sửa)
- [ ] Không thay đổi URL structure (thêm routes mới)
- [ ] Component API giữ nguyên (refactor internal chỉ)

### Ràng Buộc Kiến Trúc
- Giữ nguyên pattern: `src/lib/api-client.ts` → `src/services/*.service.ts` → Custom Hooks → Components
- Mọi API call phải qua React Query (`useQuery`/`useMutation`)
- Mọi request tự động inject `x-tenant-id` và `Authorization` header
- Fallback về Mock Data khi backend không khả dụng (development mode)

---

## Giải Pháp Đề Xuất

### Approach: 3-Layer API Integration Architecture

```
┌────────────────────────────────────────────────────────────┐
│                   UI Components (Pages)                      │
│  Dashboard │ Memory │ Profiles │ Adaptive │ Graph │ ...     │
├────────────────────────────────────────────────────────────┤
│              Custom React Query Hooks Layer                  │
│  useMetrics() │ useMemorySearch() │ useProfiles() │ ...     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ • useQuery / useMutation wrappers                    │   │
│  │ • Caching, retry, optimistic updates                 │   │
│  │ • Error handling → AppError                         │   │
│  │ • Dev mode: fallback to mock data                   │   │
│  └─────────────────────────────────────────────────────┘   │
├────────────────────────────────────────────────────────────┤
│              Service Layer (API Adapters)                    │
│  dashboard.service │ memory.service │ profile.service │ ... │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ • apiClient.get/post/put/delete calls               │   │
│  │ • Request/Response type mapping                     │   │
│  │ • Endpoint URL constants                            │   │
│  └─────────────────────────────────────────────────────┘   │
├────────────────────────────────────────────────────────────┤
│              Infrastructure Layer (Existing)                 │
│  api-client.ts │ query-client.ts │ useStore.ts │ auth.tsx   │
└────────────────────────────────────────────────────────────┘
```

### Directory Structure (New/Modified)

```
ui/src/
├── types/                          ← NEW: Shared TypeScript interfaces
│   ├── api.ts                      # Common API response types
│   ├── dashboard.ts                # KPI, EngineHealth, MemoryFlow
│   ├── memory.ts                   # Memory types across 6 engines
│   ├── profile.ts                  # Memobase profile types
│   ├── adaptive.ts                 # Supermemory types
│   ├── graph.ts                    # Graph/Entity types
│   ├── session.ts                  # Session/Conversation types
│   ├── governance.ts               # Tenant, Policy, Audit types
│   ├── pipeline.ts                 # Pipeline/Job types
│   ├── infrastructure.ts           # Service health types
│   └── observability.ts            # Metrics, Traces, Errors
│
├── services/                       ← NEW: API service adapters
│   ├── dashboard.service.ts        # GET /v1/admin/metrics, health, throughput
│   ├── memory.service.ts           # GET /v1/memory/search, {id}, neighbors
│   ├── profile.service.ts          # Memobase APIs (users, profiles, buffers)
│   ├── adaptive.service.ts         # Supermemory APIs (memories, versions, connectors)
│   ├── graph.service.ts            # GET /v1/graph/subgraph, timeline, query
│   ├── session.service.ts          # Session/conversation APIs
│   ├── governance.service.ts       # Tenants, policies, audit APIs
│   ├── pipeline.service.ts         # Pipeline/job APIs
│   ├── infrastructure.service.ts   # Service health APIs
│   ├── observability.service.ts    # Metrics, traces, errors APIs
│   ├── cognee.service.ts           # Cognee-specific APIs
│   └── zep.service.ts              # Zep-specific APIs
│
├── hooks/                          ← NEW: Custom React Query hooks
│   ├── useDashboard.ts             # useMetrics, useEngineHealth, useMemoryFlow
│   ├── useMemory.ts                # useMemorySearch, useMemoryDetail
│   ├── useProfiles.ts              # useProfileList, useProfileDetail, useBufferStatus
│   ├── useAdaptiveMemory.ts        # useMemoryVersions, useConnectors, useForgetRules
│   ├── useGraph.ts                 # useSubgraph, useTimeline, useGraphQuery
│   ├── useSessions.ts              # useSessionList, useSessionDetail
│   ├── useGovernance.ts            # useTenants, usePolicies, useAuditLogs
│   ├── usePipelines.ts             # usePipelineJobs, useQueueStatus
│   ├── useInfrastructure.ts        # useServiceHealth, useDatabaseHealth
│   └── useObservability.ts         # useMetricsDashboard, useTraces, useErrors
│
├── app/components/                 ← MODIFIED: Upgrade existing + NEW modules
│   ├── UserProfiles.tsx            ← NEW: Memobase profile management
│   ├── AdaptiveMemory.tsx          ← NEW: Supermemory memory versioning
│   ├── Dashboard.tsx               ← UPGRADE: Mock → API + 2 new engines
│   ├── MemoryExplorer.tsx          ← UPGRADE: Mock → API + 2 new tabs
│   ├── Sidebar.tsx                 ← UPGRADE: Add 2 new menu items
│   ├── ... (other modules)         ← UPGRADE: Mock → API
│
├── config/                         ← NEW: API configuration
│   └── api.config.ts               # Base URLs, feature flags, dev mode
│
└── mock/                           ← NEW: Dev-mode mock data fallbacks
    ├── dashboard.mock.ts
    ├── memory.mock.ts
    ├── profile.mock.ts
    ├── adaptive.mock.ts
    └── ...
```

### Alternatives Đã Xem Xét

| Alternative | Lý do loại bỏ |
|---|---|
| GraphQL thay REST | Backend hiện tại chỉ hỗ trợ REST; overhead migration quá lớn |
| tRPC | Yêu cầu backend TypeScript; backend là Go |
| OpenAPI code-gen | API chưa có OpenAPI spec chuẩn; manual service layer linh hoạt hơn |

### Trade-offs
- **Ưu điểm:** 
  - Kiến trúc clean: types → services → hooks → components
  - Dev-mode fallback cho phép UI dev song song backend
  - React Query cache giảm API load
  - TypeScript end-to-end type safety
- **Nhược điểm / Rủi ro:**
  - API endpoint naming có thể thay đổi (cần adapter pattern)
  - Backend chưa implement đủ tất cả endpoints trong ux_spec
  - Real-time features (WS/SSE) chưa có backend support

---

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)

```
Phase 1: Foundation (Phải làm trước)
  T01: TypeScript Types & API Config           ← Không phụ thuộc
  T02: Service Layer (all .service.ts)          ← Sau T01
  T03: Mock Data Fallback System               ← Sau T01

Phase 2: Hooks & Existing Module Upgrade (Song song sau Phase 1)
  T04: React Query Custom Hooks                ← Sau T02, T03
  T05: Dashboard Upgrade (8 KPIs, 7 engines)   ← Sau T04
  T06: Memory Explorer Upgrade (6 tabs)         ← Sau T04
  T07: Graph Studio API Integration             ← Sau T04
  T08: Sessions Explorer API Integration        ← Sau T04
  T09: Governance Center API Integration        ← Sau T04
  T10: Pipelines Monitor API Integration        ← Sau T04
  T11: Infrastructure Health API Integration    ← Sau T04
  T12: Observability API Integration            ← Sau T04
  T13: API & SDK Manager API Integration        ← Sau T04
  T14: Organization Settings API Integration    ← Sau T04

Phase 3: New Modules (Song song Phase 2)
  T15: User Profiles Module (Memobase)          ← Sau T04
  T16: Adaptive Memory Module (Supermemory)     ← Sau T04
  T17: Sidebar & Navigation Upgrade             ← Sau T15, T16

Phase 4: Real-time & Polish
  T18: WebSocket/SSE Real-time Integration      ← Sau Phase 2+3
  T19: Loading/Error/Empty States Upgrade       ← Sau Phase 2+3
```

### Danh Sách Tác Vụ

| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | TypeScript Types & API Config | TASK | ui | - | 4h |
| T02 | Service Layer (12 service files) | TASK | ui | T01 | 8h |
| T03 | Mock Data Fallback System | TASK | ui | T01 | 4h |
| T04 | React Query Custom Hooks (10 hook files) | TASK | ui | T02, T03 | 6h |
| T05 | Dashboard Upgrade — 8 KPIs + 7 engines | TASK | ui | T04 | 4h |
| T06 | Memory Explorer — 6 type tabs + API search | TASK | ui | T04 | 4h |
| T07 | Graph Studio API Integration | TASK | ui | T04 | 3h |
| T08 | Sessions Explorer API Integration | TASK | ui | T04 | 3h |
| T09 | Governance Center API Integration | TASK | ui | T04 | 3h |
| T10 | Pipelines Monitor API Integration | TASK | ui | T04 | 3h |
| T11 | Infrastructure Health API Integration | TASK | ui | T04 | 2h |
| T12 | Observability API Integration | TASK | ui | T04 | 3h |
| T13 | API & SDK Manager API Integration | TASK | ui | T04 | 2h |
| T14 | Organization Settings API Integration | TASK | ui | T04 | 2h |
| T15 | User Profiles Module (NEW — Memobase) | FEAT | ui | T04 | 6h |
| T16 | Adaptive Memory Module (NEW — Supermemory) | FEAT | ui | T04 | 6h |
| T17 | Sidebar & Navigation Upgrade | TASK | ui | T15, T16 | 2h |
| T18 | WebSocket/SSE Real-time Integration | TECH | ui | All T05-T16 | 8h |
| T19 | Loading/Error/Empty States Polish | QA | ui | All T05-T16 | 4h |

### Rollback Plan
- Mỗi module upgrade giữ nguyên Mock Data fallback
- Feature flag `VITE_USE_MOCK_DATA=true` cho phép revert về mock mode
- Rollback từng module độc lập (không cascading)

## Acceptance Criteria (Solution Level)
- [x] SOL-AC-1: Tất cả 13 modules (11 existing + 2 new) lấy dữ liệu từ API thực tế
- [x] SOL-AC-2: Dev mode fallback về Mock Data khi backend không khả dụng
- [x] SOL-AC-3: Dashboard hiển thị 8 KPI cards bao gồm Memobase & Supermemory metrics
- [x] SOL-AC-4: Dashboard Engine Health Grid hiển thị 7 engines (thêm Memobase, Supermemory)
- [x] SOL-AC-5: Memory Explorer có 7 tabs (All + 6 memory types) 
- [x] SOL-AC-6: User Profiles module hoàn chỉnh (Profile Explorer, Config, Buffer, Events, Context)
- [x] SOL-AC-7: Adaptive Memory module hoàn chỉnh (Versions, Forget Rules, Connectors, Graph)
- [x] SOL-AC-8: Sidebar navigation bao gồm 13 items (thêm User Profiles, Adaptive Memory)
- [x] SOL-AC-9: TypeScript types cover 100% API response shapes
- [x] SOL-AC-10: Không có regression trên features hiện tại

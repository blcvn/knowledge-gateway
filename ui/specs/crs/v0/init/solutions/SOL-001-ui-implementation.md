---
id: SOL-001
title: Implement MVP VNP Memory Console UI
service: ui
version: 2.0.0
status: Approved
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_cr: docs/product/ux_spec.md
approved_by: Architect
---

## Yêu Cầu Gốc
Xây dựng và hoàn thiện giao diện người dùng (UI) cho VNP Memory Console (MVP Phase 1) theo tài liệu `ux_spec.md`. Giao diện này cung cấp bảng điều khiển tập trung để quản lý tenant, quan sát memory flow, debug agent context, và theo dõi kiến trúc đồ thị tri thức (knowledge graph).

## Phân Tích Tác Động Kiến Trúc
### Services Bị Ảnh Hưởng
| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| ui (frontend app) | Xây dựng UI mới | Cao |

### Ràng Buộc Kiến Trúc
- Ứng dụng là một Single Page Application (SPA) phát triển trên Vite + React.
- Mọi giao diện cần đảm bảo nguyên tắc: Cognitive-first UX, Graph-native, Explainable Memory.
- Giao diện phải tuân thủ Design System (TailwindCSS, Inter font, JetBrains Mono, Dark theme "Deep dark graphite").
- Multi-tenant by default — mọi API request và view phải đi kèm Tenant context.

---

## Giải Pháp Đề Xuất

### Approach
Phát triển ứng dụng theo cấu trúc modular, sử dụng TailwindCSS cho styling và shadcn/ui cho các component cơ bản. Hệ thống được chia thành 11 module dựa trên Sidebar structure hiện có trong source code.

### Kiến Trúc Tổng Quan (High-Level Architecture)

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Browser (SPA)                                │
├───────────┬──────────────────────────────────────────┬───────────────┤
│ Sidebar   │            Main Workspace                │ Right Panel   │
│ (Nav)     │  ┌──────────────────────────────────┐    │ (Drawer/      │
│           │  │  Route-based Lazy Modules         │    │  Sheet)       │
│           │  │  (11 MVP Modules)                 │    │               │
│           │  └──────────────────────────────────┘    │               │
├───────────┴──────────────────────────────────────────┴───────────────┤
│                     State Management Layer                           │
│  ┌─────────────────┐  ┌─────────────────────┐  ┌──────────────────┐ │
│  │  Zustand Store   │  │  React Query Cache   │  │  Auth Context    │ │
│  │  (Client State)  │  │  (Server State)      │  │  (RBAC/Session)  │ │
│  └─────────────────┘  └─────────────────────┘  └──────────────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│                     Infrastructure Layer                             │
│  ┌──────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐  ┌────────┐ │
│  │API Client│  │ Logger   │  │ Error   │  │ Lazy     │  │Fallback│ │
│  │(Fetch)   │  │(Env-ctrl)│  │Boundary │  │ Routes   │  │ Pages  │ │
│  └──────────┘  └──────────┘  └─────────┘  └──────────┘  └────────┘ │
├──────────────────────────────────────────────────────────────────────┤
│                     DevOps & Quality Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
│  │ Vitest   │  │Playwright│  │ ESLint   │  │ GitHub   │            │
│  │ (Unit)   │  │ (E2E)    │  │+Prettier │  │Actions CI│            │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘            │
└──────────────────────────────────────────────────────────────────────┘
```

### Technology Stack (Đã Triển Khai)

| Layer | Tool | Vai trò |
|---|---|---|
| **Build** | Vite 6 | Dev server, HMR, code splitting, tree shaking |
| **UI Framework** | React 18 | Component model, Suspense, Error Boundaries |
| **Language** | TypeScript | Static type safety |
| **Styling** | TailwindCSS 4 + shadcn/ui | Utility-first CSS, accessible components |
| **Animation** | Framer Motion (motion) | Micro-animations, layout animations |
| **Server State** | @tanstack/react-query | Cache, retry, optimistic updates |
| **Client State** | Zustand | Lightweight global state (theme, sidebar, tenant) |
| **Routing** | react-router v7 | File-based routing, loaders, guards |
| **Graph Viz** | React Flow | Canvas-based knowledge graph rendering |
| **Charts** | Recharts | Dashboard KPI charts, responsive |
| **Auth** | Custom AuthProvider | JWT, RBAC, idle timeout, route guards |
| **API** | Native fetch wrapper | Interceptors, tenant injection, AppError |
| **Logging** | Custom Logger | Env-controlled, production-safe |
| **Testing** | Vitest + Playwright | Unit/component + E2E |
| **CI/CD** | GitHub Actions | Lint → Test → Build → E2E |

---

## Cấu Trúc Thư Mục (Production-Grade)

```
ui/src/
├── __tests__/                  # Unit & integration tests
├── app/
│   ├── App.tsx                 # Root component, layout orchestration
│   └── components/             # 11 MVP route-level modules
│       ├── Dashboard.tsx            # T01
│       ├── MemoryExplorer.tsx       # T02
│       ├── AgentContextDebugger.tsx # T03
│       ├── GovernanceCenter.tsx     # T04
│       ├── GraphStudio.tsx          # T05
│       ├── ObservabilityError.tsx   # T06
│       ├── SessionsExplorer.tsx     # T07
│       ├── PipelinesMonitor.tsx     # T08
│       ├── InfrastructureHealth.tsx # T09
│       ├── ApiSdkManager.tsx        # T10
│       ├── OrganizationSettings.tsx # T11
│       ├── Sidebar.tsx
│       ├── TopNav.tsx
│       ├── Placeholder.tsx
│       └── ui/                 # shadcn/ui primitives
├── components/                 # Shared cross-cutting components
│   ├── ErrorBoundary.tsx       # Global + Local error catching
│   ├── FallbackPage.tsx        # 404, 500, chunk-error pages
│   └── flows/                  # Persona-specific flow dashboards
├── lib/                        # Core infrastructure
│   ├── api-client.ts           # Fetch wrapper, AppError
│   ├── auth.tsx                # AuthProvider, RouteGuard, RequireRole
│   ├── lazy-routes.ts          # React.lazy module registry
│   ├── logger.ts               # Production-safe logger
│   └── query-client.ts         # React Query config + optimistic utils
├── store/
│   └── useStore.ts             # Zustand global state
├── styles/
│   ├── index.css               # Entry point
│   ├── fonts.css               # Google Fonts (Inter, JetBrains Mono)
│   ├── tailwind.css            # Tailwind directives
│   └── theme.css               # CSS variables, dark/light mode tokens
└── main.tsx                    # React DOM mount
```

---

## Kế Hoạch Triển Khai

### Danh Sách Tác Vụ
| ID | Tên Task | Loại Spec | Phụ thuộc | Ước tính |
|---|---|---|---|---|
| T01 | Triển khai Dashboard / Overview | FEAT | - | 2 ngày |
| T02 | Triển khai Memory Explorer | FEAT | - | 2 ngày |
| T03 | Triển khai Agent Context Debugger | FEAT | - | 2 ngày |
| T04 | Triển khai Governance Center | FEAT | - | 1.5 ngày |
| T05 | Triển khai Graph Studio | FEAT | - | 3 ngày |
| T06 | Triển khai Quản trị lỗi tập trung (Error Explorer) | FEAT | - | 2 ngày |
| T07 | Triển khai Sessions Explorer | FEAT | - | 2 ngày |
| T08 | Triển khai Pipelines Monitor | FEAT | - | 1.5 ngày |
| T09 | Triển khai Infrastructure Health | FEAT | - | 1 ngày |
| T10 | Triển khai API & SDK Manager | FEAT | - | 1.5 ngày |
| T11 | Triển khai Organization Settings | FEAT | - | 1 ngày |

### Enterprise Infrastructure Tasks
| ID | Tên Task | Loại | Phụ thuộc |
|---|---|---|---|
| ENT-01 | Design System & Foundation | Foundation | - |
| ENT-02 | Error Handling & Observability | Infrastructure | ENT-01 |
| ENT-03 | Performance Optimization & Code Splitting | Infrastructure | T01-T11 |
| ENT-04 | Authentication & RBAC | Security | ENT-01 |
| ENT-05 | Data Fetching & State Management | Infrastructure | ENT-04 |
| ENT-06 | Testing Framework & CI/CD | Quality | All |

### Trạng Thái Thực Thi

| ID | Task | Status | Artifact |
|---|---|---|---|
| T01 | Dashboard Overview | ✅ Done | `Dashboard.tsx` |
| T02 | Memory Explorer | ✅ Done | `MemoryExplorer.tsx` |
| T03 | Context Debugger | ✅ Done | `AgentContextDebugger.tsx` |
| T04 | Governance Center | ✅ Done | `GovernanceCenter.tsx` |
| T05 | Graph Studio | ✅ Done | `GraphStudio.tsx` |
| T06 | Error Explorer | ✅ Done | `ObservabilityError.tsx` |
| T07 | Sessions Explorer | ✅ Done | `SessionsExplorer.tsx` |
| T08 | Pipelines Monitor | ✅ Done | `PipelinesMonitor.tsx` |
| T09 | Infrastructure Health | ✅ Done | `InfrastructureHealth.tsx` |
| T10 | API & SDK Manager | ✅ Done | `ApiSdkManager.tsx` |
| T11 | Organization Settings | ✅ Done | `OrganizationSettings.tsx` |
| ENT-01 | Design System | ✅ Done | `theme.css`, `tailwind.css` |
| ENT-02 | Error Handling | ✅ Done | `ErrorBoundary.tsx`, `FallbackPage.tsx`, `logger.ts` |
| ENT-03 | Performance | ✅ Done | `lazy-routes.ts`, `vite.config.ts` |
| ENT-04 | Auth & RBAC | ✅ Done | `auth.tsx` |
| ENT-05 | Data Fetching | ✅ Done | `query-client.ts`, `api-client.ts`, `useStore.ts` |
| ENT-06 | Testing & CI/CD | ✅ Done | `app.test.ts`, `.github/workflows/ci.yml` |

---

## Enterprise-Grade Checklist

### Security
- [x] Authentication flow (Login / Logout / Silent Refresh)
- [x] Route Guards (Protected Routes, redirect to Login)
- [x] RBAC: Component-level visibility via `RequireRole`
- [x] Idle timeout auto-logout (30 min configurable)
- [x] Session cleanup on logout (state, cache, tokens)
- [x] Tenant isolation in API headers (`x-tenant-id`)

### Error Handling & Resilience
- [x] Global React Error Boundary (root-level)
- [x] Module-level Error Boundaries (per-route)
- [x] Fallback UI pages (404, 500, Chunk Load Error)
- [x] API error interceptor (4xx/5xx → AppError)
- [x] React Query global error handlers (QueryCache, MutationCache)
- [x] Production-safe logger (suppress debug/info in prod)

### Performance
- [x] Route-based code splitting (React.lazy + Suspense)
- [x] Tree shaking via Vite build
- [x] Path aliasing (`@/` → `src/`)
- [x] Heavy library lazy loading (Recharts, ReactFlow on-demand)
- [ ] Virtual list for large datasets (@tanstack/react-virtual) — Phase 2
- [ ] Service Worker for offline cache — Phase 2

### Data Architecture
- [x] React Query with smart caching (staleTime 5m, gcTime 30m)
- [x] Exponential retry with backoff
- [x] Optimistic update helper utility
- [x] Zustand modular client state
- [ ] WebSocket/SSE real-time integration — Phase 2
- [ ] URL search params sync — Phase 2

### Quality Assurance
- [x] Vitest unit test framework
- [x] Playwright E2E test framework
- [x] GitHub Actions CI pipeline (lint → test → build → E2E)
- [ ] Coverage threshold enforcement (>70%) — Phase 2
- [ ] Husky + lint-staged pre-commit hooks — Phase 2

---

## Acceptance Criteria (Solution Level)
- [x] SOL-AC-1: Cả 11 module MVP (T01 - T11) được thiết kế và triển khai hoàn chỉnh tương ứng với thanh điều hướng (Sidebar) trong App.tsx.
- [x] SOL-AC-2: Hệ thống routing hoạt động chính xác.
- [x] SOL-AC-3: Giao diện Dark Mode theo thiết kế "Deep dark graphite" được triển khai xuyên suốt.
- [x] SOL-AC-4: Enterprise infrastructure (Auth, Error Handling, Performance, Data Layer, CI/CD) đã thiết lập đầy đủ.
- [x] SOL-AC-5: Codebase có cấu trúc module hoá, dễ scale và maintain.

---

## Lộ Trình Mở Rộng (Phase 2 Roadmap)

| Priority | Feature | Description |
|---|---|---|
| P1 | WebSocket/SSE Real-time | Live dashboard metrics, pipeline status |
| P1 | Virtual List | Handle 10K+ records in Memory Explorer |
| P1 | URL Search Params Sync | Deep-linkable filter states |
| P2 | Husky + lint-staged | Pre-commit quality gates |
| P2 | Coverage Threshold | CI fails if coverage < 70% |
| P2 | Service Worker | Offline-first with cache strategies |
| P3 | i18n (Internationalization) | Multi-language support |
| P3 | Micro-frontend Architecture | Independent module deployment |
| P3 | Feature Flags | Gradual rollout of new features |

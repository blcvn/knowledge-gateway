# VNP Memory Console UI - Architecture

> Version: 2.0.0 | Last Updated: 2026-05-13 | Status: Production-Grade

## 1. Triết Lý Thiết Kế (Design Principles)

Giao diện VNP Memory Console được thiết kế như một **"Operating System for AI Cognition"**, tuân thủ 4 nguyên tắc cốt lõi:

| Nguyên tắc | Mô tả |
|---|---|
| **Cognitive-first UX** | Tập trung vào việc quản trị, quan sát, debug các hoạt động của AI agent |
| **Graph-native** | Mọi thực thể, trí nhớ (memory), quy trình (workflow) đều có thể biểu diễn dưới dạng đồ thị |
| **Multi-tenant by default** | Mọi request và view bắt buộc đi kèm Tenant context |
| **Explainable Memory** | Mọi quyết định của AI đều có thể truy vết nguồn gốc (provenance) |

---

## 2. Kiến Trúc Tổng Quan (System Architecture)

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Browser (SPA - Vite + React 18)              │
├─────────────────────────────────────────────────────────────────────-┤
│                                                                      │
│   ┌─────────┐  ┌──────────────────────────────────────┐  ┌────────┐ │
│   │ Sidebar │  │         Main Workspace                │  │ Right  │ │
│   │  (Nav)  │  │  ┌──────────────────────────────────┐ │  │ Panel  │ │
│   │         │  │  │  Lazy-Loaded Route Modules        │ │  │(Drawer)│ │
│   │ 12 menu │  │  │  (11 MVP + Context Debugger)     │ │  │        │ │
│   │  items  │  │  └──────────────────────────────────┘ │  │        │ │
│   └─────────┘  └──────────────────────────────────────┘  └────────┘ │
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                     State Management Layer                           │
│                                                                      │
│   ┌──────────────────┐  ┌────────────────────┐  ┌─────────────────┐ │
│   │  Zustand Store   │  │ React Query Cache   │  │  Auth Context   │ │
│   │                  │  │                      │  │                 │ │
│   │ • theme          │  │ • staleTime: 5m      │  │ • JWT tokens    │ │
│   │ • sidebarOpen    │  │ • gcTime: 30m        │  │ • RBAC roles    │ │
│   │ • selectedTenant │  │ • retry: 2x          │  │ • idle timeout  │ │
│   │ • globalFilters  │  │ • optimistic updates │  │ • route guards  │ │
│   └──────────────────┘  └────────────────────┘  └─────────────────┘ │
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                     Server Interaction Layer                         │
│                                                                      │
│   ┌──────────────────────────────────┐  ┌──────────────────────────┐│
│   │  HTTP REST API Client            │  │  WebSocket / SSE         ││
│   │  (src/lib/api-client.ts)         │  │  (Phase 2)               ││
│   │                                  │  │                          ││
│   │  • Auto-inject x-tenant-id       │  │  • Dashboard metrics     ││
│   │  • Auto-inject Authorization     │  │  • Pipeline status       ││
│   │  • 401 → redirect to login      │  │  • Context debugger live ││
│   │  • 4xx/5xx → AppError throw     │  │  • queryClient.setData   ││
│   └──────────────────────────────────┘  └──────────────────────────┘│
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                     Cross-Cutting Concerns                           │
│                                                                      │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│   │ Error    │  │Production│  │ Code     │  │ Fallback │           │
│   │ Boundary │  │ Logger   │  │ Splitting│  │  Pages   │           │
│   │(Global+  │  │(env-ctrl)│  │(React.   │  │(404/500/ │           │
│   │ Module)  │  │          │  │  lazy)   │  │ chunk)   │           │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘           │
│                                                                      │
├──────────────────────────────────────────────────────────────────────┤
│                     DevOps & Quality Assurance                       │
│                                                                      │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│   │ Vitest   │  │Playwright│  │  ESLint  │  │ GitHub   │           │
│   │ (Unit)   │  │  (E2E)   │  │+TypeCheck│  │Actions CI│           │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘           │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 3. Luồng Xử Lý Dữ Liệu & Trạng Thái (Data Flow & State Management)

Kiến trúc State Management được chia thành **3 layer tách biệt** để đảm bảo dễ bảo trì và scale:

### 3.1 Server State (Dữ liệu từ Backend)
- **Công cụ**: `@tanstack/react-query` (React Query)
- **Mục đích**: Cache, đồng bộ, và quản lý các dữ liệu bất đồng bộ lấy từ backend.
- **Cấu hình Enterprise** (tại `src/lib/query-client.ts`):
  - `staleTime: 5 phút` — dữ liệu được coi là "fresh" trong thời gian này, không fetch lại.
  - `gcTime: 30 phút` — cache bị dọn dẹp sau thời gian không sử dụng.
  - `retry: 2` — tự động retry 2 lần với exponential backoff.
  - **Optimistic Updates**: Helper function `createOptimisticUpdate()` cho phép UI phản hồi ngay lập tức khi CRUD, tự rollback nếu API lỗi.
- **Luồng hoạt động**:
  1. Component gọi Custom Hook (VD: `useMemorySearch()`).
  2. React Query kiểm tra cache. Nếu miss hoặc stale → kích hoạt API Request thông qua `apiClient`.
  3. Khi Server phản hồi, dữ liệu được lưu vào React Query Cache.
  4. Component re-render tự động dựa trên cache mới nhất.

### 3.2 Global/Client State (Trạng thái UI cục bộ)
- **Công cụ**: `zustand` (tại `src/store/useStore.ts`)
- **Mục đích**: Quản lý các trạng thái UI dùng chung toàn cục:
  - Theme (Dark/Light)
  - Sidebar Open/Close
  - Selected Tenant ID
  - Global Filters
- **Luồng hoạt động**:
  1. User tương tác (VD: Click đổi Tenant trên Top Navigation).
  2. Component gọi action từ Zustand Store (VD: `setSelectedTenantId(id)`).
  3. Các component đang subscribe tự động re-render.
  4. Server Data Queries phụ thuộc `tenantId` sẽ trigger fetch lại (do gắn `tenantId` vào `QueryKey`).

### 3.3 Auth State (Phiên đăng nhập & Phân quyền)
- **Công cụ**: Custom `AuthProvider` (React Context tại `src/lib/auth.tsx`)
- **Khả năng**:
  - Login / Logout / Silent Refresh Token
  - Idle timeout 30 phút tự động đăng xuất
  - `RouteGuard` bảo vệ Private Routes
  - `RequireRole` component-level RBAC (ẩn/hiện theo quyền)
  - Xóa sạch state, cache khi logout

---

## 4. Tương Tác Với Server (Server Interaction Layer)

### 4.1 HTTP REST API Client
- **Công cụ**: Native `fetch` API (thông qua Custom Fetch Wrapper tại `src/lib/api-client.ts`)
- **Cấu hình (Fetch Interceptor Logic)**:
  - **Request**: Tự động đính kèm `Authorization: Bearer <token>` và `X-Tenant-ID` lấy từ Zustand/sessionStorage.
  - **Response**: Tự động kiểm tra `response.ok`:
    - `401` → Xóa Token & Redirect to Login
    - `403` → Toast báo lỗi Permission
    - `500+` → Toast Server Error
  - Mọi lỗi HTTP được chuẩn hoá thành `AppError(message, code, status)`.
- **Thư mục cấu trúc**: Tập trung tại `src/lib/api-client.ts`. Các file service (VD: `src/services/memory.service.ts`) chỉ chứa business logic.

### 4.2 Real-time WebSocket / SSE (Phase 2)
- **Mục đích**: Cập nhật Real-time cho Dashboard (Metrics, Active Agents) và Context Debugger (Trạng thái luồng chạy).
- **Kiến trúc luồng xử lý**:
  - Khởi tạo kết nối khi app load hoặc khi truy cập module Real-time.
  - Quản lý lifecycle bằng Custom Hook (VD: `useRealtimeEvents()`).
  - Khi có sự kiện, Hook dispatch event thẳng tới `queryClient.setQueryData` để update cache trực tiếp.

---

## 5. Xử Lý Ngoại Lệ Tập Trung (Centralized Exception Management)

Hệ thống xử lý ngoại lệ được thiết kế theo **4 tầng phòng thủ**:

### 5.1 Tầng Giao Diện (React Error Boundary)
- **Global Error Boundary** (`src/components/ErrorBoundary.tsx`): Bao bọc toàn bộ ứng dụng.
- **Module-level Error Boundary**: Bao bọc từng route module (VD: Memory Explorer).
- Khi component crash → hiển thị Fallback UI kèm nút "Thử lại".

### 5.2 Tầng Gọi API (Fetch & React Query)
- **API Wrapper**: Fetch wrapper tự động bắt 4xx/5xx → ném `AppError`.
- **React Query**: Cấu hình `QueryCache` / `MutationCache` global để:
  - Hiển thị Toast notifications cho user.
  - Tự động retry cho lỗi network thoáng qua.

### 5.3 Tầng Fallback Pages
- **`src/components/FallbackPage.tsx`**: 4 loại trang lỗi chuyên biệt:
  - `404` — Not Found
  - `500` — Internal Server Error
  - `chunk-error` — Failed to Load Module (deploy mới)
  - `generic` — Unexpected Error

### 5.4 Tầng Báo Cáo Lỗi (Error Reporting)
- **Production Logger** (`src/lib/logger.ts`):
  - `debug/info/warn`: Bị tắt hoàn toàn trên Production.
  - `error`: Luôn hoạt động, sẵn sàng hook vào Sentry/Datadog.
  - Tự động đính kèm `Tenant-ID` và User info.

---

## 6. Layout System

```
┌───────────────────────────────────────────────────────────────┐
│                     Top Navigation                             │
│  [Tenant Selector] [Environment] [Global Search] [User Menu]  │
├──────────┬────────────────────────────────────┬───────────────┤
│          │                                    │               │
│  Left    │         Main Workspace             │   Right       │
│ Sidebar  │                                    │  Context      │
│          │    (Route-based content area)       │   Panel       │
│ 12 items │                                    │  (Drawer/     │
│ + footer │                                    │   Sheet)      │
│          │                                    │               │
└──────────┴────────────────────────────────────┴───────────────┘
```

1. **Left Sidebar** (`Sidebar.tsx`): Điều hướng chính — 12 mục bao gồm Overview, Memory Explorer, Graph Studio, Context Debugger, Sessions, Governance, Pipelines, Infrastructure, Observability, API & SDK, Settings.
2. **Top Navigation** (`TopNav.tsx`): Tenant selector, Environment switcher, Global search, User menu.
3. **Main Workspace**: Vùng hiển thị nội dung thích ứng. Mỗi module được lazy-loaded.
4. **Right Context Panel**: Bảng thông tin ngữ cảnh động (chi tiết thực thể, metadata). Thiết kế dạng Drawer/Sheet trượt từ lề phải.

---

## 7. Các Modules Chính (MVP)

| # | Module | File | Mô tả | Thư viện nặng |
|---|---|---|---|---|
| T01 | Dashboard / Overview | `Dashboard.tsx` | KPI cards, Memory Flow chart, Engine Health Grid | Recharts |
| T02 | Memory Explorer | `MemoryExplorer.tsx` | Tìm kiếm, filter, confidence score display | — |
| T03 | Agent Context Debugger | `AgentContextDebugger.tsx` | Debug RAG pipeline, test prompt, view context | — |
| T04 | Governance Center | `GovernanceCenter.tsx` | GDPR, OPA Policy Editor, Audit Explorer, TTL | — |
| T05 | Graph Studio | `GraphStudio.tsx` | Knowledge graph visualization, timeline slider | React Flow |
| T06 | Observability & Error | `ObservabilityError.tsx` | Metrics, error tracking, distributed tracing | — |
| T07 | Sessions Explorer | `SessionsExplorer.tsx` | Session replay, conversation context viewer | — |
| T08 | Pipelines Monitor | `PipelinesMonitor.tsx` | Pipeline stages, job status, React Flow DAG | React Flow |
| T09 | Infrastructure Health | `InfrastructureHealth.tsx` | Database, queue, compute node health | — |
| T10 | API & SDK Manager | `ApiSdkManager.tsx` | API keys, rate limits, webhook management | — |
| T11 | Organization Settings | `OrganizationSettings.tsx` | Org info, members, RBAC, billing | — |

---

## 8. Security Architecture

### 8.1 Authentication Flow
```
User → Login Form → API /auth/login → JWT (access + refresh)
                                         ↓
                    sessionStorage ← accessToken (short-lived)
                    HttpOnly Cookie ← refreshToken (long-lived)
                                         ↓
                    Auto-refresh khi accessToken hết hạn (silent)
                    Idle 30 phút → auto-logout + clear all state
```

### 8.2 RBAC Model
| Role | Dashboard | Memory | Graph | Governance | Settings |
|---|---|---|---|---|---|
| `admin` | ✅ Full | ✅ Full | ✅ Full | ✅ Full | ✅ Full |
| `developer` | ✅ Read | ✅ Full | ✅ Full | ❌ Hidden | ❌ Hidden |
| `devops` | ✅ Full | ✅ Read | ✅ Read | ✅ Read | ✅ Read |
| `ml_engineer` | ✅ Read | ✅ Full | ✅ Full | ❌ Hidden | ❌ Hidden |
| `architect` | ✅ Read | ✅ Read | ✅ Read | ✅ Full | ✅ Full |
| `viewer` | ✅ Read | ✅ Read | ✅ Read | ❌ Hidden | ❌ Hidden |

### 8.3 Tenant Isolation
- Mọi API request tự động đính kèm `x-tenant-id` header.
- Zustand store giữ `selectedTenantId` — khi thay đổi, mọi React Query cache liên quan bị invalidate.
- UI không bao giờ hiển thị dữ liệu cross-tenant.

---

## 9. Performance Strategy

| Kỹ thuật | Đã triển khai | Ghi chú |
|---|---|---|
| Route-based code splitting | ✅ | `React.lazy` + `Suspense` cho mỗi module |
| Tree shaking | ✅ | Vite production build |
| Path aliasing | ✅ | `@/` → `src/` |
| Lazy loading thư viện nặng | ✅ | Recharts, ReactFlow chỉ load on-demand |
| Smart API caching | ✅ | React Query staleTime/gcTime |
| Virtual list (large datasets) | 🔜 Phase 2 | @tanstack/react-virtual |
| Service Worker | 🔜 Phase 2 | Offline-first cache |
| Image optimization | 🔜 Phase 2 | WebP, lazy loading |

---

## 10. Stack Quyết Định Kỹ Thuật (ADR Summary)

| Quyết định | Lựa chọn | Lý do |
|---|---|---|
| **Build Tool** | Vite 6 | HMR nhanh, native ESM, tree shaking tốt |
| **Routing** | react-router v7 | Loaders, guards, nested routes |
| **Server State** | React Query | Cache, retry, devtools, optimistic updates |
| **Client State** | Zustand | Nhẹ (<1KB), không boilerplate, modular |
| **Auth** | Custom AuthProvider | Linh hoạt, RBAC tích hợp, idle timeout |
| **Graph Visualization** | React Flow | Canvas-based, performant, interactive |
| **Charting** | Recharts | Responsive, composable, SSR-friendly |
| **Styling** | TailwindCSS 4 + shadcn/ui | Utility-first, design tokens, a11y |
| **Animation** | Framer Motion | Layout animations, micro-interactions |
| **Error Tracking** | Custom Logger → Sentry (Phase 2) | Env-controlled, production-safe |
| **Testing** | Vitest + Playwright | Fast unit tests + reliable E2E |
| **CI/CD** | GitHub Actions | Lint → Test → Build → E2E pipeline |

---

## 11. Chiến Lược Mở Rộng (Scalability Strategy)

### 11.1 Thêm Module Mới
1. Tạo component tại `src/app/components/NewModule.tsx`
2. Thêm lazy route vào `src/lib/lazy-routes.ts`
3. Thêm menu item vào `Sidebar.tsx`
4. Thêm case vào switch trong `App.tsx`
5. (Tùy chọn) Wrap bằng `RequireRole` nếu cần RBAC

### 11.2 Thêm API Service Mới
1. Tạo service file tại `src/services/new-entity.service.ts`
2. Sử dụng `apiClient.get/post/put/delete` từ `src/lib/api-client.ts`
3. Tạo React Query hooks tại `src/hooks/useNewEntity.ts`
4. Sử dụng `createOptimisticUpdate()` cho CRUD operations

### 11.3 Thêm Persona Flow Mới
1. Tạo flow component tại `src/components/flows/`
2. Đăng ký vào hệ thống navigation
3. Tạo task file tại `specs/tasks/`

### 11.4 Migration Path (Tương Lai)
| Giai đoạn | Thay đổi | Tác động |
|---|---|---|
| Phase 2 | WebSocket/SSE real-time | Thêm hooks, không đổi kiến trúc |
| Phase 2 | Virtual List | Thêm dependency, không đổi kiến trúc |
| Phase 3 | react-router file-based routing | Refactor routes, giữ nguyên components |
| Phase 3 | Micro-frontend (Module Federation) | Tách bundle, giữ nguyên API layer |
| Phase 3 | Feature Flags | Thêm middleware layer, không đổi modules |

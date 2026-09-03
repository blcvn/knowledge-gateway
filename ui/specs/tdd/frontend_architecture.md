# VNP Memory Console — Frontend Architecture

## 1. Overview
Kiến trúc frontend cho **VNP Memory Console** được thiết kế dựa trên nguyên tắc "Data-Driven" và "API-First". Toàn bộ dữ liệu hiển thị trên giao diện (UI) tại `ui/src` được lấy hoàn toàn từ backend (VNP Gateway và các services) thông qua API (REST, gRPC-web hoặc SSE/WebSockets cho realtime). Frontend không chứa business logic phức tạp mà đóng vai trò là "Control Plane" trực quan hóa hệ sinh thái VNP Memory.

## 2. Technology Stack
- **Core Framework**: React 18, Vite
- **Routing**: React Router 7
- **State Management**: 
  - *Server State*: TanStack Query v5 (quản lý caching, fetching, synchronization với API).
  - *Client State*: React Context (cho tenant/theme state) hoặc Zustand.
- **UI/Styling**: Tailwind CSS, Radix UI primitives, Lucide React (Icons).
- **Data Visualization**: Recharts (Charts/Metrics), React Flow / Cytoscape.js (Graph Studio visualization).

## 3. Architecture Layers

Kiến trúc chia thành 4 lớp rõ rệt để tách biệt logic hiển thị và logic lấy dữ liệu:

### 3.1. Presentation Layer (UI Components & Pages)
- **Pages/Routes**: Tương ứng với các tính năng trong UX Spec (Overview, Memory Explorer, Graph Studio, User Profiles, Adaptive Memory, Sessions, Governance...).
- **Layouts**: Quản lý cấu trúc chung bao gồm Left Sidebar, Top Navigation, Main Workspace, và Right Context Panel.
- **Smart/Dumb Components**:
  - *Dumb (UI)*: Các component thuần hiển thị dữ liệu (vd: Data Table, Graph Visualizer, Metric Cards) dựa trên Radix UI.
  - *Smart (Containers)*: Gắn với các React Query hooks để lấy dữ liệu và truyền xuống cho Dumb Components.

### 3.2. State Management Layer (TanStack Query)
- Sử dụng TanStack Query làm SSOT (Single Source of Truth) cho server data.
- Các hooks được phân tách theo domains: `useMemoryQueries`, `useGraphQueries`, `useSessionQueries`.
- Cấu hình auto-refetch, stale time phù hợp cho từng loại dữ liệu (vd: Metrics cần realtime vs Ontology schema ít thay đổi).

### 3.3. Data Access Layer (API Clients)
- Tầng giao tiếp trực tiếp với Backend Gateway (`:8080`).
- Gói gọn toàn bộ HTTP calls (sử dụng `fetch` hoặc `axios`).
- Xử lý API authentication (API keys, JWT token), auto-inject Tenant headers.
- Gồm các services như: `MemoryApiService`, `GraphApiService`, `TenantApiService`.

### 3.4. Domain Types Layer
- Định nghĩa TypeScript interfaces/types đồng bộ hoàn toàn với cấu trúc dữ liệu trả về từ backend của 6 engines (Graphiti, Cognee, Zep, OpenViking, Memobase, Supermemory) và KGS Platform.

## 4. Directory Structure (`ui/src`)

```text
ui/src/
├── api/                # Data Access Layer (Axios instances, API configs, SSE clients)
│   ├── clients/        # Các API clients theo domain (memory, graph, auth)
│   └── hooks/          # React Query hooks (useMemoryRecall, useDashboardMetrics)
├── assets/             # Hình ảnh, fonts
├── components/         # Reusable UI components
│   ├── common/         # Buttons, Inputs, Dialogs (Radix UI / Shadcn)
│   ├── layouts/        # Sidebar, Header, ContextPanel
│   └── visualizations/ # GraphCanvas, TimelineSlider, NodeCharts
├── config/             # Environment configs, constants
├── contexts/           # React Context (AuthContext, TenantContext, ThemeContext)
├── pages/              # Các trang chính của ứng dụng
│   ├── overview/       # Global Health, Metrics
│   ├── memory-explorer/# Unified Search UI
│   ├── graph-studio/   # Graph interaction & Ontology designer
│   ├── profiles/       # User Profiles (Memobase)
│   └── governance/     # Tenants, Policies, Audit Logs
├── types/              # TypeScript definitions (Models, API Responses)
└── utils/              # Helper functions, formatters (date, bytes, string)
```

## 5. Data Flow (API Integration Strategy)

Toàn bộ UI tuân thủ luồng dữ liệu 1 chiều từ Backend -> API Layer -> Cache -> UI.

1. **REST APIs (CRUD & Querying):**
   - Các hành động tìm kiếm, truy xuất memory (vd: `GET /v1/memory/recall`), hay quản trị cấu hình được fetch qua REST và cache tại TanStack Query.
2. **Realtime Updates (SSE / WebSockets):**
   - Cho Dashboard Metrics, Pipeline statuses, Live Conversations, hệ thống sẽ sử dụng Server-Sent Events (SSE) hoặc WebSockets mở kết nối tới Gateway.
   - Khi có event mới (vd: `memobase.profile.changed`), dữ liệu được đẩy thẳng vào store của React Query (`queryClient.setQueryData`) để UI cập nhật tức thời mà không cần reload.
3. **Authentication & Multi-tenancy:**
   - Khi user chọn một Tenant từ Top Nav, `TenantContext` sẽ cập nhật `tenantId`.
   - Tất cả các HTTP requests từ Data Access Layer sẽ tự động đính kèm header `X-Tenant-ID` hoặc token tương ứng để KGS phân quyền.
4. **Error Handling & Retry:**
   - Các lỗi từ backend sẽ được chuẩn hóa. Lỗi Authentication sẽ đẩy người dùng về trang Login. Lỗi quyền truy cập (ABAC/OPA) sẽ hiển thị thông báo lỗi thân thiện trên UI thông qua notification/toast components.

## 6. Key Components Architecture

### 6.1 Unified Memory Explorer
- **Input Component**: Thanh search đa năng (semantic, graph, lexical).
- **Filter State**: Lưu trữ trong URL Query Params để dễ dàng chia sẻ URL và back/forward.
- **Data Hook**: `useUnifiedSearch(query, filters)` gọi API `/v1/memory/search`.
- **Result List**: Render đa dạng card layout tùy vào Memory Engine (hiển thị Unified Memory Badges theo specs).

### 6.2 Agent Context Debugger
- Kiến trúc chia 3 cột (Left: Request, Center: Pipeline, Right: Final Prompt).
- Sử dụng API Trace đặc biệt từ backend để lấy chi tiết step-by-step assembly.
- Cập nhật pipeline trực quan dạng sequence dựa trên data mảng pipeline steps.

### 6.3 Graph Studio Visualization
- Tích hợp thư viện Graph rendering (React Flow cho Ontology Designer, Cytoscape/Force-graph cho Knowledge Graph explorer mật độ cao).
- Component sẽ fetch dữ liệu Node và Edge rời rạc từ `/v1/graph/context` và update state cục bộ của canvas.

## 7. Security & Performance Considerations

- **Bundle Size Optimization:** Sử dụng Vite Code Splitting và React.lazy() cho các trang nặng như Graph Studio.
- **Data Caching:** TanStack Query cache giúp tránh gọi API lặp lại khi chuyển đổi qua lại giữa các tab (vd: từ Overview sang Explorer rồi quay lại).
- **API Security:** Không lưu trữ API Keys trực tiếp ở client source code. Tất cả xác thực qua hệ thống đăng nhập và sử dụng JWT token (httpOnly cookie hoặc secure memory storage).

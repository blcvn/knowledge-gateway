---
id: FEAT-001
title: Triển khai Dashboard Overview Module
service: ui
version: 1.1.0
status: Ready
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.1)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp góc nhìn tổng quan toàn diện về hệ thống (Platform-wide operational awareness). Đây là màn hình trang chủ `/app/overview`. Yêu cầu AI có thể tự động viết code dựa trên thông số cụ thể bên dưới với Mock Data, không cần đợi backend.

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/overview`
- **Layout**: Sử dụng layout chung có Left Sidebar và Top Navigation.

### 2. TypeScript Interfaces (`src/types/dashboard.ts`)
Yêu cầu AI định nghĩa các kiểu dữ liệu sau:
```typescript
export interface KPIData {
  activeAgents: number;
  recallLatency: string; // e.g., "p50: 45ms / p95: 120ms"
  contextSavings: string; // e.g., "42%"
  graphGrowth: string; // e.g., "+1.2k nodes/day"
  errorRate: string; // e.g., "0.02%"
  activeSessions: number;
}

export interface EngineHealth {
  id: string;
  name: string;
  status: 'Healthy' | 'Warning' | 'Critical';
  latency: string;
  queue: number;
}

export interface MemoryFlowMetrics {
  ingestPerSec: number;
  embeddingsPerSec: number;
  recallPerSec: number;
  queueBacklog: number;
}

export interface HeatmapData {
  time: string;
  density: number;
  frequency: number;
}
```

### 3. Cấu trúc Component (`src/pages/overview/`)
AI cần tạo các component sau:
1. `OverviewPage.tsx`: Component cha, wrap tất cả.
2. `KPICards.tsx`: Render 6 card sử dụng component `Card` của shadcn/ui. Các icon từ `lucide-react` (Users, Activity, Percent, Network, AlertTriangle, MonitorPlay).
3. `MemoryFlowVisualization.tsx`: Dùng flexbox/SVG kết nối các step: `Agent → Gateway → Engine → KGS → Storage`. Ở dưới mỗi step hiển thị metric từ `MemoryFlowMetrics`.
4. `EngineHealthGrid.tsx`: Dùng component `Table` của shadcn/ui. Badge màu xanh cho 'Healthy', vàng cho 'Warning', đỏ cho 'Critical'.
5. `MemoryHeatmap.tsx`: Dùng `Recharts` (ScatterChart hoặc Tooltip bar) để vẽ biểu đồ phân phối memory retrieval theo thời gian.

### 4. Mock Data Khởi Tạo (`src/mock/dashboard.ts`)
Tạo mock data tương ứng với các interface trên để feed vào component.

## Giao Diện & Styling (TailwindCSS)
- Toàn bộ dùng chế độ Dark Mode (`dark` class ở root).
- Nền trang: `bg-slate-950` (Deep dark graphite).
- Các panel: `bg-slate-900 border-slate-800 rounded-xl` (Frosted glass / elevated cards).
- Font: `font-sans` cho UI, `font-mono` cho các giá trị số hoặc text log.

## Acceptance Criteria
> AI tự kiểm tra: Nếu tất cả các tiêu chí này pass → tính năng hoàn chỉnh.

- [ ] AC-1: Route `/app/overview` tải thành công không lỗi console.
- [ ] AC-2: Hiển thị 6 KPI Cards với giá trị và Icon lấy từ thư viện Lucide (Active Agents, Recall Latency, Context Savings, Graph Growth, Error Rate, Active Sessions).
- [ ] AC-3: Table Engine Health Grid hiển thị ít nhất 4 dòng (Graphiti, Cognee, Zep, OpenViking). Trạng thái (Status) dùng `Badge` component với màu thích hợp.
- [ ] AC-4: Memory Flow Diagram hiển thị mũi tên chỉ luồng đi rõ ràng từ Agent đến Storage kèm các chỉ số Ingest/sec, Embeddings/sec.
- [ ] AC-5: Biểu đồ Heatmap (Recharts) render thành công với dữ liệu Mock.


## Yêu cầu Enterprise & Product-Grade

Để đảm bảo chất lượng hệ thống mức Enterprise, Component/Feature này bắt buộc phải xử lý các ràng buộc sau:

### 1. Phân quyền (RBAC) & Bảo mật
- Yêu cầu xác thực (Authentication) hợp lệ để truy cập route này.
- Component phải kiểm tra quyền (Role) trước khi hiển thị các thao tác nhạy cảm (như Delete, Update). Nếu không có quyền, hiển thị trạng thái `disabled` kèm Tooltip giải thích, hoặc ẩn hoàn toàn.

### 2. Trạng thái giao diện (UI States)
- **Loading State**: Sử dụng Skeleton thay vì Spinner mặc định khi fetch dữ liệu lần đầu để giảm thiểu layout shift.
- **Empty State**: Khi không có dữ liệu, hiển thị hình ảnh minh hoạ (Illustration) tinh tế kèm thông điệp rõ ràng và nút "Call to Action" (ví dụ: "Tạo mới").
- **Error State**: Tích hợp Error Boundary tại cấp độ Component; nếu gọi API lỗi (500, 4xx), hiển thị Toast Notification và nút "Thử lại".

### 3. Tối ưu Hiệu suất (Performance)
- Mọi danh sách (List/Table) dài hơn 50 phần tử phải tự động áp dụng Pagination hoặc Virtual Scrolling.
- Các biểu đồ phức tạp (Recharts) hoặc khối dữ liệu lớn phải được bọc trong `React.memo` hoặc sử dụng Server-side Pagination.
- Áp dụng Optimistic UI cho các thao tác thay đổi trạng thái nhỏ (ví dụ: gạt Toggle, xoá item) để trải nghiệm mượt mà.

## Definition of Done
- [ ] Khởi tạo thư mục và file đúng chuẩn.
- [ ] Tất cả Component không có lỗi Type TypeScript (`any`).
- [ ] Tích hợp shadcn/ui chính xác (Card, Badge, Table).

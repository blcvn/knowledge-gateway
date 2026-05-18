---
id: FEAT-006
title: Triển khai Observability & Error Explorer Module (Quản trị lỗi tập trung)
service: ui
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.9)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp giao diện "Quản trị lỗi tập trung" (Error Explorer) để SRE và Developers có thể theo dõi, phân tích các ngoại lệ (exceptions) như crash UI, lỗi 500 từ API, lỗi timeout hay vi phạm policy. Đồng thời theo dõi Metrics và Traces cơ bản.

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/observability`
- **Layout**: Sử dụng `Tabs` để chuyển đổi giữa 3 màn hình: Metrics, Trace Viewer, và Error Explorer.

### 2. TypeScript Interfaces (`src/types/observability.ts`)
```typescript
export interface AppErrorLog {
  id: string;
  timestamp: string;
  level: 'error' | 'warning' | 'fatal';
  message: string;
  stackTrace: string;
  source: 'ui' | 'api' | 'agent';
  tenantId: string;
  userId: string;
  resolved: boolean;
}

export interface MetricData {
  time: string;
  apiLatency: number;
  retrievalLatency: number;
  errorCount: number;
}
```

### 3. Cấu trúc Component (`src/pages/observability/`)
1. `ObservabilityPage.tsx`: Component cha wrap các Tabs (Metrics, Traces, Errors).
2. `MetricsDashboardTab.tsx`: 
   - Dùng `Recharts` vẽ các biểu đồ (Line chart) hiển thị API latency, error counts qua thời gian.
3. `TraceViewerTab.tsx`:
   - Hiển thị luồng tracing phân tán (VD: Gateway -> Graphiti -> KGS -> OpenAI). 
   - Dạng danh sách đổ xuống có độ trễ từng bước.
4. `ErrorExplorerTab.tsx` (Quản trị lỗi tập trung):
   - Bảng hiển thị danh sách `AppErrorLog` bằng shadcn `DataTable`.
   - Sidebar/Sheet hoặc Modal khi click vào một dòng lỗi sẽ bung chi tiết `stackTrace` (dùng font-mono) và thông tin request kèm theo (tenant, user, url).
   - Có nút Action: "Mark as Resolved", "Create Bug Ticket" (chỉ làm nút mock).

### 4. Mock Data Khởi Tạo (`src/mock/observability.ts`)
Tạo danh sách các lỗi giả lập (stack traces, timeout errors) và data biểu đồ mẫu.

## Giao Diện & Styling (TailwindCSS)
- Danh sách lỗi dùng màu highlight nếu level = fatal (VD: `border-l-4 border-red-600 bg-red-950/20`).
- Stacktrace view: Thẻ `<pre>` với `bg-slate-900 text-red-400 p-4 rounded-md overflow-x-auto text-sm`.

## Acceptance Criteria
- [ ] AC-1: Tab `Error Explorer` tải được bảng lỗi có ít nhất 3 cột chính: Timestamp, Source, Message, Level.
- [ ] AC-2: Khi nhấn vào một dòng lỗi (error row), một màn hình chi tiết hiện ra chứa `stackTrace` định dạng code.
- [ ] AC-3: Tab `Metrics` hiển thị ít nhất một biểu đồ Line mô tả số lượng lỗi theo thời gian.
- [ ] AC-4: UI cung cấp bộ lọc lỗi (Filter) theo trạng thái "Resolved" và "Unresolved".


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
- [ ] Khởi tạo thư mục và chia file đúng chuẩn.
- [ ] Chạy mượt mà, không render lỗi trắng trang do dùng `recharts`.
- [ ] Linter và Types không báo lỗi.

---
id: FEAT-005
title: Triển khai Graph Studio Module
service: ui
version: 1.1.0
status: Ready
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.3)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp công cụ Interactive Graph Canvas để theo dõi Knowledge Graph, hiển thị liên kết giữa các thực thể và dòng thời gian phát triển của graph.

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/graph`
- **Layout**: 
  - Vùng trung tâm: Full màn hình là Graph Canvas.
  - Sidebar bên phải: Floating/Fixed Panel là Entity Inspector.
  - Dải bên dưới (Bottom): Timeline Slider Overlay trên Canvas.

### 2. Package Bổ Sung Yêu Cầu
Yêu cầu cài đặt thư viện đồ thị trực quan (Khuyến nghị `reactflow` hoặc thư viện tương đương AI quen thuộc nếu setup nhanh).

### 3. Cấu trúc Component (`src/pages/graph/`)
1. `GraphStudioPage.tsx`: Layout cha điều phối trạng thái (Node đang chọn, Giá trị Timeline).
2. `GraphCanvas.tsx`: Render các Node và Edge Mock.
   - Node: Hiển thị tên thực thể (ví dụ: `User`, `Project`, `Document`).
   - Hỗ trợ Zoom in/out, Pan.
3. `EntityInspector.tsx`: Sidebar hiển thị dữ liệu chi tiết của Node khi được click.
   - Thông tin: Type, Schema, Confidence Score, Tenant.
4. `TimelineSlider.tsx`: Component Slider (shadcn `Slider`) nằm ở viền dưới. Cung cấp chức năng kéo để mock xem thay đổi lịch sử.

### 4. Mock Data (`src/mock/graph.ts`)
Tạo mock JSON cấu trúc Graph:
```typescript
export const initialNodes = [
  { id: '1', position: { x: 250, y: 50 }, data: { label: 'User: binhnt', type: 'Actor', confidence: 0.99 } },
  { id: '2', position: { x: 100, y: 200 }, data: { label: 'Project: VNP', type: 'Entity', confidence: 0.85 } },
  { id: '3', position: { x: 400, y: 200 }, data: { label: 'Document: Spec', type: 'Entity', confidence: 0.9 } },
];
export const initialEdges = [
  { id: 'e1-2', source: '1', target: '2', label: 'OWNS' },
  { id: 'e1-3', source: '1', target: '3', label: 'WROTE' },
];
```

## Giao Diện & Styling (TailwindCSS)
- Graph Background: `bg-slate-950` có grid nền (CSS pattern) để cảm giác là canvas.
- Node styles: Khối có viền phát sáng (Neon edge highlights) `border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.3)] bg-slate-900`.
- Text trên Graph: `font-sans` hoặc `font-mono`.

## Acceptance Criteria
- [ ] AC-1: Truy cập `/app/graph` hiển thị được một không gian 2D với ít nhất 3 Nodes liên kết với nhau.
- [ ] AC-2: Có thể kéo rê Nodes và Zoom in/out toàn canvas.
- [ ] AC-3: Click vào một Node bất kỳ sẽ cập nhật dữ liệu hiển thị bên trong bảng Entity Inspector Sidebar (thông tin thay đổi tương ứng).
- [ ] AC-4: Thanh Timeline Slider có thể tương tác trượt ngang ở dưới đáy màn hình.


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
- [ ] Cài đặt `reactflow` (hoặc library tương đương) đúng cách.
- [ ] Canvas không bị lỗi CSS vỡ layout chồng chéo.
- [ ] Linter và TypeScript Compiler không báo lỗi.

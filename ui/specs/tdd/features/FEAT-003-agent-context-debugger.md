---
id: FEAT-003
title: Triển khai Agent Context Debugger Module
service: ui
version: 1.1.0
status: Ready
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.4)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp giao diện để debug quá trình Agent Context được xây dựng. Đây là tính năng khác biệt lớn nhất (Signature differentiator).

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/debugger`
- **Layout**: CSS Grid 4 khu vực (`grid-cols-3` cho dòng trên, dòng dưới full width).
  - Cột 1 (Left): Agent Request
  - Cột 2 (Center): Context Pipeline
  - Cột 3 (Right): Token Analysis
  - Dòng 2 (Bottom): Final Prompt

### 2. TypeScript Interfaces (`src/types/debugger.ts`)
```typescript
export interface DebugRequest {
  prompt: string;
  metadata: { tenant: string; session: string; model: string };
}

export interface PipelineStep {
  step: number;
  name: string;
  description: string;
  status: 'pending' | 'active' | 'completed';
  timeTakenMs: number;
}

export interface TokenMetrics {
  totalAllocated: number;
  memoryCategories: { category: string; tokens: number }[];
  compressionSavings: number; // percentage
}
```

### 3. Cấu trúc Component (`src/pages/debugger/`)
1. `DebuggerPage.tsx`: Layout chính sử dụng CSS Grid.
2. `AgentRequestPanel.tsx`: Component hiển thị thông tin prompt của user, với badge model.
3. `ContextPipeline.tsx`: Vẽ timeline dọc (Stepper) với các bước:
   1. Query Rewrite
   2. Semantic Recall
   3. Graph Traversal
   4. Salience Ranking
   5. Policy Filter
   6. Compression
   7. Final Context
4. `TokenAnalysis.tsx`: Dùng `Recharts` (PieChart) vẽ phân bổ token theo category (Episodic, Semantic, etc.).
5. `FinalPromptViewer.tsx`: Component dạng Code Editor (hoặc `<pre>`) highlight từ khóa và trích dẫn.

### 4. Mock Data (`src/mock/debugger.ts`)
Tạo dữ liệu JSON giả lập chi tiết cho từng bước Pipeline và Prompt Assembly.

## Giao Diện & Styling (TailwindCSS)
- Timeline dọc (Stepper): Sử dụng đường viền bên trái `border-l-2 border-slate-700` với các chấm tròn `rounded-full`. Bước đang "active" dùng glow effect (`shadow-[0_0_10px_rgba(59,130,246,0.5)]`).
- Font: Prompt view phải sử dụng `font-mono text-sm leading-relaxed text-slate-300 bg-slate-900 p-4 rounded-md overflow-x-auto`.

## Acceptance Criteria
- [ ] AC-1: Grid Layout hiển thị đủ 4 Panel ở vị trí chính xác (Left, Center, Right, Bottom) và scale tốt trên màn hình to.
- [ ] AC-2: Center Panel (Pipeline) hiển thị chuẩn xác luồng 7 bước dạng danh sách/timeline dọc dọc xuống, kèm thời gian giả định (ví dụ: `24ms`).
- [ ] AC-3: Right Panel hiển thị ít nhất 1 biểu đồ tròn (Pie chart) của Recharts mô tả Memory Category Tokens.
- [ ] AC-4: Bottom Panel hiển thị Full Prompt và các injected memories có màu phân biệt (ví dụ text in đậm màu xanh khi đề cập đến một memory injection).


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
- [ ] Toàn bộ UI Responsive tốt.
- [ ] Các Component chia file rõ ràng.
- [ ] Cài đặt thư viện `recharts` và sử dụng không lỗi type.

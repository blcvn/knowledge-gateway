---
id: FEAT-002
title: Triển khai Memory Explorer Module
service: ui
version: 1.1.0
status: Ready
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.2)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp giao diện ElasticSearch/Kibana style để tìm kiếm và kiểm tra mọi loại memory.

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/memory`
- **Layout**: 
  - Bên trái/trên: Khung tìm kiếm và Bộ lọc nâng cao.
  - Vùng chính: Tabs và Danh sách thẻ kết quả (Result Card).
  - Bên phải (Fixed/Drawer): Side Inspector hiển thị chi tiết khi click vào kết quả.

### 2. TypeScript Interfaces (`src/types/memory.ts`)
```typescript
export type MemoryType = 'Episodic' | 'Semantic' | 'Conversational' | 'Procedural';

export interface MemoryItem {
  id: string;
  type: MemoryType;
  title: string;
  summary: string;
  entities: string[];
  sessionsCount: number;
  temporalValidity: string;
  sourceEngine: string;
  policyTags: string[];
  confidenceScore: number;
  metadata: Record<string, any>;
  rawPayload: string;
}
```

### 3. Cấu trúc Component (`src/pages/memory/`)
1. `MemoryExplorerPage.tsx`: Quản lý state tìm kiếm, danh sách kết quả, và trạng thái item đang được chọn (`selectedMemoryId`).
2. `SearchHeader.tsx`: 
   - Input text chính lớn (Search input).
   - Dropdown filters (Tenant, User, Memory Type, Time range) bằng shadcn `Select`.
3. `ResultTabs.tsx`: shadcn `Tabs` với các triggers: All, Episodic, Semantic, Conversational, Procedural.
4. `MemoryResultCard.tsx`:
   - Dùng badge màu sắc chuẩn UX Pattern: Episodic (Purple), Semantic (Blue), Conversational (Green), Procedural (Orange).
   - Hiển thị Title, Summary, Tags.
5. `SideInspector.tsx`:
   - Trượt ra từ phải (dùng `Sheet` component của shadcn hoặc render fixed).
   - Chia section: Provenance, Vector Similarity, Raw Payload (dùng thẻ `<pre>` với font `JetBrains Mono`), Metadata.

### 4. Mock Data Khởi Tạo (`src/mock/memory.ts`)
Tạo danh sách 10 item mẫu với đủ 4 loại memory.

## Giao Diện & Styling (TailwindCSS)
- Tuân thủ Unified Memory Badge System (Classes gợi ý):
  - Episodic: `bg-purple-500/20 text-purple-400 border-purple-500/50`
  - Semantic: `bg-blue-500/20 text-blue-400 border-blue-500/50`
  - Conversational: `bg-green-500/20 text-green-400 border-green-500/50`
  - Procedural: `bg-orange-500/20 text-orange-400 border-orange-500/50`

## Acceptance Criteria
- [ ] AC-1: UI có một thanh tìm kiếm ngang toàn màn hình ở trên cùng kèm nút "Advanced Filters".
- [ ] AC-2: Hiển thị danh sách Mock Data tương ứng khi đổi Tab (Ví dụ: Tab Semantic chỉ hiện kết quả Semantic).
- [ ] AC-3: Mỗi thẻ kết quả có màu sắc Badge đồng nhất với "Unified Memory Badge System".
- [ ] AC-4: Khi nhấn vào thẻ kết quả, mở Side Inspector Panel bên phải hiển thị chi tiết (Raw JSON data).
- [ ] AC-5: Các nút filter dropdown có thể click mở ra dẫu chỉ chứa dữ liệu giả.


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
- [ ] Code không có cảnh báo linter.
- [ ] Code tách biệt component nhỏ hợp lý.
- [ ] Dùng `lucide-react` icon: Clock (Episodic), Network (Semantic), MessageSquare (Conversational), Folder (Procedural).

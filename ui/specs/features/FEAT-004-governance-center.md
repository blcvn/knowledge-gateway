---
id: FEAT-004
title: Triển khai Governance Center Module
service: ui
version: 1.1.0
status: Ready
priority: P0
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md (Section 6.6)
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp khu vực dành riêng cho việc quản trị, phân quyền (Tenants, Policies) và giám sát (Audit logs, GDPR Forget).

## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)

### 1. Route & Layout
- **Path**: `/app/governance`
- **Layout**: Sử dụng thành phần `Tabs` của shadcn/ui để tạo 4 sub-views:
  1. Tenant Management
  2. OPA Policy Editor
  3. GDPR Forget Center
  4. Audit Explorer

### 2. TypeScript Interfaces (`src/types/governance.ts`)
```typescript
export interface Tenant {
  id: string;
  name: string;
  quotaLimit: number;
  apiUsage: number;
  status: 'Active' | 'Suspended';
}

export interface AuditLog {
  id: string;
  timestamp: string;
  actor: string;
  action: string;
  entity: string;
  tenant: string;
  policyResult: 'Allowed' | 'Denied';
}
```

### 3. Cấu trúc Component (`src/pages/governance/`)
1. `GovernancePage.tsx`: Wrap các Tabs.
2. `TenantManagementTab.tsx`: Bảng dữ liệu Shadcn `DataTable`. Cột: Name, Namespace, Quota, API Usage, Status, Action.
3. `OPAPolicyEditorTab.tsx`:
   - Vùng Textarea (hoặc React CodeMirror nếu khả dụng) với font monospaced.
   - Hiển thị mẫu code Rego: `allow { input.user.role == "admin" }`.
   - Nút Save, Format, Test Policy.
4. `GDPRForgetTab.tsx`:
   - Input search user email/ID.
   - Nút đỏ "Erase User Data". Khi click mở modal (Dialog của shadcn) cảnh báo "Cascade Deletion Preview".
5. `AuditExplorerTab.tsx`:
   - Thanh tìm kiếm, filter theo thời gian.
   - Bảng lịch sử (Actor, Action, Policy Result - Xanh/Đỏ).

### 4. Mock Data (`src/mock/governance.ts`)
Tạo mảng dữ liệu giả cho Tenants và Audit Logs (ít nhất 5 dòng mỗi loại).

## Giao Diện & Styling (TailwindCSS)
- GDPR Button: `bg-red-600 hover:bg-red-700 text-white`.
- Audit Log "Denied" row/badge: `bg-red-500/20 text-red-500`. "Allowed": `bg-green-500/20 text-green-500`.

## Acceptance Criteria
- [ ] AC-1: Màn hình hiển thị Navbar có 4 Tab (Tenants, Policies, GDPR, Audit) và chuyển đổi mượt mà.
- [ ] AC-2: Tab Policy Editor phải render được đoạn code `.rego` mẫu bằng thẻ font-mono với syntax highlighting giả lập (màu text cơ bản).
- [ ] AC-3: Tab GDPR Forget Center: Khi nhấn "Erase User Data", bắt buộc phải hiển thị Dialog Xác nhận cảnh báo hành động bất khả nghịch.
- [ ] AC-4: Bảng Audit Logs phân biệt màu sắc rõ ràng ở cột `Policy Result` (Xanh/Đỏ).


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
- [ ] Cấu trúc code module hóa từng Tab riêng.
- [ ] UI không có lỗi linter/type.

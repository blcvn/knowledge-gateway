---
id: FEAT-010
title: Triển khai API & SDK Manager Module
service: ui
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_prd: ux_spec.md
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp giao diện `API & SDK` tương ứng theo cấu trúc tại `ui/src/app/App.tsx`. Dùng để cấp phát API keys, theo dõi rate limits và cấu hình webhook tích hợp.

## Thiết Kế Kỹ Thuật

### 1. Route & Layout
- **Path**: `/app/api`

### 2. Cấu trúc Component (`src/pages/api/`)
1. `ApiManagerPage.tsx`
2. `ApiKeysTable.tsx`: Quản lý cấp phát key (Create, Revoke). Hiển thị dạng ẩn `sk-***`.
3. `RateLimitSettings.tsx`: Form cấu hình Quota (Requests per minute).

## Acceptance Criteria
- [ ] AC-1: Bảng hiển thị danh sách API Key. Nút "Reveal" để xem key thật.
- [ ] AC-2: Nút "Generate New Key" hiển thị modal cho phép tạo key với scopes cụ thể.


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

---
id: FEAT-008
title: Triển khai Pipelines Monitor Module
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
Cung cấp giao diện `Pipelines` tương ứng theo cấu trúc tại `ui/src/app/App.tsx`. Giám sát quá trình ingest data và xử lý embeddings pipeline.

## Thiết Kế Kỹ Thuật

### 1. Route & Layout
- **Path**: `/app/pipelines`

### 2. Cấu trúc Component (`src/pages/pipelines/`)
1. `PipelinesPage.tsx`: Root component.
2. `PipelineGraph.tsx`: Sơ đồ luồng data pipeline (sử dụng React Flow tương tự Graph Studio nhưng với mô hình cố định các pipeline stages: Ingestion -> Chunking -> Embedding -> KGS Sync -> Vector Storage).
3. `JobQueueTable.tsx`: Bảng giám sát các job đang chạy, thất bại hoặc hoàn tất.

## Acceptance Criteria
- [ ] AC-1: Màn hình hiển thị danh sách các tiến trình Pipeline dạng Node-Edge.
- [ ] AC-2: Có bảng quản lý Job bên dưới, tự động refresh (mock setInterval) hiển thị % tiến độ.


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

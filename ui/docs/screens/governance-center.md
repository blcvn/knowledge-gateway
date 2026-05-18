# Giao diện Governance Center

Mô tả thiết kế giao diện chi tiết dự kiến cho phân hệ Quản trị & Tuân thủ (`governance-center`).

## 1. Cấu trúc tổng quan
Màn hình sử dụng **Hệ thống Tabs ngang** (`shadcn/ui Tabs`) đặt ở sát dưới phần Header để chuyển đổi qua lại giữa 4 khu vực quản trị độc lập.

### Khối 1: Tab Tenant Management
Giao diện quản trị khách thuê (Tenants).
- Một bảng dữ liệu (DataTable) lớn bao gồm các cột: Tên Tenant, Namespace, Quota Limit (Giới hạn tài nguyên), API Usage (Sử dụng API), Status (Active/Suspended).
- Các nút thao tác (Actions) ở cuối dòng: Edit, Suspend.

### Khối 2: Tab OPA Policy Editor
Môi trường viết code cấu hình bảo mật trực tiếp.
- Một khung soạn thảo mã nguồn (Code Editor / Textarea monospace) tối màu.
- Hiển thị syntax highlighting giả lập cho ngôn ngữ `Rego`.
- Bên phải (hoặc bên dưới) là các nút: **Format Code**, **Test Policy** và **Save**.

### Khối 3: Tab GDPR Forget Center
Khu vực dành riêng cho thao tác xử lý quyền lãng quên (Right to be Forgotten).
- Thanh tìm kiếm: "Nhập User ID hoặc Email để quét dữ liệu".
- Nút bấm chính: **"Erase User Data"** (Nút màu đỏ cảnh báo nguy hiểm `bg-red-600`).
- Khi bấm nút, một Dialog/Modal cảnh báo hiển thị "Cascade Deletion Preview" (Danh sách các vùng dữ liệu sẽ bị xóa) yêu cầu gõ lại tên User để xác nhận.

### Khối 4: Tab Audit Explorer
Khu vực truy vết lịch sử thao tác của toàn hệ thống.
- Bảng hiển thị: Thời gian, Actor (Người/Bot thực hiện), Action (Hành động), Entity (Thực thể bị tác động), Policy Result.
- **Trực quan hóa**: Cột Policy Result sử dụng Badge màu đỏ cho "Denied" và màu xanh lá cho "Allowed".

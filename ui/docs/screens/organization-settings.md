# Giao diện Organization Settings

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Cài đặt tổ chức (`organization-settings`).

## 1. Cấu trúc tổng quan
Sử dụng Layout đặc trưng của trang Cài đặt (Settings Layout): Một Menu dọc nhỏ phía bên trái (Sidebar con) và Nội dung cài đặt nằm bên phải.

### Khối 1: Settings Navigation (Left Panel)
Danh sách các mục con:
- Profile (Hồ sơ cá nhân)
- Team (Đội ngũ)
- Billing (Thanh toán)
- Preferences (Tùy chỉnh hệ thống)

### Khối 2: Team Management (Right Content)
Giao diện quản lý danh sách thành viên.
- Bảng danh sách User: Email, Role (Admin/Viewer), Trạng thái (Active/Pending).
- Vùng Nhập Email (Input bar) ở đầu bảng để gửi lời mời (Invite Team Member), kèm nút "Send Invite" màu xanh.

### Khối 3: System Preferences (Right Content)
Giao diện form thay đổi cấu hình hiển thị:
- **UI Theme**: 3 ô vuông lớn để chọn giao diện (Light / Dark / System). Mặc định tích chọn thẻ "Dark".
- **Date Format**: Dropdown chọn định dạng ngày (MM/DD/YYYY hoặc YYYY-MM-DD).
- Nút "Save Changes" ở dưới cùng để lưu trạng thái cục bộ vào Zustand.

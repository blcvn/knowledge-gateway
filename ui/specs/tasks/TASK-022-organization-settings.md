---
id: TASK-022
title: Implement Giao diện Organization Settings
service: ui
type: task
status: done
source: docs/screens/organization-settings.md
---

# TASK-022: Triển khai Giao diện Organization Settings

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/screens/organization-settings.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Cài đặt tổ chức (`organization-settings`)....

**## 1. Cấu trúc tổng quan**
### Khối 1: Settings Navigation (Left Panel)
- Profile (Hồ sơ cá nhân)
- Team (Đội ngũ)
- Billing (Thanh toán)
- Preferences (Tùy chỉnh hệ thống)
### Khối 2: Team Management (Right Content)
- Bảng danh sách User: Email, Role (Admin/Viewer), Trạng thái (Active/Pending).
- Vùng Nhập Email (Input bar) ở đầu bảng để gửi lời mời (Invite Team Member), kèm nút "Send Invite" màu xanh.
### Khối 3: System Preferences (Right Content)
- **UI Theme**: 3 ô vuông lớn để chọn giao diện (Light / Dark / System). Mặc định tích chọn thẻ "Dark".
- **Date Format**: Dropdown chọn định dạng ngày (MM/DD/YYYY hoặc YYYY-MM-DD).
- Nút "Save Changes" ở dưới cùng để lưu trạng thái cục bộ vào Zustand.

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [x] Code tuân thủ theo đúng chuẩn của dự án.
- [x] Giao diện (nếu có) hiển thị đúng theo mô tả trong document.
- [x] Mọi chức năng/luồng tương tác trong tài liệu đều hoạt động chính xác.
- [x] Build thành công và không phá vỡ các luồng (flows) hiện tại.


### 💎 Enterprise & Product-Grade UI/UX Standards
- [x] **Premium Aesthetics**: Giao diện mang cảm giác cao cấp (premium). Tránh dùng màu sắc cơ bản. Ưu tiên dùng hệ màu HSL mượt mà, dark mode sâu sắc (deep dark), hiệu ứng gradient tinh tế và glassmorphism.
- [x] **Typography**: Sử dụng modern typography (Inter, Roboto, Outfit). Layout tuân thủ chặt chẽ spacing grid system, UI không bị chật chội hoặc lỏng lẻo.
- [x] **Dynamic & Responsive**: Tích hợp các micro-animations, hiệu ứng hover, focus states, và transition mượt mà giúp giao diện "sống động" và phản hồi cao.
- [x] **Enterprise Completeness**: Xử lý triệt để loading states, empty states, error boundaries, và accessible (a11y) đầy đủ.

## 4. Tài liệu tham khảo
- [Source Document](../../../docs/screens/organization-settings.md)

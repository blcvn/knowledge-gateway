---
id: TASK-041
title: Triển khai Authentication & Phân quyền RBAC (Enterprise Grade)
service: ui
type: task
status: done
source: enterprise-requirements
---

# TASK-041: Triển khai Authentication & Phân quyền RBAC (Enterprise Grade)

## 1. Mục tiêu (Objective)
Đảm bảo an toàn thông tin và tính toàn vẹn của nền tảng thông qua cơ chế Xác thực (Authentication) và Phân quyền (Role-Based Access Control - RBAC) chặt chẽ ở cấp độ Frontend.

## 2. Phạm vi công việc (Scope)
- **Authentication Flow**: Xây dựng luồng đăng nhập (Login), Đăng xuất (Logout), và tự động Refresh Token an toàn (Silent refresh).
- **Secure Storage**: Cấu hình lưu trữ an toàn cho Auth tokens (ưu tiên HttpOnly Cookies, nếu dùng localStorage/sessionStorage cần mã hóa phù hợp).
- **Route Guards**: Thiết lập Higher-Order Components (HOC) hoặc Middleware để bảo vệ các Private Routes, chuyển hướng người dùng chưa đăng nhập về trang Login.
- **RBAC (Role-Based Access Control)**: Cấu hình cơ chế ẩn/hiện Component và Route dựa trên quyền hạn (Roles/Permissions) của người dùng (ví dụ: Dev, Admin, Viewer).
- **Session Timeout**: Triển khai cơ chế theo dõi thao tác người dùng (Idle timeout) để tự động đăng xuất nhằm bảo mật dữ liệu.

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [x] Người dùng không có token hợp lệ không thể truy cập bất kỳ trang nào ngoại trừ Login/Public.
- [x] Hệ thống RBAC hoạt động đúng: Các menu/nút bấm không thuộc quyền sẽ bị ẩn hoặc disabled.
- [x] Luồng Refresh Token hoạt động ngầm (transparent) không làm đứt gãy thao tác của người dùng.
- [x] Đăng xuất sẽ xóa sạch toàn bộ state, cache (React Query), và thông tin session hiện tại ở trình duyệt.

## 4. Tài liệu tham khảo
- Enterprise Security & RBAC Guidelines.

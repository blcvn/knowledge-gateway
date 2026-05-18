# Giao diện API & SDK Manager

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Quản lý API Key (`api-sdk-manager`).

## 1. Cấu trúc tổng quan
Màn hình cung cấp giao diện chuẩn của một hệ thống Dev Portal.

### Khối 1: API Keys Table
- Giao diện dạng Bảng liệt kê các API Key đã cấp phát.
- **Hiển thị Key (Security mode)**: Các key mặc định bị che dấu dạng `sk-*****************`.
- **Nút Reveal**: Có một icon hình con mắt. Khi bấm vào sẽ hiển thị Key thật, nếu bấm Copy sẽ hiện hiệu ứng Tick xanh (Đã copy).
- **Nút Generate New Key**: Nằm góc trên phải. Bấm vào sẽ mở ra Modal cho phép điền tên Key và chọn quyền (Scopes).

### Khối 2: Rate Limit Settings
- Giao diện dạng Form cấu hình.
- Các Input (số lượng) cho phép nhập cấu hình Quota: "Requests per minute", "Tokens per day".
- Thanh Progress Bar hiển thị lượng API đã tiêu thụ trong tháng hiện tại (Ví dụ: 80,000 / 100,000 requests) màu xanh dương, hoặc màu đỏ nếu sắp hết.

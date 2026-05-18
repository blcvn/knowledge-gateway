# Giao diện Placeholder Screens

Mô tả chi tiết kiến trúc hiển thị của component màn hình tạm (dựa trên source code `Placeholder.tsx`) được dùng chung cho các phân hệ chưa hoàn thiện.

## 1. Cấu trúc tổng quan
Thành phần `Placeholder` được thiết kế cực kỳ tối giản, đóng vai trò lấp đầy không gian chính (Main Workspace) khi người dùng bấm vào các mục điều hướng đang trong quá trình phát triển (Draft/Pending features). 

Màn hình áp dụng Flexbox để căn giữa hoàn toàn nội dung theo cả hai chiều ngang và dọc (`flex items-center justify-center`).

## 2. Các thành phần bên trong
Mỗi Placeholder Screen bao gồm một khối (block) được căn giữa màn hình (`text-center`) giới hạn độ rộng tối đa (`max-w-md`), chứa 3 thành phần xếp dọc:

1. **Icon Đại diện (Hero Icon)**:
   - Icon kích thước cực lớn (`w-12 h-12`), màu mờ (`text-white/40`).
   - Nằm bên trong một khối hình vuông bo tròn sâu (`rounded-2xl`) với nền xám mờ (`bg-white/5`).
2. **Tiêu đề (Title)**:
   - Text chữ in hoa đậm vừa (`font-semibold text-xl`), hiển thị tên của module tương ứng.
3. **Mô tả ngắn (Description)**:
   - Đoạn văn bản nhỏ (`text-sm`), có độ trong suốt 50% (`text-white/50`), tóm tắt công năng của module đó trong tương lai.

## 3. Các phân hệ hiện đang áp dụng Placeholder
Trong `App.tsx`, có 7 mục đang tái sử dụng giao diện này (tương ứng với các Spec FEAT-004 đến FEAT-011):
- **Sessions**: Xem và replay AI agent conversations.
- **Governance Center**: Quản lý policies và tenants.
- **Pipelines**: Theo dõi luồng ingest.
- **Infrastructure**: Trạng thái service health.
- **Observability**: Tracking Metrics và traces.
- **API & SDK**: Quản lý API Key.
- **Settings**: Cấu hình tổ chức.

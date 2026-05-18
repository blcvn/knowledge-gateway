# User Navigation Flow: P5 - IDE Plugin User (Secondary)

## Vai trò & Mục tiêu
Developer sử dụng các AI coding assistants (như Cursor, GitHub Copilot, hoặc VNP Plugin) trên Editor cục bộ. Họ muốn AI assistant "nhớ" được coding style, cấu trúc project và các context cũ giữa các phiên code.

## Flow 1: Thiết lập tích hợp VNP Memory vào IDE
**Ngữ cảnh**: Người dùng muốn cấu hình cho Plugin AI cục bộ kết nối lên VNP Memory để đồng bộ trí nhớ.
1. **Truy cập `API & SDK Manager`**:
   - (Thông qua trình duyệt web Console) Chọn mục **Generate New Key**.
   - Tạo một API Key với quyền giới hạn (vd: chỉ được phép `read/write` vào không gian bộ nhớ của riêng User đó).
   - Copy mã API Key (`sk-****`).
2. **Tại màn hình Settings của IDE Plugin**:
   - Dán API Key vào mục "VNP Memory Backend".
   - (Optional) Bật cờ "Auto-sync Workspaces".

## Flow 2: Quản lý bộ nhớ Code Context (Self-service)
**Ngữ cảnh**: AI Assistant vô tình nhớ sai một Design Pattern của dự án. Developer muốn sửa lại ngay.
1. **Truy cập `Memory Explorer`**:
   - Tìm kiếm đoạn text liên quan đến "Design Pattern" hoặc tên hàm bị lỗi.
   - Khi hệ thống hiển thị thẻ Memory chứa context bị sai, User có thể nhấn **View Details ->** và **Edit/Delete** để cập nhật lại context đúng (nếu Role cho phép).

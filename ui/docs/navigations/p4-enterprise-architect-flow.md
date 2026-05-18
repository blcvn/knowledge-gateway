# User Navigation Flow: P4 - Enterprise Architect (Governance & Compliance)

## Vai trò & Mục tiêu
Đảm bảo hệ thống AI tuân thủ các quy định nghiêm ngặt về quyền riêng tư (Privacy), bảo mật (Security), và theo dõi dấu vết kiểm toán (Audit Trail) cho tổ chức Enterprise.

## Flow 1: Thực thi quyền lãng quên GDPR (Right to be Forgotten)
**Ngữ cảnh**: Một người dùng (End-User) yêu cầu xóa toàn bộ dữ liệu trí nhớ mà AI đã thu thập về họ theo chuẩn luật GDPR Châu Âu.
1. **Truy cập `Governance Center`**:
   - Chuyển sang Tab **GDPR Forget Center**.
   - Nhập User ID hoặc Email của khách hàng vào thanh tìm kiếm.
2. **Kiểm tra và Xóa (Cascade Deletion)**:
   - Hệ thống quét và hiển thị "Cascade Deletion Preview" (vd: Phát hiện 24 Memories, 13 Entities, 5 Graph Edges liên quan).
   - Nhấp vào nút cảnh báo đỏ **"Erase User Data"**.
   - Gõ lại User ID vào Modal xác nhận để đảm bảo thao tác không bị bấm nhầm.
   - Nhấn Confirm. Dữ liệu sẽ bị xóa Hard Delete hoặc đánh dấu Tombstone (tùy cấu hình).

## Flow 2: Cập nhật Chính sách ABAC (Attribute-Based Access Control)
**Ngữ cảnh**: Cần cấm Agent không được truy cập vào các loại Memory được dán nhãn `Financial`.
1. **Truy cập `Governance Center`**:
   - Chuyển sang Tab **OPA Policy Editor**.
   - Viết hoặc sửa đoạn mã Rego: `deny { input.memory.tags[_] == "Financial" }`.
   - Nhấn **Test Policy** bằng Mock data xem có chặn đúng không.
   - Nhấn **Save** để deploy policy áp dụng toàn hệ thống.

## Flow 3: Truy vết Kiểm toán (Audit)
**Ngữ cảnh**: Nghi ngờ có một API Key đang bị lộ và đọc trộm Memory trái phép.
2. **Truy cập `Governance Center`**:
   - Chuyển sang Tab **Audit Explorer**.
   - Lọc theo Action `Read_Memory` và trạng thái Policy Result là `Denied`.
   - Tìm ra Actor (API Key hoặc Tenant) đang spam request và xử lý khóa Key bên màn hình `API & SDK Manager`.

## Flow 4: Cấu hình Data Retention & TTL
**Ngữ cảnh**: Tổ chức có quy định không được lưu trữ dữ liệu hội thoại quá 90 ngày.
1. **Truy cập `Governance Center`**:
   - Chọn tab cấu hình Retention Policies.
   - Thiết lập quy tắc TTL (Time-to-Live) = 90 days đối với tất cả các Memory có phân loại `Personal_Data`.
   - Lưu chính sách. Hệ thống sẽ có một cron job tự động chạy quét và dọn dẹp mỗi đêm.

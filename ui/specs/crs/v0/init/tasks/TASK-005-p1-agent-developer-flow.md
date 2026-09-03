---
id: TASK-005
title: Implement User Navigation Flow: P1 - AI Agent Developer
service: ui
type: task
status: done
source: docs/navigations/p1-agent-developer-flow.md
---

# TASK-005: Triển khai User Navigation Flow: P1 - AI Agent Developer

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/navigations/p1-agent-developer-flow.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**## Vai trò & Mục tiêu**
AI Agent Developer (P1) sử dụng VNP Memory Console chủ yếu để kiểm tra xem Agent của mình có "nhớ" đúng thông tin hay không và debug lý do tại sao một thông tin lại (hoặc không) được tiêm vào Context (Prompt)....

**## Flow 1: Debugging Agent Context (Kiểm tra chất lượng RAG)**
**Ngữ cảnh**: Developer phát hiện Agent trả lời sai hoặc quên thông tin trong một phiên chat cụ thể.
1. **Truy cập `Sessions Explorer`**:
   - Sử dụng thanh tìm kiếm để nhập ID của người dùng hoặc Session ID bị lỗi.
   - Nhấp vào phiên hội thoại để mở **Right Panel (Session Replay Viewer)**.
   - Định vị tin nhắn mà Agent trả lời sai.
2. **Kiểm tra thông tin nguồn (Memory Explorer)**:
   - Từ tin nhắn lỗi, chuyển sang tab **Memory Explorer**.
   - Tìm kiếm từ khóa hoặc filter theo Session ID đó....

**## Flow 2: Đánh giá chất lượng Ontology tự sinh**
**Ngữ cảnh**: Developer muốn xem hệ thống Auto-ontology có trích xuất đúng các thực thể (Entities) từ hội thoại không.
1. **Truy cập `Graph Studio`**:
   - Sử dụng chuột để Zoom in vào các Node mới sinh trong tuần.
   - Nhấp vào một Node bất kỳ để mở **Entity Inspector Sidebar**.
   - Kiểm tra các `Related Facts` xem LLM có sinh ra các fact bị trùng lặp hoặc mâu thuẫn không.
2. Nhanh chóng chuyển sang **Memory Explorer** thông qua nút "View Source Memory" trên thanh Inspector để đọc lại payload ...

**## Flow 3: Truy vấn Temporal Timeline**
**Ngữ cảnh**: Developer muốn xem một sự kiện (Fact) đã thay đổi như thế nào theo thời gian (ví dụ: User từng thích màu Đỏ, sau đó đổi sang màu Xanh).
1. **Truy cập `Graph Studio`**:
   - Sử dụng công cụ Timeline Slider ở mép dưới màn hình.
   - Kéo thanh trượt qua các mốc thời gian để xem các Node "Đỏ" và "Xanh" được sinh ra và kết nối với User như thế nào.
   - Nhấp vào cạnh (Edge) nối giữa các Node để xem Metadata về thời gian Valid From / Valid To....

**## Flow 4: Cấu hình tự động quên (Auto-forget)**
**Ngữ cảnh**: Developer muốn hệ thống tự động dọn dẹp các ngữ cảnh ngắn hạn sau một thời gian để tiết kiệm dung lượng.
1. **Truy cập `Settings` hoặc `Governance Center`**:
   - Tìm đến mục cấu hình Policy / TTL.
   - Định nghĩa luật: Các Memory có type là `Conversational` sẽ bị expired sau 30 ngày....

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
- [Source Document](../../../docs/navigations/p1-agent-developer-flow.md)

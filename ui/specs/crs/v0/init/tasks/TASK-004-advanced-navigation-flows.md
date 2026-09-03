---
id: TASK-004
title: Implement Các Luồng Điều Hướng Nâng Cao (20 Advanced Flows) - Bản Chi Tiết
service: ui
type: task
status: done
source: docs/navigations/advanced-navigation-flows.md
---

# TASK-004: Triển khai Các Luồng Điều Hướng Nâng Cao (20 Advanced Flows) - Bản Chi Tiết

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/navigations/advanced-navigation-flows.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Tài liệu này cung cấp mô tả UI/UX chi tiết từng bước (Step-by-step interaction) cho 20 luồng điều hướng nâng cao thuộc VNP Memory Console. Bản chi tiết này phục vụ trực tiếp cho quá trình thiết kế Wireframe/Figma và viết kịch bản Test (E2E Testing)....

**## 1. P1 - AI Agent Developer (+4 Flows)**
### Flow 5: Kiểm thử cơ chế dự phòng (Fallback RAG Testing)
- **Tiền điều kiện**: Agent đã được nạp dữ liệu nhưng tắt chức năng Graph Search tự động.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: UI hiển thị luồng chẻ nhánh: nhánh `Semantic Search` báo nhãn `Low Confidence (< 0.6)`, mũi tên flow chuyển hướng sang nhánh `Graph Traversal` báo nhãn `Success` với độ sâu 2 hops.
### Flow 6: Giả lập chia sẻ trí nhớ đa tác tử (Multi-Agent Memory Sharing)
- **Tiền điều kiện**: Hệ thống có Bot A (DevOps) và Bot B (Developer) cùng chung một Shared Namespace.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Trong panel chi tiết, trường "Read Access" hiển thị danh sách các bot được quyền đọc (bao gồm Bot_B). Nút Edit bị làm mờ (Disable) vì Bot B chỉ có quyền Read-only đối với trí nhớ của Bot A.
### Flow 7: Import lịch sử hội thoại hàng loạt (Bulk Import)
- **Tiền điều kiện**: File `history_chat.jsonl` chuẩn bị sẵn trên máy.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Một hàng (Row) mới xuất hiện với trạng thái `Processing`. Thanh Progress bar màu xanh dương chạy dần từ 0% đến 100%. Thông số Throughput (Chunks/sec) nhảy liên tục.
### Flow 8: Xuất bản Graph Snapshot để Debug cục bộ
- **Tiền điều kiện**: Đang mở màn hình Graph Studio với các Node đang hiển thị.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Trình duyệt tải xuống một file `snapshot_123.graphml`. Hệ thống hiện Toast notification xanh lá: "Exported 10 nodes and 14 edges successfully".

**## 2. P2 - Platform / DevOps Engineer (+4 Flows)**
### Flow 9: Khôi phục cấu hình Engine bị lỗi (Rollback Engine Config)
- **Tiền điều kiện**: Dịch vụ Qdrant bị suy giảm hiệu năng do cập nhật sai ConfigMap.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Thẻ Qdrant chuyển sang trạng thái `Restarting` màu xanh dương khoảng 30s, sau đó đèn báo chuyển sang màu Xanh lá (Healthy). Cột Uptime được reset.
### Flow 10: Quản lý Chứng chỉ Bảo mật (SSL/TLS & Webhook Auth)
- **Tiền điều kiện**: Chứng chỉ TLS nội bộ sắp hết hạn.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Hệ thống chạy thanh tiến trình "Validating Chain...". Xong hiện Toast báo "Certificate Updated. Valid until 2027".
### Flow 11: Chuyển dữ liệu cũ sang Cold Storage (Data Archiving)
- **Tiền điều kiện**: Dung lượng ổ SSD NVMe vượt ngưỡng 85%.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Giao diện tự động điều hướng sang `Pipelines`, hiển thị một Job mới tên là `Cold_Storage_Migration_Job` đang chạy với thanh tiến trình.
### Flow 12: Cô lập Tenant bị xâm phạm (Kill Switch)
- **Tiền điều kiện**: Có cảnh báo hệ thống báo Tenant "Alpha" đang có dấu hiệu truy cập trái phép lượng lớn.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Status của Tenant Alpha lập tức đổi sang cục badge đỏ `Suspended`. Mọi truy vấn API từ Tenant này ngay lập tức trả về lỗi 403 Forbidden.

**## 3. P3 - ML/AI Engineer (+3 Flows)**
### Flow 13: Đấu nối mô hình Embedding tùy chỉnh
- **Tiền điều kiện**: Kỹ sư vừa host một mô hình `bge-m3` nội bộ trên port 8000.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Mô hình xuất hiện trong danh sách Model. Khi sang màn `Context Debugger`, tại ô Dropdown Model, đã có thể chọn `Medical_BGE_M3`.
### Flow 14: Kiểm thử độ chính xác của trích xuất thực thể (NER Precision)
- **Tiền điều kiện**: Cần test thử Ontology "Medical_Diseases".
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Các cụm từ "đái tháo đường type 2", "Bệnh nhân nam" được highlight màu vàng trong văn bản. Phía trên Canvas mờ ảo hiện ra các Node ảo (Preview Nodes) kết nối với nhau bằng viền đứt đoạn. 
### Flow 15: Phân tích tối ưu chi phí (Token vs Context Window)
- **Tiền điều kiện**: Cần báo cáo ngân sách Token LLM hàng tháng.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: UX hiển thị Tooltip sắc nét với số liệu cụ thể. Kỹ sư dựa vào đó để điều chỉnh lại Max Context Top_K trong Config nhằm giảm tải.

**## 4. P4 - Enterprise Architect (+3 Flows)**
### Flow 16: Trích xuất Báo cáo Tuân thủ (SOC2 / HIPAA Audit Report)
- **Tiền điều kiện**: Cần cung cấp bằng chứng hệ thống lưu vết (Audit Trail) không bị chỉnh sửa.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Thanh Toast thông báo "Generating report...", sau 10s tải xuống file `HIPAA_Audit_Q1.pdf`. File PDF chứa danh sách Logs được đóng dấu hash mã hóa để chống giả mạo.
### Flow 17: Cấu hình Phân quyền RBAC nội bộ
- **Tiền điều kiện**: Có một nhân sự mới cần cấp quyền vào Dashboard để xử lý lỗi nhưng cấm xem Data thật.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Trở lại tab `Members`, bấm Invite User với email hỗ trợ viên và gán Role `L1_Support`. Giao diện của hỗ trợ viên khi đăng nhập sẽ tự động làm mờ các nội dung Memory nhạy cảm.
### Flow 18: Kích hoạt luật che mờ dữ liệu nhạy cảm (Data Masking/PII Redaction)
- **Tiền điều kiện**: Hệ thống đang bị lưu lọt số điện thoại của người dùng vào Vector DB.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Khi sang `Context Debugger` và nhập thử text có chứa số điện thoại, Output hiển thị trong Memory Explorer của chuỗi hội thoại đó sẽ hoàn toàn hiện dấu hoa thị (***).

**## 5. P7 - AI Power User (+2 Flows)**
### Flow 19: Gộp thực thể thủ công (Manual Entity Merging)
- **Tiền điều kiện**: Graph sinh ra bị lặp Node do sai chính tả từ phía người dùng khi chat.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Animation (hiệu ứng) hút Node "JS" vào trong "Javascript" xảy ra mượt mà. Các đường dây (Edges) cũ của JS được chuyển hết sang Javascript.
### Flow 20: Xuất kho lưu trữ cá nhân (Export Personal Vault)
- **Tiền điều kiện**: Người dùng muốn chuyển Data sang nền tảng AI khác.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Một file `My_AI_Memories.zip` được tải về chứa toàn bộ file Markdown dễ đọc.

**## 6. P5 & P6 - Ecosystem Users (+2 Flows)**
### Flow 21: Thu hồi API Key của IDE Plugin đã bị lộ (Revoke Key)
- **Tiền điều kiện**: Developer P5 dùng máy tính công ty cài Plugin, nay đã nghỉ việc.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Dòng Key biến mất (hoặc gạch ngang). Thiết bị cài IDE Plugin lập tức văng ra trạng thái Unauthenticated khi gửi Prompt tiếp theo.
### Flow 22: Theo dõi Webhook Delivery Failures (P6)
- **Tiền điều kiện**: Đối tác báo họ không nhận được event `node.created`.
- **Các bước thực hiện**:
- **Kết quả mong đợi**: Kỹ sư chụp ảnh màn hình Response Headers lỗi 502 này gửi cho phía đối tác để họ sửa cấu hình Firewall chặn IP của VNP Memory. Dưới đáy có nút `Resend Payload` để thử lại ngay sau khi fix xong.

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
- [Source Document](../../../docs/navigations/advanced-navigation-flows.md)

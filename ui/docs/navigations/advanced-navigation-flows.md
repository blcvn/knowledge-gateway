# Các Luồng Điều Hướng Nâng Cao (20 Advanced Flows) - Bản Chi Tiết

Tài liệu này cung cấp mô tả UI/UX chi tiết từng bước (Step-by-step interaction) cho 20 luồng điều hướng nâng cao thuộc VNP Memory Console. Bản chi tiết này phục vụ trực tiếp cho quá trình thiết kế Wireframe/Figma và viết kịch bản Test (E2E Testing).

---

## 1. P1 - AI Agent Developer (+4 Flows)

### Flow 5: Kiểm thử cơ chế dự phòng (Fallback RAG Testing)
- **Tiền điều kiện**: Agent đã được nạp dữ liệu nhưng tắt chức năng Graph Search tự động.
- **Các bước thực hiện**:
  1. Trên Sidebar, click vào `Agent Context Debugger`.
  2. Tại bảng "Query Settings" bên phải, toggle OFF nút `Force Semantic Only`.
  3. Tại ô Input "Test Prompt", gõ câu hỏi đòi hỏi Multi-hop reasoning: *"Công ty mà bạn của CEO đang đầu tư tên là gì?"*.
  4. Bấm `Run Debug`.
  5. Cuộn xuống khu vực **Context Pipeline**. Click mở rộng bước `Retrieval Engine`.
- **Kết quả mong đợi**: UI hiển thị luồng chẻ nhánh: nhánh `Semantic Search` báo nhãn `Low Confidence (< 0.6)`, mũi tên flow chuyển hướng sang nhánh `Graph Traversal` báo nhãn `Success` với độ sâu 2 hops.

### Flow 6: Giả lập chia sẻ trí nhớ đa tác tử (Multi-Agent Memory Sharing)
- **Tiền điều kiện**: Hệ thống có Bot A (DevOps) và Bot B (Developer) cùng chung một Shared Namespace.
- **Các bước thực hiện**:
  1. Truy cập màn hình `Memory Explorer`.
  2. Trên thanh **Filter Toolbar**, mở dropdown `Agent Scope`. Chọn `Bot_B (Developer)`.
  3. Mở dropdown `Namespace`. Chọn `Project_Phoenix`.
  4. Chọn thẻ Memory có title "Kubernetes Cluster IP: 192.168.1.5" do Bot A tạo.
  5. Click nút `View Details`.
- **Kết quả mong đợi**: Trong panel chi tiết, trường "Read Access" hiển thị danh sách các bot được quyền đọc (bao gồm Bot_B). Nút Edit bị làm mờ (Disable) vì Bot B chỉ có quyền Read-only đối với trí nhớ của Bot A.

### Flow 7: Import lịch sử hội thoại hàng loạt (Bulk Import)
- **Tiền điều kiện**: File `history_chat.jsonl` chuẩn bị sẵn trên máy.
- **Các bước thực hiện**:
  1. Nhấp vào mục `Pipelines` trên Sidebar.
  2. Ở góc trên bên phải, bấm nút `New Ingestion Job`.
  3. Một Modal mở ra, chọn tab `Upload File`. Kéo thả file `history_chat.jsonl` vào khu vực Dropzone.
  4. Bấm `Start Pipeline`.
  5. Quay lại **Job Queue Table** ở nửa dưới màn hình `Pipelines`.
- **Kết quả mong đợi**: Một hàng (Row) mới xuất hiện với trạng thái `Processing`. Thanh Progress bar màu xanh dương chạy dần từ 0% đến 100%. Thông số Throughput (Chunks/sec) nhảy liên tục.

### Flow 8: Xuất bản Graph Snapshot để Debug cục bộ
- **Tiền điều kiện**: Đang mở màn hình Graph Studio với các Node đang hiển thị.
- **Các bước thực hiện**:
  1. Truy cập `Graph Studio`.
  2. Sử dụng thanh công cụ Lasso (chọn vùng) trên Top Header để khoanh tròn một cụm 10 Nodes trên Canvas.
  3. Click chuột phải vào vùng đã chọn, một Context Menu hiện ra.
  4. Chọn `Export Data...`.
  5. Trong Dialog, chọn Format là `GraphML` và tích vào `Include Entity Metadata`.
  6. Bấm `Download`.
- **Kết quả mong đợi**: Trình duyệt tải xuống một file `snapshot_123.graphml`. Hệ thống hiện Toast notification xanh lá: "Exported 10 nodes and 14 edges successfully".

---

## 2. P2 - Platform / DevOps Engineer (+4 Flows)

### Flow 9: Khôi phục cấu hình Engine bị lỗi (Rollback Engine Config)
- **Tiền điều kiện**: Dịch vụ Qdrant bị suy giảm hiệu năng do cập nhật sai ConfigMap.
- **Các bước thực hiện**:
  1. Bấm vào `Infrastructure` trên Sidebar.
  2. Tại **Service Status Grid**, tìm thẻ `Qdrant Vector DB`. Đèn báo trạng thái đang nháy màu Vàng (Degraded).
  3. Click vào thẻ để mở **Service Detail Panel**.
  4. Chuyển sang tab `Config History`.
  5. Trong danh sách version, chọn version của "Yesterday", bấm icon `Menu (3 chấm)` -> Chọn `Rollback to this version`.
  6. Bảng cảnh báo xuất hiện, gõ `CONFIRM` và bấm `Apply`.
- **Kết quả mong đợi**: Thẻ Qdrant chuyển sang trạng thái `Restarting` màu xanh dương khoảng 30s, sau đó đèn báo chuyển sang màu Xanh lá (Healthy). Cột Uptime được reset.

### Flow 10: Quản lý Chứng chỉ Bảo mật (SSL/TLS & Webhook Auth)
- **Tiền điều kiện**: Chứng chỉ TLS nội bộ sắp hết hạn.
- **Các bước thực hiện**:
  1. Chuyển đến `API & SDK`.
  2. Tại thanh menu phụ nằm dọc bên trái (hoặc Tab), chọn `Security Certificates`.
  3. Tại bảng `Load Balancer TLS`, bấm nút `Replace Certificate`.
  4. Modal hiện ra: Upload file `cert.pem` và `private.key`.
  5. Bấm `Apply & Reload Traefik/Gateway`.
- **Kết quả mong đợi**: Hệ thống chạy thanh tiến trình "Validating Chain...". Xong hiện Toast báo "Certificate Updated. Valid until 2027".

### Flow 11: Chuyển dữ liệu cũ sang Cold Storage (Data Archiving)
- **Tiền điều kiện**: Dung lượng ổ SSD NVMe vượt ngưỡng 85%.
- **Các bước thực hiện**:
  1. Bấm vào `Settings` trên Sidebar.
  2. Chọn mục `Storage Management`.
  3. Kéo xuống phần `Data Lifecycle`. Tại ô `Archive threshold`, chọn mốc `Older than 365 days`.
  4. Bấm nút `Estimate Size`. Hệ thống báo "1.2 TB of memories will be moved to S3".
  5. Bấm `Start Migration Job`.
- **Kết quả mong đợi**: Giao diện tự động điều hướng sang `Pipelines`, hiển thị một Job mới tên là `Cold_Storage_Migration_Job` đang chạy với thanh tiến trình.

### Flow 12: Cô lập Tenant bị xâm phạm (Kill Switch)
- **Tiền điều kiện**: Có cảnh báo hệ thống báo Tenant "Alpha" đang có dấu hiệu truy cập trái phép lượng lớn.
- **Các bước thực hiện**:
  1. Điều hướng tới `Governance` -> Tab `Tenant Management`.
  2. Tại ô Search của bảng DataTable, gõ `Alpha`.
  3. Nhấn trực tiếp vào nút vuông màu đỏ `Emergency Suspend` (Icon hình nút Dừng khẩn cấp) nằm ở cột Action.
  4. Gõ lý do: `Suspicious scraping activity`.
  5. Bấm `Execute Kill Switch`.
- **Kết quả mong đợi**: Status của Tenant Alpha lập tức đổi sang cục badge đỏ `Suspended`. Mọi truy vấn API từ Tenant này ngay lập tức trả về lỗi 403 Forbidden.

---

## 3. P3 - ML/AI Engineer (+3 Flows)

### Flow 13: Đấu nối mô hình Embedding tùy chỉnh
- **Tiền điều kiện**: Kỹ sư vừa host một mô hình `bge-m3` nội bộ trên port 8000.
- **Các bước thực hiện**:
  1. Vào `Settings` -> Tab `AI Models`.
  2. Tại bảng **Embedding Models**, bấm `Add Custom Endpoint`.
  3. Điền Tên: `Medical_BGE_M3`. Điền Base URL: `http://10.0.0.5:8000/v1`.
  4. Nhập API Key (nếu có) và Vector Dimension (vd: `1024`).
  5. Bấm `Test Connection`. Hệ thống báo "Ping Success 12ms". Bấm `Save`.
- **Kết quả mong đợi**: Mô hình xuất hiện trong danh sách Model. Khi sang màn `Context Debugger`, tại ô Dropdown Model, đã có thể chọn `Medical_BGE_M3`.

### Flow 14: Kiểm thử độ chính xác của trích xuất thực thể (NER Precision)
- **Tiền điều kiện**: Cần test thử Ontology "Medical_Diseases".
- **Các bước thực hiện**:
  1. Bấm vào `Graph Studio`.
  2. Click nút có icon Cây bút thần kỳ (Wand) `Manual Entity Test` trên thanh Toolbar.
  3. Một Textbox lớn hiện ra. Dán đoạn bệnh án: *"Bệnh nhân nam 45 tuổi, có tiền sử đái tháo đường type 2..."*
  4. Bấm `Extract & Preview`.
- **Kết quả mong đợi**: Các cụm từ "đái tháo đường type 2", "Bệnh nhân nam" được highlight màu vàng trong văn bản. Phía trên Canvas mờ ảo hiện ra các Node ảo (Preview Nodes) kết nối với nhau bằng viền đứt đoạn. 

### Flow 15: Phân tích tối ưu chi phí (Token vs Context Window)
- **Tiền điều kiện**: Cần báo cáo ngân sách Token LLM hàng tháng.
- **Các bước thực hiện**:
  1. Truy cập `Dashboard`.
  2. Ở góc trên cùng, có một Tab toggle đổi từ `System View` sang `Analytics View`.
  3. Tại biểu đồ `Token/Request Ratio`, chọn xem dữ liệu `Last 30 Days`.
  4. Rê chuột vào đường đồ thị màu tím để xem chi tiết: "Trung bình 4,500 context tokens / request". 
- **Kết quả mong đợi**: UX hiển thị Tooltip sắc nét với số liệu cụ thể. Kỹ sư dựa vào đó để điều chỉnh lại Max Context Top_K trong Config nhằm giảm tải.

---

## 4. P4 - Enterprise Architect (+3 Flows)

### Flow 16: Trích xuất Báo cáo Tuân thủ (SOC2 / HIPAA Audit Report)
- **Tiền điều kiện**: Cần cung cấp bằng chứng hệ thống lưu vết (Audit Trail) không bị chỉnh sửa.
- **Các bước thực hiện**:
  1. Vào `Governance`.
  2. Mở tab `Audit Explorer`.
  3. Bấm nút `Generate Compliance Report` ở góc trên bảng.
  4. Modal mở ra, chọn Standard: `HIPAA`.
  5. Date Range: `01/01/2026 - 31/03/2026`.
  6. Bấm `Export PDF`.
- **Kết quả mong đợi**: Thanh Toast thông báo "Generating report...", sau 10s tải xuống file `HIPAA_Audit_Q1.pdf`. File PDF chứa danh sách Logs được đóng dấu hash mã hóa để chống giả mạo.

### Flow 17: Cấu hình Phân quyền RBAC nội bộ
- **Tiền điều kiện**: Có một nhân sự mới cần cấp quyền vào Dashboard để xử lý lỗi nhưng cấm xem Data thật.
- **Các bước thực hiện**:
  1. Truy cập `Settings`.
  2. Chọn mục `Team`.
  3. Sang tab `Roles & Permissions`. Bấm `Create Role`.
  4. Tên Role: `L1_Support`.
  5. Trong lưới quyền (Permission Grid): 
     - Tick xanh: `View_Dashboard`, `View_Metrics`, `View_Error_Log_Stacktrace`.
     - Bỏ tick: `Read_Raw_Memory`, `Export_Graph`.
  6. Bấm `Save Role`.
- **Kết quả mong đợi**: Trở lại tab `Members`, bấm Invite User với email hỗ trợ viên và gán Role `L1_Support`. Giao diện của hỗ trợ viên khi đăng nhập sẽ tự động làm mờ các nội dung Memory nhạy cảm.

### Flow 18: Kích hoạt luật che mờ dữ liệu nhạy cảm (Data Masking/PII Redaction)
- **Tiền điều kiện**: Hệ thống đang bị lưu lọt số điện thoại của người dùng vào Vector DB.
- **Các bước thực hiện**:
  1. Điều hướng tới `Governance` -> Tab `OPA Policy Editor`.
  2. Trên Sidebar của Editor, chọn file `pii_masking.rego`.
  3. Bật Switch Toggle `Enable Module` ở góc phải trình soạn thảo.
  4. Viết bổ sung Regex: `masking.regex_replace(input, "\\d{4}-\\d{4}", "****-****")`.
  5. Bấm `Save`.
- **Kết quả mong đợi**: Khi sang `Context Debugger` và nhập thử text có chứa số điện thoại, Output hiển thị trong Memory Explorer của chuỗi hội thoại đó sẽ hoàn toàn hiện dấu hoa thị (***).

---

## 5. P7 - AI Power User (+2 Flows)

### Flow 19: Gộp thực thể thủ công (Manual Entity Merging)
- **Tiền điều kiện**: Graph sinh ra bị lặp Node do sai chính tả từ phía người dùng khi chat.
- **Các bước thực hiện**:
  1. Đăng nhập Console. Màn hình tự động filter User ID của người dùng.
  2. Mở `Graph Studio`.
  3. Trên Canvas, tìm 2 node "Javascript" và "JS".
  4. Giữ phím `Ctrl / Cmd` và Click chuột trái vào 2 Node để chọn đa luồng (Multi-select).
  5. Thanh Action Bar nổi lên. Bấm nút `Merge Entities` (Icon 2 mũi tên gộp làm 1).
  6. Chọn Node "Javascript" làm Master Node. Bấm `Confirm`.
- **Kết quả mong đợi**: Animation (hiệu ứng) hút Node "JS" vào trong "Javascript" xảy ra mượt mà. Các đường dây (Edges) cũ của JS được chuyển hết sang Javascript.

### Flow 20: Xuất kho lưu trữ cá nhân (Export Personal Vault)
- **Tiền điều kiện**: Người dùng muốn chuyển Data sang nền tảng AI khác.
- **Các bước thực hiện**:
  1. Bấm vào Avatar ở góc phải trên cùng (`TopNav`), chọn `My Profile`.
  2. Trong màn hình Profile Settings, cuộn xuống phần `Data Portability`.
  3. Nhấn nút xám `Download My Data Archive`.
  4. Một Modal hiện lên yêu cầu nhập Mật khẩu / 2FA Code để xác nhận chính chủ.
  5. Nhập mã 2FA, bấm `Verify & Download`.
- **Kết quả mong đợi**: Một file `My_AI_Memories.zip` được tải về chứa toàn bộ file Markdown dễ đọc.

---

## 6. P5 & P6 - Ecosystem Users (+2 Flows)

### Flow 21: Thu hồi API Key của IDE Plugin đã bị lộ (Revoke Key)
- **Tiền điều kiện**: Developer P5 dùng máy tính công ty cài Plugin, nay đã nghỉ việc.
- **Các bước thực hiện**:
  1. Bấm vào `API & SDK`.
  2. Tại bảng **API Keys Table**, tìm dòng Key có Label `Copilot_Plugin_Office_PC`.
  3. Click vào icon thùng rác (Delete/Revoke) ở cột Action cuối cùng.
  4. Modal cảnh báo hiện ra: "This action will immediately disconnect all devices using this key".
  5. Bấm `Revoke Key` màu đỏ.
- **Kết quả mong đợi**: Dòng Key biến mất (hoặc gạch ngang). Thiết bị cài IDE Plugin lập tức văng ra trạng thái Unauthenticated khi gửi Prompt tiếp theo.

### Flow 22: Theo dõi Webhook Delivery Failures (P6)
- **Tiền điều kiện**: Đối tác báo họ không nhận được event `node.created`.
- **Các bước thực hiện**:
  1. Vào `API & SDK` -> Tab `Webhooks`.
  2. Trong danh sách Webhooks, tìm Endpoint của đối tác. Ở cột Status đang hiện cục Badge màu đỏ `Failing`.
  3. Click vào Endpoint đó. Một Panel phụ trượt từ bên phải ra (`Sliding Drawer`).
  4. Tab `Delivery Logs` trong Panel hiện hàng loạt request mã `502 Bad Gateway`.
  5. Click vào một request bất kỳ để xem Raw Request Payload và Raw Response Headers.
- **Kết quả mong đợi**: Kỹ sư chụp ảnh màn hình Response Headers lỗi 502 này gửi cho phía đối tác để họ sửa cấu hình Firewall chặn IP của VNP Memory. Dưới đáy có nút `Resend Payload` để thử lại ngay sau khi fix xong.

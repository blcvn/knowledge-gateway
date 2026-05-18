# User Navigation Flow: P2 - Platform / DevOps Engineer

## Vai trò & Mục tiêu
Platform/DevOps Engineer (P2) sử dụng VNP Memory Console để đảm bảo toàn bộ hệ thống lưu trữ, pipeline ingest và các memory engines chạy ổn định, không nghẽn cổ chai và đúng giới hạn tài nguyên.

## Flow 1: Giám sát Sức khỏe Hệ thống (Daily Routine)
**Ngữ cảnh**: Kiểm tra định kỳ buổi sáng hoặc nhận được alert từ Slack.
1. **Truy cập `Dashboard Overview`**:
   - Xem ngay các KPI Cards: Active Sessions, Recall Latency.
   - Quan sát biểu đồ **Memory Flow** xem lượng Ingest/sec có sụt giảm bất thường không.
   - Check bảng **Engine Health Grid** xem có Engine nào (Zep, Graphiti, Neo4j) đang báo vàng (Warning) hoặc đỏ (Critical) do quá tải Queue không.
2. **Khám phá Hạ tầng (Infrastructure Health)**:
   - Truy cập màn hình **Infrastructure**.
   - Kiểm tra biểu đồ CPU/Memory xem có Node nào đang OOM (Out of Memory) không.

## Flow 2: Xử lý Lỗi Hệ thống (Troubleshooting)
**Ngữ cảnh**: Ứng dụng báo lỗi 500 khi lưu Memory.
1. **Truy cập `Observability & Error Explorer`**:
   - Chuyển sang Tab **Errors**.
   - Lọc theo mức độ `Fatal` hoặc `Error`.
   - Bấm vào lỗi mới nhất. Khung Stacktrace bên phải hiện ra.
   - Đọc log, copy Trace ID để đối chiếu với Kibana/Datadog (nếu có).
2. **Truy cập `Pipelines Monitor`**:
   - Xem các Job đang bị kẹt ở trạng thái `Failed`.
   - Bấm "Retry Job" (nếu hỗ trợ) để chạy lại pipeline bị đứt.

## Flow 3: Quản lý Khách thuê & Cấp quyền (Tenant Provisioning)
**Ngữ cảnh**: Có một team mới trong công ty cần tích hợp VNP Memory.
1. **Truy cập `API & SDK Manager`**:
   - Chọn "Generate New Key".
   - Cấu hình Rate Limits cho team mới (vd: 1000 requests/minute) ở mục **Rate Limit Settings**.
2. Chia sẻ API Key cho team Developer.

# Giao diện Infrastructure Health

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình sức khỏe hạ tầng (`infrastructure-health`).

## 1. Cấu trúc tổng quan
Màn hình cung cấp giao diện Dashboard kỹ thuật (DevOps/SRE Dashboard) với nhiều thẻ chỉ số và biểu đồ diện rộng.

### Khối 1: Service Status Grid
Lưới các thẻ (Cards) ở trên cùng đại diện cho các dịch vụ cốt lõi.
- Mỗi thẻ gồm: Tên dịch vụ (VD: `Memory Gateway`, `Neo4j DB`, `Qdrant Vector DB`, `Redis Cache`).
- Đèn báo trạng thái (Status Indicator): 
  - Một chấm tròn lớn nhấp nháy phát sáng: Xanh (`Up`), Vàng (`Degraded`), Đỏ (`Down`).
- Hiển thị text Uptime (vd: `99.98% uptime`).

### Khối 2: Resource Utilization Charts
Khu vực chiếm diện tích lớn, chứa các biểu đồ theo dõi tài nguyên phần cứng (Recharts).
- **CPU & Memory Usage**: Biểu đồ dạng đường (Line Chart) theo dõi mức % RAM và % CPU tiêu thụ của cluster trong 24h qua.
- **Network I/O**: Biểu đồ hiển thị lưu lượng In/Out (Rx/Tx) màu xanh lam và màu cam.
- Biểu đồ có hệ thống Tooltip hiện ra khi rê chuột để xem chi tiết thông số tại một thời điểm cố định.

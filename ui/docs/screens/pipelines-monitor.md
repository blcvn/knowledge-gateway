# Giao diện Pipelines Monitor

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Giám sát tiến trình dữ liệu (`pipelines-monitor`).

## 1. Cấu trúc tổng quan
Bố cục chia làm 2 phần theo chiều dọc: Nửa trên hiển thị đồ họa trực quan (Canvas), nửa dưới hiển thị dữ liệu bảng điều khiển (Bảng danh sách công việc).

### Khối 1: Nửa trên - Pipeline Stage Graph (React Flow)
- Giao diện tương tự Graph Studio nhưng tập trung vào một luồng tuyến tính (Linear flow) cố định nằm ngang.
- **Các Node**: Thể hiện các stage của đường ống xử lý: `Data Ingestion` -> `Text Chunking` -> `Embedding Generation` -> `KGS Sync` -> `Vector Storage`.
- **Trạng thái Node**: Mỗi node có một icon loading (spinner vòng xoay) hoặc biểu tượng check-mark xanh lá cây thể hiện trạng thái hoạt động. Dưới node hiện thông số Throughput (vd: 120 chunks/sec).

### Khối 2: Nửa dưới - Job Queue Table
- Một bảng dữ liệu theo dõi trạng thái các tiến trình (Jobs) hiện tại.
- Các cột hiển thị: ID Job, Tên File/Nguồn, Trạng thái (Pending, Processing, Completed, Failed), Thanh tiến trình (% Progress bar xanh dương).
- Danh sách này tự động chớp hiệu ứng và cập nhật mỗi vài giây (Real-time mockup).

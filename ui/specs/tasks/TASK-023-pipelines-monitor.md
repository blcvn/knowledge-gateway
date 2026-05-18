---
id: TASK-023
title: Implement Giao diện Pipelines Monitor
service: ui
type: task
status: done
source: docs/screens/pipelines-monitor.md
---

# TASK-023: Triển khai Giao diện Pipelines Monitor

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/screens/pipelines-monitor.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Giám sát tiến trình dữ liệu (`pipelines-monitor`)....

**## 1. Cấu trúc tổng quan**
### Khối 1: Nửa trên - Pipeline Stage Graph (React Flow)
- Giao diện tương tự Graph Studio nhưng tập trung vào một luồng tuyến tính (Linear flow) cố định nằm ngang.
- **Các Node**: Thể hiện các stage của đường ống xử lý: `Data Ingestion` -> `Text Chunking` -> `Embedding Generation` -> `KGS Sync` -> `Vector Storage`.
- **Trạng thái Node**: Mỗi node có một icon loading (spinner vòng xoay) hoặc biểu tượng check-mark xanh lá cây thể hiện trạng thái hoạt động. Dưới node hiện thông số Throughput (vd: 120 chunks/sec).
### Khối 2: Nửa dưới - Job Queue Table
- Một bảng dữ liệu theo dõi trạng thái các tiến trình (Jobs) hiện tại.
- Các cột hiển thị: ID Job, Tên File/Nguồn, Trạng thái (Pending, Processing, Completed, Failed), Thanh tiến trình (% Progress bar xanh dương).
- Danh sách này tự động chớp hiệu ứng và cập nhật mỗi vài giây (Real-time mockup).

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
- [Source Document](../../../docs/screens/pipelines-monitor.md)

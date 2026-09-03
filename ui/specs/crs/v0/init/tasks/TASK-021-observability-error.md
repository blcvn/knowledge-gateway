---
id: TASK-021
title: Implement Giao diện Observability & Error Explorer
service: ui
type: task
status: done
source: docs/screens/observability-error.md
---

# TASK-021: Triển khai Giao diện Observability & Error Explorer

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/screens/observability-error.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Quản trị lỗi & Giám sát (`observability-error`)....

**## 1. Cấu trúc tổng quan**
### Khối 1: Error Explorer Tab
- **Error DataTable**: 
- **Stacktrace Drawer (Panel trượt từ phải)**: 
### Khối 2: Metrics Dashboard Tab
- Hiển thị biểu đồ Line (Recharts) đo lượng Error Count (Số lượng lỗi nhảy lên/xuống theo thời gian) và API Latency.
### Khối 3: Trace Viewer Tab
- Hiển thị luồng Trace Request dạng dọc (Ví dụ: `Gateway (12ms)` -> `Engine (800ms)` -> `OpenAI (2100ms)`).
- Hiển thị băng thời gian (Gantt chart đơn giản) cho mỗi Request.

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
- [Source Document](../../../docs/screens/observability-error.md)

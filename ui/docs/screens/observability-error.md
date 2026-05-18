# Giao diện Observability & Error Explorer

Mô tả thiết kế giao diện chi tiết dự kiến cho màn hình Quản trị lỗi & Giám sát (`observability-error`).

## 1. Cấu trúc tổng quan
Màn hình chia thành 3 phần chính nằm trong Hệ thống Tab (Metrics, Traces, Errors). Trong đó Tab **Errors** là khu vực cốt lõi nhất.

### Khối 1: Error Explorer Tab
Bảng thống kê lỗi hệ thống chuyên dụng.
- **Error DataTable**: 
  - Cột hiển thị: Timestamp, Level (Fatal, Error, Warning), Nguồn gốc (UI, API, Agent), Message lỗi.
  - Các lỗi có Level Fatal sẽ có viền dọc bên lề màu đỏ sẫm (`border-l-4 border-red-600 bg-red-950/20`).
- **Stacktrace Drawer (Panel trượt từ phải)**: 
  - Hiển thị khi người dùng Click vào một dòng lỗi cụ thể.
  - Vùng hiển thị Stacktrace mô phỏng giao diện Terminal: Thẻ `<pre>` nền đen (`bg-slate-900`), text màu đỏ nhạt (`text-red-400`), font chữ `monospace` có thanh cuộn ngang.
  - Các nút hành động: "Mark as Resolved" (xóa đỏ) và "Create Ticket".

### Khối 2: Metrics Dashboard Tab
- Hiển thị biểu đồ Line (Recharts) đo lượng Error Count (Số lượng lỗi nhảy lên/xuống theo thời gian) và API Latency.

### Khối 3: Trace Viewer Tab
- Hiển thị luồng Trace Request dạng dọc (Ví dụ: `Gateway (12ms)` -> `Engine (800ms)` -> `OpenAI (2100ms)`).
- Hiển thị băng thời gian (Gantt chart đơn giản) cho mỗi Request.

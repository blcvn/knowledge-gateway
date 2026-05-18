---
id: TASK-031
title: Implement Triển khai Observability & Error Explorer Module (Quản trị lỗi tập trung)
service: ui
type: task
status: done
source: specs/features/FEAT-006-observability-error-explorer.md
---

# TASK-031: Triển khai Triển khai Observability & Error Explorer Module (Quản trị lỗi tập trung)

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/specs/features/FEAT-006-observability-error-explorer.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**## Mục Tiêu**
Cung cấp giao diện "Quản trị lỗi tập trung" (Error Explorer) để SRE và Developers có thể theo dõi, phân tích các ngoại lệ (exceptions) như crash UI, lỗi 500 từ API, lỗi timeout hay vi phạm policy. Đồng thời theo dõi Metrics và Traces cơ bản.

**## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)**
### 1. Route & Layout
- **Path**: `/app/observability`
- **Layout**: Sử dụng `Tabs` để chuyển đổi giữa 3 màn hình: Metrics, Trace Viewer, và Error Explorer.
### 2. TypeScript Interfaces (`src/types/observability.ts`)
### 3. Cấu trúc Component (`src/pages/observability/`)
### 4. Mock Data Khởi Tạo (`src/mock/observability.ts`)

**## Giao Diện & Styling (TailwindCSS)**
- Danh sách lỗi dùng màu highlight nếu level = fatal (VD: `border-l-4 border-red-600 bg-red-950/20`).
- Stacktrace view: Thẻ `<pre>` với `bg-slate-900 text-red-400 p-4 rounded-md overflow-x-auto text-sm`.

**## Definition of Done**
- [ ] Khởi tạo thư mục và chia file đúng chuẩn.
- [ ] Chạy mượt mà, không render lỗi trắng trang do dùng `recharts`.
- [ ] Linter và Types không báo lỗi.

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [ ] AC-1: Tab `Error Explorer` tải được bảng lỗi có ít nhất 3 cột chính: Timestamp, Source, Message, Level.
- [ ] AC-2: Khi nhấn vào một dòng lỗi (error row), một màn hình chi tiết hiện ra chứa `stackTrace` định dạng code.
- [ ] AC-3: Tab `Metrics` hiển thị ít nhất một biểu đồ Line mô tả số lượng lỗi theo thời gian.
- [ ] AC-4: UI cung cấp bộ lọc lỗi (Filter) theo trạng thái "Resolved" và "Unresolved".


### 💎 Enterprise & Product-Grade UI/UX Standards
- [x] **Premium Aesthetics**: Giao diện mang cảm giác cao cấp (premium). Tránh dùng màu sắc cơ bản. Ưu tiên dùng hệ màu HSL mượt mà, dark mode sâu sắc (deep dark), hiệu ứng gradient tinh tế và glassmorphism.
- [x] **Typography**: Sử dụng modern typography (Inter, Roboto, Outfit). Layout tuân thủ chặt chẽ spacing grid system, UI không bị chật chội hoặc lỏng lẻo.
- [x] **Dynamic & Responsive**: Tích hợp các micro-animations, hiệu ứng hover, focus states, và transition mượt mà giúp giao diện "sống động" và phản hồi cao.
- [x] **Enterprise Completeness**: Xử lý triệt để loading states, empty states, error boundaries, và accessible (a11y) đầy đủ.

## 4. Tài liệu tham khảo
- [Source Document](../../../specs/features/FEAT-006-observability-error-explorer.md)

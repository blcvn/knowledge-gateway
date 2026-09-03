---
id: TASK-002
title: Implement VNP Memory Console UI - Architecture
service: ui
type: task
status: done
source: docs/architecture.md
---

# TASK-002: Triển khai VNP Memory Console UI - Architecture

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/architecture.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**## 1. Cấu Trúc Tổng Quan**
- **Cognitive-first UX**: Tập trung vào việc quản trị, quan sát các hoạt động của AI agent.
- **Graph-native**: Mọi thực thể, trí nhớ (memory), quy trình (workflow) đều có thể được theo dõi dưới dạng đồ thị.
- **Multi-tenant by default**: Thiết kế bắt buộc có tenant context đi kèm trong mọi requests và view.

**## 2. Luồng Xử Lý Sự Kiện & Dữ Liệu Tập Trung (Data Flow & State Management)**
### 2.1 Server State (Dữ liệu từ Backend)
- **Công cụ**: `@tanstack/react-query` (React Query)
- **Mục đích**: Cache, đồng bộ, và quản lý các dữ liệu bất đồng bộ lấy từ backend.
- **Luồng hoạt động**:
### 2.2 Global/Client State (Trạng thái UI cục bộ)
- **Công cụ**: `zustand`
- **Mục đích**: Quản lý các trạng thái UI dùng chung toàn cục mà không liên quan trực tiếp đến CSDL server (VD: Trạng thái Theme (Dark/Light), Sidebar Open/Close, Selected Tenant ID, Global Filters).
- **Luồng hoạt động**:

**## 3. Tương Tác Với Server (Server Interaction Layer)**
### 3.1 HTTP REST API Client
- **Công cụ**: Native `fetch` API (thông qua Custom Fetch Wrapper)
- **Cấu hình (Fetch Interceptor Logic)**:
- **Thư mục cấu trúc**: Đặt tập trung tại `src/lib/api-client.ts` và các file service như `src/services/memory.service.ts`.
### 3.2 Real-time WebSocket / SSE
- **Mục đích**: Cập nhật Real-time cho Dashboard (Metrics, Active Agents) và Context Debugger (Trạng thái luồng chạy).
- **Kiến trúc luồng xử lý**:

**## 4. Xử Lý Ngoại Lệ Tập Trung (Centralized Exception Management)**
### 4.1 Tầng Giao Diện (React Error Boundary)
- Sử dụng **React Error Boundary** bao bọc toàn bộ ứng dụng (Global) và từng module riêng lẻ (Local - VD: bao bọc toàn bộ `Memory Explorer`).
- Khi một component gặp crash trong quá trình render, Error Boundary sẽ bắt lỗi và hiển thị một Fallback UI thân thiện ("Đã có lỗi xảy ra ở thành phần này") kèm nút "Thử lại", thay vì làm trắng toàn bộ màn hình.
### 4.2 Tầng Gọi API (Fetch & React Query)
- **API Wrapper Error Throwing**: Fetch wrapper tự động bắt các mã lỗi HTTP (4xx, 5xx) và map chúng thành một đối tượng `AppError` chuẩn hoá (chứa mã code, message).
- **React Query Error Handling**: Sử dụng thuộc tính `onError` cục bộ và cấu hình `QueryCache` / `MutationCache` global của React Query để hiển thị thông báo lỗi (Toast notifications) hoặc tự động retry đối với các lỗi network thoáng qua.
### 4.3 Tầng Báo Cáo Lỗi (Error Reporting)
- Mọi lỗi nghiêm trọng (crash UI, lỗi 500 từ Server) sẽ được ghi log lại qua một Error Tracking Service (VD: Sentry, hoặc hệ thống log nội bộ).
- Log sẽ tự động đính kèm `Tenant-ID` và thông tin User hiện tại để dễ dàng truy vết (traceability).

**## 5. Layout System**
1. **Left Sidebar**: Điều hướng chính (Overview, Memory Explorer, Graph Studio, Sessions, Governance, v.v.).
2. **Top Navigation**: Tenant selector, Environment switcher, Global search.
3. **Main Workspace**: Vùng hiển thị nội dung thích ứng.
4. **Right Context Panel**: Bảng thông tin ngữ cảnh động (chi tiết thực thể, metadata, v.v.). Thiết kế dạng trượt ra từ lề phải (Drawer/Sheet)....

**## 6. Các Modules Chính (MVP)**
- **Dashboard / Overview**: Trạng thái hệ thống tổng thể.
- **Memory Explorer**: Giao diện tìm kiếm. Xử lý sự kiện Filter nặng phụ thuộc vào việc Sync trạng thái Zustand và URL Query Params.
- **Graph Studio**: Khám phá knowledge graph trực quan. Do phải render Canvas đồ thị nặng, các sự kiện rê chuột / click node cần được chặn re-render tổng (dùng `useMemo`, `useCallback`).
- **Agent Context Debugger**: Debug quá trình xây dựng context cho AI.
- **Governance Center**: Quản trị tenant, chính sách OPA, Audit logs.

**## 7. Stack Quyết Định Kỹ Thuật (ADR)**
- **Routing**: `react-router-dom` (v6). Quản lý phân quyền và fetch dữ liệu thô ở tầng Loader.
- **State Management**: `Zustand` (Global), `React Query` (Server cache).
- **Graph Visualization**: `React Flow`.
- **Charting**: `Recharts` (Ưu tiên vì hỗ trợ style SSR và responsive tốt).
- **Styling**: `TailwindCSS` + `shadcn/ui` + `framer-motion` (hoạt cảnh).

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
- [Source Document](../../../docs/architecture.md)

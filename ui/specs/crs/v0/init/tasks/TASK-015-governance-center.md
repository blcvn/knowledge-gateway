---
id: TASK-015
title: Implement Giao diện Governance Center
service: ui
type: task
status: done
source: docs/screens/governance-center.md
---

# TASK-015: Triển khai Giao diện Governance Center

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/screens/governance-center.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Mô tả thiết kế giao diện chi tiết dự kiến cho phân hệ Quản trị & Tuân thủ (`governance-center`)....

**## 1. Cấu trúc tổng quan**
### Khối 1: Tab Tenant Management
- Một bảng dữ liệu (DataTable) lớn bao gồm các cột: Tên Tenant, Namespace, Quota Limit (Giới hạn tài nguyên), API Usage (Sử dụng API), Status (Active/Suspended).
- Các nút thao tác (Actions) ở cuối dòng: Edit, Suspend.
### Khối 2: Tab OPA Policy Editor
- Một khung soạn thảo mã nguồn (Code Editor / Textarea monospace) tối màu.
- Hiển thị syntax highlighting giả lập cho ngôn ngữ `Rego`.
- Bên phải (hoặc bên dưới) là các nút: **Format Code**, **Test Policy** và **Save**.
### Khối 3: Tab GDPR Forget Center
- Thanh tìm kiếm: "Nhập User ID hoặc Email để quét dữ liệu".
- Nút bấm chính: **"Erase User Data"** (Nút màu đỏ cảnh báo nguy hiểm `bg-red-600`).
- Khi bấm nút, một Dialog/Modal cảnh báo hiển thị "Cascade Deletion Preview" (Danh sách các vùng dữ liệu sẽ bị xóa) yêu cầu gõ lại tên User để xác nhận.
### Khối 4: Tab Audit Explorer
- Bảng hiển thị: Thời gian, Actor (Người/Bot thực hiện), Action (Hành động), Entity (Thực thể bị tác động), Policy Result.
- **Trực quan hóa**: Cột Policy Result sử dụng Badge màu đỏ cho "Denied" và màu xanh lá cho "Allowed".

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
- [Source Document](../../../docs/screens/governance-center.md)

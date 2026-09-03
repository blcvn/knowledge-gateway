---
id: TASK-016
title: Implement Giao diện Graph Studio
service: ui
type: task
status: done
source: docs/screens/graph-studio.md
---

# TASK-016: Triển khai Giao diện Graph Studio

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/docs/screens/graph-studio.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**General**
Mô tả chi tiết kiến trúc giao diện hiện tại của màn hình Graph Studio (dựa trên source code `GraphStudio.tsx`)....

**## 1. Cấu trúc tổng quan**
### Khối 1: Header
- Nằm trên cùng với tiêu đề **Graph Studio** và phụ đề "Visual knowledge graph exploration".
- Cạnh phải Header là cụm 3 nút thao tác nhanh: **Zoom In**, **Zoom Out**, và **Maximize** (Phóng to toàn màn hình).
### Khối 2: Graph Canvas (Vùng Tương Tác Cốt Lõi)
- **Background**: Màu tối sâu (`#0a0a0f`) kèm Pattern đường lưới (Grid line mờ).
- **Edges (Các cạnh)**: Các đường line kết nối giữa các Node với màu sắc khác nhau, độ mờ (opacity) 0.6.
- **Nodes (Thực thể)**: 
- **Nodes**: Số lượng đỉnh (vd: 1,247).
- **Edges**: Số lượng cạnh (vd: 3,821).
- **Clusters**: Số lượng cụm (vd: 24).
- Tiêu đề: **Entity Inspector**.
- Thông tin chính: Type, Ontology Class (Loại Schema).
- **Confidence**: Thanh tiến trình (Progress bar) trực quan màu xanh lam (ví dụ: 92%).
- Số Node liên kết.
- **Related Facts**: Danh sách các khối nhỏ (`bg-white/5`) hiển thị thông tin thực tế dạng text (VD: "Works on Project Alpha").
- Nút Play/Pause để giả lập mô phỏng chạy theo thời gian.
- Text hiển thị mốc bắt đầu (2026-01-01) và kết thúc (2026-05-13).
- Một thanh trượt (`input type="range"`) cho phép kéo tay để "duyệt" lịch sử tri thức (Temporal playback).

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
- [Source Document](../../../docs/screens/graph-studio.md)

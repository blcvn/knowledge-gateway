---
id: TASK-029
title: Implement Triển khai Governance Center Module
service: ui
type: task
status: done
source: specs/features/FEAT-004-governance-center.md
---

# TASK-029: Triển khai Triển khai Governance Center Module

## 1. Mục tiêu (Objective)
- Triển khai các yêu cầu và đặc tả kỹ thuật từ tài liệu: `ui/specs/features/FEAT-004-governance-center.md`
- Đảm bảo 100% tính năng và giao diện được mô tả trong tài liệu được áp dụng vào source code.

## 2. Phạm vi công việc (Scope)
Dưới đây là chi tiết công việc được trích xuất từ tài liệu gốc:

**## Mục Tiêu**
Cung cấp khu vực dành riêng cho việc quản trị, phân quyền (Tenants, Policies) và giám sát (Audit logs, GDPR Forget).

**## Thiết Kế Kỹ Thuật (Hướng Dẫn Triển Khai Cho AI)**
### 1. Route & Layout
- **Path**: `/app/governance`
- **Layout**: Sử dụng thành phần `Tabs` của shadcn/ui để tạo 4 sub-views:
### 2. TypeScript Interfaces (`src/types/governance.ts`)
### 3. Cấu trúc Component (`src/pages/governance/`)
### 4. Mock Data (`src/mock/governance.ts`)

**## Giao Diện & Styling (TailwindCSS)**
- GDPR Button: `bg-red-600 hover:bg-red-700 text-white`.
- Audit Log "Denied" row/badge: `bg-red-500/20 text-red-500`. "Allowed": `bg-green-500/20 text-green-500`.

**## Definition of Done**
- [ ] Cấu trúc code module hóa từng Tab riêng.
- [ ] UI không có lỗi linter/type.

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [ ] AC-1: Màn hình hiển thị Navbar có 4 Tab (Tenants, Policies, GDPR, Audit) và chuyển đổi mượt mà.
- [ ] AC-2: Tab Policy Editor phải render được đoạn code `.rego` mẫu bằng thẻ font-mono với syntax highlighting giả lập (màu text cơ bản).
- [ ] AC-3: Tab GDPR Forget Center: Khi nhấn "Erase User Data", bắt buộc phải hiển thị Dialog Xác nhận cảnh báo hành động bất khả nghịch.
- [ ] AC-4: Bảng Audit Logs phân biệt màu sắc rõ ràng ở cột `Policy Result` (Xanh/Đỏ).


### 💎 Enterprise & Product-Grade UI/UX Standards
- [x] **Premium Aesthetics**: Giao diện mang cảm giác cao cấp (premium). Tránh dùng màu sắc cơ bản. Ưu tiên dùng hệ màu HSL mượt mà, dark mode sâu sắc (deep dark), hiệu ứng gradient tinh tế và glassmorphism.
- [x] **Typography**: Sử dụng modern typography (Inter, Roboto, Outfit). Layout tuân thủ chặt chẽ spacing grid system, UI không bị chật chội hoặc lỏng lẻo.
- [x] **Dynamic & Responsive**: Tích hợp các micro-animations, hiệu ứng hover, focus states, và transition mượt mà giúp giao diện "sống động" và phản hồi cao.
- [x] **Enterprise Completeness**: Xử lý triệt để loading states, empty states, error boundaries, và accessible (a11y) đầy đủ.

## 4. Tài liệu tham khảo
- [Source Document](../../../specs/features/FEAT-004-governance-center.md)

---
id: TASK-042
title: Quản lý Trạng thái (State) & Data Fetching (Enterprise Grade)
service: ui
type: task
status: done
source: enterprise-requirements
---

# TASK-042: Quản lý Trạng thái (State) & Data Fetching (Enterprise Grade)

## 1. Mục tiêu (Objective)
Cấu hình kiến trúc quản lý dữ liệu hiệu quả cao, hỗ trợ bộ nhớ đệm (caching), đồng bộ hóa tự động và các thao tác cập nhật dữ liệu mượt mà, hạn chế tối đa các lỗi do không đồng bộ dữ liệu.

## 2. Phạm vi công việc (Scope)
- **Data Fetching Layer**: Tích hợp `@tanstack/react-query` hoặc SWR làm công cụ chính cho Server State. Thiết lập Global Config (staleTime, cacheTime, retry policy).
- **Client State Management**: Cấu hình `Zustand` (hoặc Redux Toolkit) để quản lý Global Client State (Ví dụ: Trạng thái đóng mở Sidebar, UI Theme, Active Workspace).
- **Optimistic Updates**: Triển khai Optimistic UI cho các thao tác CRUD (Create/Update/Delete) để người dùng có cảm giác ứng dụng phản hồi ngay lập tức, không có độ trễ mạng.
- **Data Synchronization**: Hỗ trợ Polling hoặc WebSockets/SSE để cập nhật các luồng dữ liệu thời gian thực (real-time) như trạng thái Pipeline hoặc System Health.

## 3. Tiêu chí nghiệm thu (Acceptance Criteria)
- [x] Dữ liệu từ API được cache thông minh; không gọi lại API thừa khi người dùng điều hướng qua lại giữa các tab (nếu dữ liệu chưa stale).
- [x] Các actions (như Delete hoặc Update) hiển thị kết quả ngay lập tức (Optimistic Update) và tự động rollback nếu API lỗi.
- [x] Client State (Zustand) được cấu trúc module hoá, dễ dàng scale.
- [x] Tích hợp React Query Devtools để hỗ trợ quá trình Debug.

## 4. Tài liệu tham khảo
- React Query & State Management Best Practices.

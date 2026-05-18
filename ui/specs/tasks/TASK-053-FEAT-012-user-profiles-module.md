---
id: TASK-053
title: Implement FEAT-012 User Profiles Module
service: ui
type: FEATURE
priority: P0
status: TODO
created: 2026-05-13
updated: 2026-05-13
linked_specs:
  - specs/features/FEAT-012-user-profiles-module.md
---

## Mục Tiêu
Xây dựng module quản lý User Profiles (giao diện cho Memobase engine) để trực quan hóa, cấu hình, và theo dõi profile của người dùng tại route `/app/profiles`.

## Phạm Vi Công Việc (Scope)
1. **Navigation**: Thêm entry "User Profiles" vào SidebarNavigation với biểu tượng User màu teal (`#14b8a6`).
2. **Routing & Container**: Tạo main container `UserProfiles.tsx` xử lý routing phụ (`/app/profiles/*`).
3. **Các Thành Phần Cốt Lõi**:
   - `ProfileExplorer.tsx`: Danh sách người dùng (tìm kiếm, duyệt qua các tenant).
   - `ProfileDetail.tsx`: Chế độ xem dạng cây (collapsible tree) của thông tin người dùng (topic → sub_topic → content).
   - `ProfileConfigEditor.tsx`: Giao diện thay đổi schema cấu hình profile và chế độ strict mode.
   - `BufferZoneMonitor.tsx`: Hiển thị real-time trạng thái token accumulation, thanh tiến trình buffer, lịch sử flush.
   - `EventTimeline.tsx`: Hiển thị dòng thời gian các sự kiện của người dùng, hỗ trợ tính năng search.
   - `ContextAssemblyPreview.tsx`: Preview đoạn prompt chuẩn bị được gửi đi cùng context.
4. **Data Integration**: Gắn kết UI component với các hook từ `profile.service.ts` (`useProfileList`, `useProfileDetail`, `useBufferStatus`, v.v.).

## Acceptance Criteria
- [ ] AC-1: Màn hình `/app/profiles` hiển thị đúng danh sách người dùng.
- [ ] AC-2: Cấu trúc thư mục ProfileDetail dạng node cây hoạt động mượt mà.
- [ ] AC-3: Buffer Zone Monitor phản ánh thay đổi dữ liệu chính xác qua hook tương ứng.
- [ ] AC-4: UI components tuân thủ các UX Patterns (Màu Teal cho Memobase, badge token).
- [ ] AC-5: Hỗ trợ Responsive layouts (Desktop & Tablet).
- [ ] AC-6: Có các state Loading / Error / Empty đầy đủ cho tất cả component.

## Definition of Done
- [ ] Hoàn thành implementation các components theo spec.
- [ ] Vượt qua Strict Type Checking và Linting.
- [ ] Storybook (nếu có) phản ánh chính xác các UI states.
- [ ] Unit test cơ bản cho các tương tác UI (collapsing tree, toggles).

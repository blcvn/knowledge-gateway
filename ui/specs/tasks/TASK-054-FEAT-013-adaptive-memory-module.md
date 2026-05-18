---
id: TASK-054
title: Implement FEAT-013 Adaptive Memory Module
service: ui
type: FEATURE
priority: P0
status: TODO
created: 2026-05-13
updated: 2026-05-13
linked_specs:
  - specs/features/FEAT-013-adaptive-memory-module.md
---

## Mục Tiêu
Xây dựng module quản lý Adaptive Memory (giao diện cho Supermemory engine) để quản lý bộ nhớ có tính tiến hóa, luật tự quên (auto-forget), xử lý mâu thuẫn (contradictions), và external connectors tại route `/app/adaptive`.

## Phạm Vi Công Việc (Scope)
1. **Navigation**: Thêm entry "Adaptive Memory" vào SidebarNavigation với biểu tượng Sparkle màu amber (`#f59e0b`).
2. **Routing & Container**: Tạo main container `AdaptiveMemory.tsx` cho các route phụ (`/app/adaptive/*`).
3. **Các Thành Phần Cốt Lõi**:
   - `MemoryVersionExplorer.tsx`: Giao diện duyệt version chain với liên kết parent→root, hỗ trợ Diff view giữa các phiên bản.
   - `AutoForgetRules.tsx`: Giao diện cấu hình luật Auto-Forget (duration input, strict mode, noise filtering) và hiển thị lịch sử xử lý contradiction.
   - `ExternalConnectors.tsx`: Bảng quản lý Connector (Google Drive, Notion, v.v.), trạng thái đồng bộ, các action controls.
   - `AdaptiveMemoryGraph.tsx`: Sơ đồ graph node thể hiện node static/dynamic, độ mờ suy giảm theo time-decay.
   - `AdaptiveAnalytics.tsx`: Biểu đồ thống kê tỷ lệ tạo/xoá/merge.
4. **Data Integration**: Sử dụng hooks từ `adaptive.service.ts` (`useAdaptiveMemories`, `useMemoryVersions`, `useConnectors`, v.v.).

## Acceptance Criteria
- [ ] AC-1: Màn hình `/app/adaptive` hiển thị đúng danh sách Adaptive Memories.
- [ ] AC-2: Giao diện Version Chain hiển thị đúng cây liên kết, badge `isLatest` hiển thị rõ.
- [ ] AC-3: Bảng External Connectors đầy đủ chức năng quản lý, phân biệt màu sắc theo status (green=connected, gray=disconnected).
- [ ] AC-4: Tính năng side-by-side Diff view cho hai memory versions hoạt động chính xác.
- [ ] AC-5: Hỗ trợ Responsive layouts và đầy đủ các Loading/Error states.
- [ ] AC-6: Màu chủ đạo của Adaptive Memory (Amber #f59e0b) được áp dụng đồng bộ trong module.

## Definition of Done
- [ ] Mã nguồn hoàn thiện, không lỗi hiển thị, tuân thủ spec.
- [ ] Vượt qua Strict Type Checking và Linting.
- [ ] Khả năng xử lý tốt khi không có dữ liệu (Empty State) từ API.
- [ ] Unit/Integration tests cho các hooks và tính năng chuyển đổi giao diện quan trọng.

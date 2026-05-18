---
id: TASK-052
title: Implement API Endpoint Mapping
service: ui
type: ENHANCEMENT
priority: P1
status: Done
created: 2026-05-13
updated: 2026-05-13
linked_docs:
  - docs/api-endpoint-mapping.md
---

## Mục Tiêu
Đảm bảo tất cả 44 API endpoints được liệt kê trong `docs/api-endpoint-mapping.md` được tích hợp đầy đủ vào các Service files và React Query hooks tương ứng của ứng dụng.

## Phạm Vi Công Việc (Scope)
1. **Service Layer Updates**: Cập nhật 12 service files (`dashboard.service.ts`, `memory.service.ts`, `graph.service.ts`, `profile.service.ts`, `adaptive.service.ts`, `cognee.service.ts`, `zep.service.ts`, `governance.service.ts`, v.v.) để khai báo chính xác các API endpoints.
2. **React Query Hooks**: Triển khai và kiểm tra khoảng hơn 40 custom hooks tương ứng (như `useMetrics`, `useMemorySearch`, `useProfileDetail`, `useConnectors`, v.v.).
3. **Gateway Namespace Alignment**: Đảm bảo các REST calls tuân thủ namespace mới của Gateway API (đặc biệt là các routes `/v1/console/*`).
4. **Testing**: Viết unit test cho các service methods và hooks. Đảm bảo hỗ trợ Mock Data Fallback (TASK-046) khi các endpoints chưa sẵn sàng.

## Acceptance Criteria
- [x] AC-1: Tất cả 44 endpoints trong tài liệu mapping đã có hàm gọi tương ứng trong thư mục `src/lib/services/`.
- [x] AC-2: Mỗi endpoint đều có một React Query hook tương ứng được định nghĩa trong `src/lib/hooks/`.
- [x] AC-3: Endpoint paths khớp chính xác với cấu trúc định tuyến API mới của Backend/Gateway.
- [x] AC-4: Khi network request thất bại, cơ chế fallback to mock data hoạt động chính xác.

## Definition of Done
- [x] Hoàn thành code cho tất cả services và hooks.
- [x] Vượt qua quá trình linting và type checking.
- [x] Không có TypeScript errors nào liên quan đến API payload/response types.
- [x] Đã thêm docs comment (JSDoc) cho các hooks phức tạp.

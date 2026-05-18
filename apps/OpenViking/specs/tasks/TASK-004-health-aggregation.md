---
id: TASK-004
title: Implement Health Aggregation
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P2
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu
Cung cấp một endpoint tổng hợp báo cáo trạng thái sức khoẻ của tất cả các services đã được embedded trong monolith (Health Aggregation).

## Scope
### In Scope
- Thêm một handler cho Monolith tại `/status` hoặc `/health`.
- Gom nhóm dữ liệu từ Gateway (cổng 8083) và 6 domain services (9091-9095, 9099) để trả về một JSON duy nhất.

### Out of Scope
- Viết lại logic check health của từng service. Các service con vốn đã có sẵn gRPC Health check hoặc HTTP check theo architecture reference (mục 8. Port Assignment).

## Thiết Kế Kỹ Thuật
- Tạo HTTP server riêng cho supervisor tại port 8000 (Monolith management port).
- Expose GET `/status`. Khi có request, handler sẽ gọi health endpoint hoặc check process status của 7 module con.
- Trả về JSON:
```json
{
  "status": "up",
  "gateway": "up",
  "services": {
    "ov-fs": "up",
    "ov-search": "up",
    "ov-session": "up",
    "ov-resource": "up",
    "ov-crypto": "up",
    "ov-admin": "up"
  }
}
```

## Acceptance Criteria
- [x] AC-1: API `GET /status` trả về thông tin trạng thái sức khoẻ tổng hợp.
- [x] AC-2: Thông tin thể hiện được trạng thái (up/down) của Gateway và từng domain service riêng biệt.
- [x] AC-3: Gọi health check đồng thời (concurrent) có timeout (VD: 3s) để không làm treo handler.

## Test Requirements
- Unit test cho health aggregation logic.
- Mock các HTTP health endpoints để kiểm tra logic tổng hợp.

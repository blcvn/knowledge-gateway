---
id: TASK-005
title: Engine Bootstrappers (Part 2)
service: apps-memory
status: Done
priority: P1
created: 2026-05-14
---

# TASK-005: Engine Bootstrappers (OpenViking, Zep, Supermemory)

## 1. Mục Tiêu
Hoàn tất việc tích hợp 21 service còn lại thuộc OpenViking, Zep, và Supermemory vào ứng dụng Monolithic.

## 2. Các Bước Thực Thi

Tạo các file bootstrap tại `apps/memory/internal/bootstrap/`:

1. **`openviking.go`**:
   - Wire 6 services: `ov-fs`, `ov-search`, `ov-session`, `ov-resource`, `ov-crypto`, `ov-admin`.
2. **`zep.go`**:
   - Wire 6 services: `zep-user`, `zep-thread`, `zep-memory`, `zep-graph`, `zep-search`, `zep-admin`.
3. **`supermemory.go`**:
   - Wire 9 services: `sm-document`, `sm-memory`, `sm-search`, `sm-profile`, `sm-connector`, `sm-mcp`, `sm-auth`, `sm-analytics`, `sm-project`.

Cập nhật `main.go` để gọi 3 hàm bootstrap này.

## 3. Wiring NATS Events
Chú ý các events quan trọng cần wire trực tiếp bằng handler in-process (sử dụng Embedded NATS context):
- `ov.resource.ingested`
- `zep.memory.messages.ingested`
- `sm.document.created`

## 4. Acceptance Criteria
- [ ] Tất cả 35 domain services được đăng ký thành công vào `GRPCBus`.
- [ ] Gateway `InProcessRegistry` có khả năng resolve endpoint cho toàn bộ 35 services.
- [ ] Memory footprint của empty binary không vượt quá 300MB khi start.
